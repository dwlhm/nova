package app

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/dwlhm/nova/internal/diagnostic"
	"github.com/dwlhm/nova/internal/effect"
	"github.com/dwlhm/nova/internal/scheduler"
	novatypes "github.com/dwlhm/nova/internal/types"
	"github.com/dwlhm/nova/internal/view"
)

const (
	EventAppStarted  scheduler.SchedulerEvent = "@app_started"
	EventAppResumed  scheduler.SchedulerEvent = "@app_resumed"
	EventAppPaused   scheduler.SchedulerEvent = "@app_paused"
	EventAppStopped  scheduler.SchedulerEvent = "@app_stopped"
	EventAppRestored scheduler.SchedulerEvent = "@app_restored"

	EventRouteChanged     scheduler.SchedulerEvent = "@route_changed"
	EventNavigate         scheduler.SchedulerEvent = "@navigate"
	EventNavigationFailed scheduler.SchedulerEvent = "@navigation_failed"
)

type InstanceID string
type TargetID string

type HostPort interface {
	Start(context.Context, AppInstance) error
	Stop(context.Context, AppInstance) error
}

type AppInstance struct {
	ID        InstanceID
	Root      scheduler.CapabilityRef
	Target    TargetID
	Scheduler scheduler.Runtime
	Renderer  view.RendererPort
	Host      HostPort
	External  effect.ExternalOperationPort
}

type LifecycleEvent struct {
	Name    scheduler.SchedulerEvent
	Payload scheduler.DataValue
}

type LifecycleEnqueueResult struct {
	Enqueued    []scheduler.EventEnvelope
	Diagnostics []diagnostic.Diagnostic
}

type Diagnostic = diagnostic.Diagnostic

func CoalesceLifecycleEvents(events []LifecycleEvent) []LifecycleEvent {
	out := make([]LifecycleEvent, 0, len(events))
	for _, event := range events {
		if len(out) > 0 && out[len(out)-1].Name == event.Name && reflect.DeepEqual(out[len(out)-1].Payload, event.Payload) {
			continue
		}
		out = append(out, event)
	}
	return out
}

func EnqueueLifecycleEvents(runtime scheduler.Runtime, source scheduler.CapabilityRef, active map[scheduler.SchedulerEvent]bool, events []LifecycleEvent) (scheduler.Runtime, LifecycleEnqueueResult) {
	next := runtime
	result := LifecycleEnqueueResult{}
	for _, event := range CoalesceLifecycleEvents(events) {
		if !novatypes.IsSerializable(event.Payload) {
			result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
				Code:     "NVA-EVENT-019",
				Severity: diagnostic.SeverityError,
				Message:  fmt.Sprintf("app lifecycle event %s payload must be Nova data", event.Name),
			})
			continue
		}
		if !active[event.Name] {
			result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
				Code:     "NVA-EVENT-018",
				Severity: diagnostic.SeverityWarning,
				Message:  fmt.Sprintf("app lifecycle event %s is not active for this capability", event.Name),
			})
			continue
		}
		var envelope scheduler.EventEnvelope
		var err error
		next, envelope, err = scheduler.Enqueue(next, source, event.Name, event.Payload)
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
				Code:     "NVA-RUNTIME-018",
				Severity: diagnostic.SeverityError,
				Message:  err.Error(),
			})
			continue
		}
		result.Enqueued = append(result.Enqueued, envelope)
	}
	return next, result
}

type Route struct {
	Path     string
	Params   map[string]scheduler.DataValue
	Query    map[string]scheduler.DataValue
	Fragment string
}

func (r Route) Data() map[string]scheduler.DataValue {
	data := map[string]scheduler.DataValue{"path": r.Path}
	if len(r.Params) > 0 {
		data["params"] = cloneDataMap(r.Params)
	}
	if len(r.Query) > 0 {
		data["query"] = cloneDataMap(r.Query)
	}
	if r.Fragment != "" {
		data["fragment"] = r.Fragment
	}
	return data
}

func ValidateRoute(route Route) error {
	if route.Path == "" || !strings.HasPrefix(route.Path, "/") {
		return fmt.Errorf("route path must start with /")
	}
	if !novatypes.IsSerializable(route.Data()) {
		return fmt.Errorf("route payload must be serializable Nova data")
	}
	return nil
}

type NavigationStack struct {
	Current Route
	Entries []Route
}

func NewNavigationStack(route Route) NavigationStack {
	return NavigationStack{Current: route, Entries: []Route{route}}
}

type NavigationActionKind string

const (
	NavigatePush    NavigationActionKind = "push"
	NavigateReplace NavigationActionKind = "replace"
	NavigateBack    NavigationActionKind = "back"
)

type NavigationAction struct {
	Kind  NavigationActionKind
	Route *Route
}

func ApplyNavigation(stack NavigationStack, action NavigationAction) (NavigationStack, scheduler.EventToEmit, error) {
	switch action.Kind {
	case NavigatePush:
		if action.Route == nil {
			return stack, scheduler.EventToEmit{}, fmt.Errorf("push navigation requires route")
		}
		return pushRoute(stack, *action.Route)
	case NavigateReplace:
		if action.Route == nil {
			return stack, scheduler.EventToEmit{}, fmt.Errorf("replace navigation requires route")
		}
		return replaceRoute(stack, *action.Route)
	case NavigateBack:
		return popRoute(stack)
	default:
		return stack, scheduler.EventToEmit{}, fmt.Errorf("unsupported navigation action %s", action.Kind)
	}
}

func pushRoute(stack NavigationStack, route Route) (NavigationStack, scheduler.EventToEmit, error) {
	if err := ValidateRoute(route); err != nil {
		return stack, scheduler.EventToEmit{}, err
	}
	next := NavigationStack{
		Current: route,
		Entries: append(cloneRoutes(stack.Entries), route),
	}
	return next, routeChanged(route), nil
}

func replaceRoute(stack NavigationStack, route Route) (NavigationStack, scheduler.EventToEmit, error) {
	if err := ValidateRoute(route); err != nil {
		return stack, scheduler.EventToEmit{}, err
	}
	entries := cloneRoutes(stack.Entries)
	if len(entries) == 0 {
		entries = []Route{route}
	} else {
		entries[len(entries)-1] = route
	}
	return NavigationStack{Current: route, Entries: entries}, routeChanged(route), nil
}

func popRoute(stack NavigationStack) (NavigationStack, scheduler.EventToEmit, error) {
	if len(stack.Entries) <= 1 {
		return stack, scheduler.EventToEmit{}, fmt.Errorf("back navigation rejected by route state")
	}
	entries := cloneRoutes(stack.Entries[:len(stack.Entries)-1])
	route := entries[len(entries)-1]
	return NavigationStack{Current: route, Entries: entries}, routeChanged(route), nil
}

func routeChanged(route Route) scheduler.EventToEmit {
	return scheduler.Emit("@nova/navigation", EventRouteChanged, route.Data())
}

func cloneRoutes(routes []Route) []Route {
	out := make([]Route, len(routes))
	for i, route := range routes {
		out[i] = route
		out[i].Params = cloneDataMap(route.Params)
		out[i].Query = cloneDataMap(route.Query)
	}
	return out
}

func cloneDataMap(input map[string]scheduler.DataValue) map[string]scheduler.DataValue {
	if input == nil {
		return nil
	}
	out := make(map[string]scheduler.DataValue, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
