package conformance

import (
	"fmt"

	"github.com/dwlhm/nova/internal/core/scheduler"
	"github.com/dwlhm/nova/internal/core/security"
	"github.com/dwlhm/nova/internal/project"
	"github.com/dwlhm/nova/internal/provider/build"
)

type ExpectedScheduler struct {
	Steps              []SchedulerStep       `json:"steps"`
	CompleteExternals  bool                  `json:"completeExternals,omitempty"`
	ExternalStub       string                `json:"externalStub,omitempty"`
	RuntimePermissions []security.Permission `json:"runtimePermissions,omitempty"`
	Trace              Trace                 `json:"trace"`
}

type SchedulerStep struct {
	Enqueue   *EnqueueStep   `json:"enqueue,omitempty"`
	Lifecycle *LifecycleStep `json:"lifecycle,omitempty"`
}

type LifecycleStep struct {
	Phase  string `json:"phase"`
	Source string `json:"source,omitempty"`
}

type EnqueueStep struct {
	Source  string `json:"source"`
	Event   string `json:"event"`
	Payload any    `json:"payload,omitempty"`
}

func runSchedulerFixture(plan build.BuildPlan, sources []build.SourceFile, projectPermissions project.PermissionMap, expected *ExpectedScheduler) ([]Diagnostic, Trace, bool) {
	if expected == nil {
		return nil, Trace{}, true
	}

	runtime, diagnostics := buildSchedulerRuntime(plan, sources)
	if len(diagnostics) > 0 {
		return diagnostics, Trace{}, false
	}

	actual := Trace{}
	stubMode, stubErr := parseExternalStubMode(expected.ExternalStub)
	if expected.CompleteExternals && stubErr != nil {
		return []Diagnostic{fixtureDiagnostic("NVA-CONFORMANCE-034", stubErr.Error())}, Trace{}, false
	}
	for index, step := range expected.Steps {
		if step.Lifecycle != nil {
			source := scheduler.CapabilityRef(step.Lifecycle.Source)
			if source == "" {
				source = "runtime"
			}
			runtime, _ = scheduler.RunLifecycle(runtime, scheduler.LifecyclePhase(step.Lifecycle.Phase), source)
			var stepTrace Trace
			runtime, stepTrace = drainSchedulerWithOptionalExternals(runtime, plan, projectPermissions, expected.RuntimePermissions, expected.CompleteExternals, stubMode)
			actual = mergeTrace(actual, stepTrace)
			continue
		}
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

	var stepTrace Trace
	runtime, stepTrace = drainSchedulerWithOptionalExternals(runtime, plan, projectPermissions, expected.RuntimePermissions, expected.CompleteExternals, stubMode)
	actual = mergeTrace(actual, stepTrace)
	if !traceSpecified(expected.Trace) {
		return nil, actual, true
	}
	return CompareTrace(expected.Trace, actual), actual, true
}

func mergeTrace(left Trace, right Trace) Trace {
	return Trace{
		Events:         append(left.Events, right.Events...),
		Commits:        append(left.Commits, right.Commits...),
		LifecycleCalls: append(left.LifecycleCalls, right.LifecycleCalls...),
		ExternalCalls:  append(left.ExternalCalls, right.ExternalCalls...),
		Errors:         append(left.Errors, right.Errors...),
	}
}

func traceSpecified(trace Trace) bool {
	return len(trace.Events) > 0 ||
		len(trace.Commits) > 0 ||
		len(trace.LifecycleCalls) > 0 ||
		len(trace.ExternalCalls) > 0 ||
		len(trace.Errors) > 0
}
