package conformance

import (
	"fmt"
	"reflect"

	"github.com/dwlhm/nova/internal/core/diagnostic"
	"github.com/dwlhm/nova/internal/core/scheduler"
)

type EventTrace struct {
	Sequence scheduler.LogicalSequence `json:"sequence"`
	Source   scheduler.CapabilityRef   `json:"source"`
	Name     scheduler.SchedulerEvent  `json:"name"`
	Payload  scheduler.DataValue       `json:"payload,omitempty"`
}

type LifecycleTrace struct {
	Owner scheduler.CapabilityRef  `json:"owner"`
	Phase scheduler.LifecyclePhase `json:"phase"`
	Event scheduler.SchedulerEvent `json:"event"`
	Error string                   `json:"error,omitempty"`
}

type ExternalCallTrace struct {
	Source     scheduler.CapabilityRef        `json:"source"`
	Capability string                         `json:"capability"`
	Operation  string                         `json:"operation"`
	Input      map[string]scheduler.DataValue `json:"input,omitempty"`
	OutputType string                         `json:"outputType"`
	OnSuccess  scheduler.SchedulerEvent       `json:"onSuccess"`
	OnFailure  scheduler.SchedulerEvent       `json:"onFailure"`
}

type ErrorTrace struct {
	Source  scheduler.CapabilityRef  `json:"source"`
	Phase   scheduler.SchedulerPhase `json:"phase"`
	Event   scheduler.SchedulerEvent `json:"event"`
	State   string                   `json:"state,omitempty"`
	Message string                   `json:"message"`
}

type Trace struct {
	Events         []EventTrace        `json:"events,omitempty"`
	Commits        []CommitTrace       `json:"commits,omitempty"`
	LifecycleCalls []LifecycleTrace    `json:"lifecycleCalls,omitempty"`
	ExternalCalls  []ExternalCallTrace `json:"externalCalls,omitempty"`
	Errors         []ErrorTrace        `json:"errors,omitempty"`
}

type CommitTrace struct {
	Sequence      scheduler.LogicalSequence `json:"sequence"`
	Event         scheduler.SchedulerEvent  `json:"event"`
	Committed     bool                      `json:"committed"`
	Changes       []StateChangeTrace        `json:"changes,omitempty"`
	Invalidations []StateKeyTrace           `json:"invalidations,omitempty"`
}

type StateChangeTrace struct {
	Key    StateKeyTrace       `json:"key"`
	Before scheduler.DataValue `json:"before,omitempty"`
	After  scheduler.DataValue `json:"after,omitempty"`
}

type StateKeyTrace struct {
	Owner scheduler.CapabilityRef `json:"owner"`
	Name  scheduler.StateName     `json:"name"`
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
		trace.Commits = append(trace.Commits, commitTraceFromScheduler(result.Commit))
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

func commitTraceFromScheduler(commit scheduler.StateCommit) CommitTrace {
	changes := make([]StateChangeTrace, 0, len(commit.Changes))
	for _, change := range commit.Changes {
		changes = append(changes, StateChangeTrace{
			Key:    StateKeyTrace{Owner: change.Key.Owner, Name: change.Key.Name},
			Before: normalizeTraceDataValue(change.Before),
			After:  normalizeTraceDataValue(change.After),
		})
	}
	invalidations := make([]StateKeyTrace, 0, len(commit.Invalidations))
	for _, key := range commit.Invalidations {
		invalidations = append(invalidations, StateKeyTrace{Owner: key.Owner, Name: key.Name})
	}
	return CommitTrace{
		Sequence:      commit.Sequence,
		Event:         commit.Event,
		Committed:     commit.Committed,
		Changes:       changes,
		Invalidations: invalidations,
	}
}

func normalizeTraceDataValue(value scheduler.DataValue) scheduler.DataValue {
	switch typed := value.(type) {
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case int32:
		return float64(typed)
	case float32:
		return float64(typed)
	default:
		return value
	}
}

func CompareTrace(expected Trace, actual Trace) []Diagnostic {
	expected = normalizeTrace(expected)
	actual = normalizeTrace(actual)
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

func normalizeTrace(trace Trace) Trace {
	for index := range trace.Events {
		trace.Events[index].Payload = normalizeTraceDataValue(trace.Events[index].Payload)
	}
	for index := range trace.Commits {
		for changeIndex := range trace.Commits[index].Changes {
			change := &trace.Commits[index].Changes[changeIndex]
			change.Before = normalizeTraceDataValue(change.Before)
			change.After = normalizeTraceDataValue(change.After)
		}
	}
	for index := range trace.ExternalCalls {
		for key, value := range trace.ExternalCalls[index].Input {
			trace.ExternalCalls[index].Input[key] = normalizeTraceDataValue(value)
		}
	}
	return trace
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
