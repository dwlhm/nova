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
	case map[string]scheduler.DataValue:
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			out[key] = normalizeTraceDataValue(value)
		}
		return out
	case map[string]any:
		return normalizeTraceStringMap(typed)
	default:
		return value
	}
}

func normalizeTraceStringMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = normalizeTraceDataValue(value)
	}
	return out
}

type CompareTraceOptions struct {
	IgnoreErrors bool
}

func CompareTrace(expected Trace, actual Trace, options ...CompareTraceOptions) []Diagnostic {
	opts := CompareTraceOptions{}
	if len(options) > 0 {
		opts = options[0]
	}
	expected = normalizeTrace(expected)
	actual = normalizeTrace(actual)
	diagnostics := make([]Diagnostic, 0)
	if len(expected.Events) > 0 && !reflect.DeepEqual(expected.Events, actual.Events) {
		diagnostics = append(diagnostics, mismatch("NVA-CONFORMANCE-001", "scheduler events", expected.Events, actual.Events))
	}
	if len(expected.Commits) > 0 {
		diagnostics = append(diagnostics, compareCommitTraces(expected.Commits, actual.Commits)...)
	}
	if len(expected.LifecycleCalls) > 0 && !reflect.DeepEqual(expected.LifecycleCalls, actual.LifecycleCalls) {
		diagnostics = append(diagnostics, mismatch("NVA-CONFORMANCE-003", "lifecycle calls", expected.LifecycleCalls, actual.LifecycleCalls))
	}
	if len(expected.ExternalCalls) > 0 {
		diagnostics = append(diagnostics, compareExternalCallTraces(expected.ExternalCalls, actual.ExternalCalls)...)
	}
	if !opts.IgnoreErrors && len(expected.Errors) > 0 && !reflect.DeepEqual(expected.Errors, actual.Errors) {
		diagnostics = append(diagnostics, mismatch("NVA-CONFORMANCE-005", "runtime errors", expected.Errors, actual.Errors))
	}
	return diagnostic.StableSort(diagnostics)
}

func compareCommitTraces(expected []CommitTrace, actual []CommitTrace) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	for _, expectedCommit := range expected {
		actualCommit, ok := findCommitTrace(actual, expectedCommit.Sequence, expectedCommit.Event)
		if !ok {
			diagnostics = append(diagnostics, mismatch("NVA-CONFORMANCE-002", "state commits", expectedCommit, actual))
			continue
		}
		if expectedCommit.Committed != actualCommit.Committed {
			diagnostics = append(diagnostics, mismatch("NVA-CONFORMANCE-002", "state commits", expectedCommit, actualCommit))
			continue
		}
		for _, expectedChange := range expectedCommit.Changes {
			actualChange, ok := findStateChange(actualCommit.Changes, expectedChange.Key)
			if !ok || !stateChangeMatches(expectedChange, actualChange) {
				diagnostics = append(diagnostics, mismatch("NVA-CONFORMANCE-002", "state commits", expectedChange, actualChange))
			}
		}
		for _, expectedKey := range expectedCommit.Invalidations {
			if !containsStateKey(actualCommit.Invalidations, expectedKey) {
				diagnostics = append(diagnostics, mismatch("NVA-CONFORMANCE-002", "state commits", expectedKey, actualCommit.Invalidations))
			}
		}
	}
	return diagnostics
}

func compareExternalCallTraces(expected []ExternalCallTrace, actual []ExternalCallTrace) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	for index, expectedCall := range expected {
		if index >= len(actual) {
			diagnostics = append(diagnostics, mismatch("NVA-CONFORMANCE-004", "external calls", expectedCall, actual))
			continue
		}
		actualCall := actual[index]
		if expectedCall.Source != actualCall.Source ||
			expectedCall.Capability != actualCall.Capability ||
			expectedCall.Operation != actualCall.Operation ||
			expectedCall.OutputType != actualCall.OutputType ||
			expectedCall.OnSuccess != actualCall.OnSuccess ||
			expectedCall.OnFailure != actualCall.OnFailure {
			diagnostics = append(diagnostics, mismatch("NVA-CONFORMANCE-004", "external calls", expectedCall, actualCall))
			continue
		}
		for key, expectedValue := range expectedCall.Input {
			actualValue, ok := actualCall.Input[key]
			if !ok || !reflect.DeepEqual(normalizeTraceDataValue(expectedValue), normalizeTraceDataValue(actualValue)) {
				diagnostics = append(diagnostics, mismatch("NVA-CONFORMANCE-004", "external calls", expectedCall.Input, actualCall.Input))
				break
			}
		}
	}
	return diagnostics
}

func findCommitTrace(commits []CommitTrace, sequence scheduler.LogicalSequence, event scheduler.SchedulerEvent) (CommitTrace, bool) {
	for _, commit := range commits {
		if commit.Sequence == sequence && commit.Event == event {
			return commit, true
		}
	}
	return CommitTrace{}, false
}

func findStateChange(changes []StateChangeTrace, key StateKeyTrace) (StateChangeTrace, bool) {
	for _, change := range changes {
		if change.Key == key {
			return change, true
		}
	}
	return StateChangeTrace{}, false
}

func stateChangeMatches(expected StateChangeTrace, actual StateChangeTrace) bool {
	if expected.Before != nil && !reflect.DeepEqual(normalizeTraceDataValue(expected.Before), normalizeTraceDataValue(actual.Before)) {
		return false
	}
	if expected.After != nil && !reflect.DeepEqual(normalizeTraceDataValue(expected.After), normalizeTraceDataValue(actual.After)) {
		return false
	}
	return true
}

func containsStateKey(keys []StateKeyTrace, want StateKeyTrace) bool {
	for _, key := range keys {
		if key == want {
			return true
		}
	}
	return false
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
