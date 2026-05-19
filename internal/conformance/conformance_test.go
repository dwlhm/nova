package conformance

import (
	"testing"

	"github.com/dwlhm/nova/internal/scheduler"
)

func TestTraceSchedulerResultsCapturesEventsCommitsAndExternalCalls(t *testing.T) {
	key := scheduler.Key("Counter", "count")
	runtime := scheduler.NewRuntime(
		[]scheduler.StateCell{
			scheduler.NewStateCell("Counter", "count", "number", 0, scheduler.On("@increment", func(snapshot scheduler.Snapshot, event scheduler.EventEnvelope) (scheduler.DataValue, error) {
				return snapshot.MustValue(key).(int) + 1, nil
			})),
		},
		[]scheduler.LifecycleHandler{
			scheduler.After("Counter", "@increment", func(ctx scheduler.LifecycleContext) (scheduler.LifecycleOutput, error) {
				return scheduler.LifecycleOutput{
					External: []scheduler.ExternalOperationRequest{
						scheduler.ExternalOperation("Counter", "storage", "set", map[string]scheduler.DataValue{"value": ctx.Snapshot.MustValue(key)}, "void", "@stored", "@store_failed"),
					},
				}, nil
			}),
		},
	)
	runtime, _, _ = scheduler.Enqueue(runtime, "host", "@increment", nil)
	_, results := scheduler.Drain(runtime)

	trace := TraceSchedulerResults(results)

	if len(trace.Events) != 1 || trace.Events[0].Name != "@increment" {
		t.Fatalf("events trace = %+v", trace.Events)
	}
	if len(trace.Commits) != 1 || len(trace.Commits[0].Changes) != 1 {
		t.Fatalf("commits trace = %+v", trace.Commits)
	}
	if len(trace.ExternalCalls) != 1 || trace.ExternalCalls[0].Operation != "set" {
		t.Fatalf("external trace = %+v", trace.ExternalCalls)
	}
}

func TestCompareTraceReportsStableDiagnostics(t *testing.T) {
	expected := Trace{Events: []EventTrace{{Sequence: 1, Source: "host", Name: "@increment"}}}
	actual := Trace{Events: []EventTrace{{Sequence: 1, Source: "host", Name: "@decrement"}}}

	diagnostics := CompareTrace(expected, actual)

	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics = %+v, want one mismatch", diagnostics)
	}
	if diagnostics[0].Code != "NVA-CONFORMANCE-001" {
		t.Fatalf("diagnostic code = %s, want NVA-CONFORMANCE-001", diagnostics[0].Code)
	}
}
