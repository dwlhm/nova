package conformance

import (
	"fmt"
	"reflect"

	"github.com/dwlhm/nova/internal/diagnostic"
	"github.com/dwlhm/nova/internal/scheduler"
)

type EventTrace struct {
	Sequence scheduler.LogicalSequence
	Source   scheduler.CapabilityRef
	Name     scheduler.SchedulerEvent
	Payload  scheduler.DataValue
}

type LifecycleTrace struct {
	Owner scheduler.CapabilityRef
	Phase scheduler.LifecyclePhase
	Event scheduler.SchedulerEvent
	Error string
}

type ExternalCallTrace struct {
	Source     scheduler.CapabilityRef
	Capability string
	Operation  string
	Input      map[string]scheduler.DataValue
	OutputType string
	OnSuccess  scheduler.SchedulerEvent
	OnFailure  scheduler.SchedulerEvent
}

type ErrorTrace struct {
	Source  scheduler.CapabilityRef
	Phase   scheduler.SchedulerPhase
	Event   scheduler.SchedulerEvent
	State   string
	Message string
}

type Trace struct {
	Events         []EventTrace
	Commits        []scheduler.StateCommit
	LifecycleCalls []LifecycleTrace
	ExternalCalls  []ExternalCallTrace
	Errors         []ErrorTrace
}

type Diagnostic = diagnostic.Diagnostic

func TraceSchedulerResults(results []scheduler.StepResult) Trace {
	trace := Trace{}
	for _, result := range results {
		trace.Events = append(trace.Events, EventTrace{
			Sequence: result.Event.Sequence,
			Source:   result.Event.Source,
			Name:     result.Event.Name,
			Payload:  result.Event.Payload,
		})
		trace.Commits = append(trace.Commits, result.Commit)
		for _, lifecycle := range result.LifecycleInvocations {
			errorMessage := ""
			if lifecycle.Error != nil {
				errorMessage = lifecycle.Error.Message
			}
			trace.LifecycleCalls = append(trace.LifecycleCalls, LifecycleTrace{
				Owner: lifecycle.Owner,
				Phase: lifecycle.Phase,
				Event: lifecycle.Event,
				Error: errorMessage,
			})
		}
		for _, external := range result.ExternalOperations {
			trace.ExternalCalls = append(trace.ExternalCalls, ExternalCallTrace{
				Source:     external.Source,
				Capability: external.Capability,
				Operation:  external.Operation,
				Input:      cloneInput(external.Input),
				OutputType: external.OutputType,
				OnSuccess:  external.OnSuccess,
				OnFailure:  external.OnFailure,
			})
		}
		for _, err := range result.Errors {
			trace.Errors = append(trace.Errors, errorTrace(err))
		}
	}
	return trace
}

func CompareTrace(expected Trace, actual Trace) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	if !reflect.DeepEqual(expected.Events, actual.Events) {
		diagnostics = append(diagnostics, mismatch("NVA-CONFORMANCE-001", "scheduler events", expected.Events, actual.Events))
	}
	if !reflect.DeepEqual(expected.Commits, actual.Commits) {
		diagnostics = append(diagnostics, mismatch("NVA-CONFORMANCE-002", "state commits", expected.Commits, actual.Commits))
	}
	if !reflect.DeepEqual(expected.LifecycleCalls, actual.LifecycleCalls) {
		diagnostics = append(diagnostics, mismatch("NVA-CONFORMANCE-003", "lifecycle calls", expected.LifecycleCalls, actual.LifecycleCalls))
	}
	if !reflect.DeepEqual(expected.ExternalCalls, actual.ExternalCalls) {
		diagnostics = append(diagnostics, mismatch("NVA-CONFORMANCE-004", "external calls", expected.ExternalCalls, actual.ExternalCalls))
	}
	if !reflect.DeepEqual(expected.Errors, actual.Errors) {
		diagnostics = append(diagnostics, mismatch("NVA-CONFORMANCE-005", "runtime errors", expected.Errors, actual.Errors))
	}
	return diagnostic.StableSort(diagnostics)
}

func mismatch(code string, section string, expected any, actual any) Diagnostic {
	return Diagnostic{
		Code:     code,
		Severity: diagnostic.SeverityError,
		Message:  fmt.Sprintf("conformance %s mismatch: expected %+v, actual %+v", section, expected, actual),
	}
}

func errorTrace(err scheduler.SchedulerError) ErrorTrace {
	state := ""
	if err.State != nil {
		state = fmt.Sprintf("%s.%s", err.State.Owner, err.State.Name)
	}
	return ErrorTrace{
		Source:  err.Source,
		Phase:   err.Phase,
		Event:   err.Event,
		State:   state,
		Message: err.Message,
	}
}

func cloneInput(input map[string]scheduler.DataValue) map[string]scheduler.DataValue {
	if input == nil {
		return nil
	}
	out := make(map[string]scheduler.DataValue, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
