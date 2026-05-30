package app

import (
	"testing"

	"github.com/dwlhm/nova/internal/scheduler"
)

func TestCoalesceLifecycleEventsPreservesLogicalOrder(t *testing.T) {
	events := []LifecycleEvent{
		{Name: EventAppStarted},
		{Name: EventAppResumed},
		{Name: EventAppResumed},
		{Name: EventAppPaused},
		{Name: EventAppPaused},
		{Name: EventAppResumed},
	}

	coalesced := CoalesceLifecycleEvents(events)

	want := []scheduler.SchedulerEvent{EventAppStarted, EventAppResumed, EventAppPaused, EventAppResumed}
	if len(coalesced) != len(want) {
		t.Fatalf("coalesced events = %+v, want %d events", coalesced, len(want))
	}
	for i, event := range coalesced {
		if event.Name != want[i] {
			t.Fatalf("event %d = %s, want %s", i, event.Name, want[i])
		}
	}
}

func TestEnqueueLifecycleEventsRequiresActiveContractAndNovaPayload(t *testing.T) {
	runtime := scheduler.NewRuntime(nil, nil)
	active := map[scheduler.SchedulerEvent]bool{EventAppStarted: true}

	next, result := EnqueueLifecycleEvents(runtime, scheduler.CapabilityRef("@nova/app"), active, []LifecycleEvent{
		{Name: EventAppStarted},
		{Name: EventAppPaused},
		{Name: EventAppRestored, Payload: make(chan int)},
	})

	if len(result.Enqueued) != 1 || result.Enqueued[0].Name != EventAppStarted {
		t.Fatalf("enqueued = %+v, want only active @app_started", result.Enqueued)
	}
	if len(next.Queue()) != 1 {
		t.Fatalf("queue length = %d, want 1", len(next.Queue()))
	}
	if len(result.Diagnostics) != 2 {
		t.Fatalf("diagnostics = %+v, want inactive event and non-serializable payload diagnostics", result.Diagnostics)
	}
}

func TestApplyNavigationUsesSerializableRouteDataAsSourceOfTruth(t *testing.T) {
	stack := NewNavigationStack(Route{Path: "/"})

	stack, event, err := ApplyNavigation(stack, NavigationAction{
		Kind:  NavigatePush,
		Route: &Route{Path: "/profile", Query: map[string]scheduler.DataValue{"tab": "posts"}},
	})
	if err != nil {
		t.Fatalf("push navigation failed: %v", err)
	}
	if stack.Current.Path != "/profile" || len(stack.Entries) != 2 {
		t.Fatalf("stack after push = %+v", stack)
	}
	if event.Name != EventRouteChanged {
		t.Fatalf("event = %+v, want route_changed", event)
	}
	if _, _, err := scheduler.Enqueue(scheduler.NewRuntime(nil, nil), "@nova/navigation", event.Name, event.Payload); err != nil {
		t.Fatalf("route payload should be serializable Nova data: %v", err)
	}

	stack, _, err = ApplyNavigation(stack, NavigationAction{
		Kind:  NavigateReplace,
		Route: &Route{Path: "/settings"},
	})
	if err != nil {
		t.Fatalf("replace navigation failed: %v", err)
	}
	if stack.Current.Path != "/settings" || len(stack.Entries) != 2 {
		t.Fatalf("stack after replace = %+v", stack)
	}

	stack, _, err = ApplyNavigation(stack, NavigationAction{Kind: NavigateBack})
	if err != nil {
		t.Fatalf("back navigation failed: %v", err)
	}
	if stack.Current.Path != "/" || len(stack.Entries) != 1 {
		t.Fatalf("stack after back = %+v", stack)
	}
}

func TestNavigationRejectsInvalidRouteBeforeAdapterReconciliation(t *testing.T) {
	stack := NewNavigationStack(Route{Path: "/"})
	_, _, err := ApplyNavigation(stack, NavigationAction{
		Kind:  NavigatePush,
		Route: &Route{Path: "relative", Params: map[string]scheduler.DataValue{"native": make(chan int)}},
	})
	if err == nil {
		t.Fatalf("expected invalid route diagnostic before platform reconciliation")
	}
}
