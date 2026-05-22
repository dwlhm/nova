package conformance

import (
	"fmt"

	"github.com/dwlhm/nova/internal/build"
	"github.com/dwlhm/nova/internal/scheduler"
)

type ExpectedScheduler struct {
	Steps []SchedulerStep `json:"steps"`
	Trace Trace           `json:"trace"`
}

type SchedulerStep struct {
	Enqueue *EnqueueStep `json:"enqueue,omitempty"`
}

type EnqueueStep struct {
	Source  string `json:"source"`
	Event   string `json:"event"`
	Payload any    `json:"payload,omitempty"`
}

func runSchedulerFixture(plan build.BuildPlan, sources []build.SourceFile, expected *ExpectedScheduler) ([]Diagnostic, Trace, bool) {
	if expected == nil {
		return nil, Trace{}, true
	}

	runtime, diagnostics := buildSchedulerRuntime(plan, sources)
	if len(diagnostics) > 0 {
		return diagnostics, Trace{}, false
	}

	for index, step := range expected.Steps {
		if step.Enqueue == nil {
			return []Diagnostic{fixtureDiagnostic("NVA-CONFORMANCE-031", fmt.Sprintf("scheduler step %d is empty", index))}, Trace{}, false
		}
		var err error
		runtime, _, err = scheduler.Enqueue(
			runtime,
			scheduler.CapabilityRef(step.Enqueue.Source),
			scheduler.SchedulerEvent(step.Enqueue.Event),
			step.Enqueue.Payload,
		)
		if err != nil {
			return []Diagnostic{fixtureDiagnostic("NVA-CONFORMANCE-032", fmt.Sprintf("scheduler step %d enqueue %s: %s", index, step.Enqueue.Event, err.Error()))}, Trace{}, false
		}
	}

	_, results := scheduler.Drain(runtime)
	actual := TraceSchedulerResults(results)
	if !traceSpecified(expected.Trace) {
		return nil, actual, true
	}
	return CompareTrace(expected.Trace, actual), actual, true
}

func traceSpecified(trace Trace) bool {
	return len(trace.Events) > 0 ||
		len(trace.Commits) > 0 ||
		len(trace.LifecycleCalls) > 0 ||
		len(trace.ExternalCalls) > 0 ||
		len(trace.Errors) > 0
}
