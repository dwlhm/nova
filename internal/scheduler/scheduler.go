package scheduler

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	novatypes "github.com/dwlhm/nova/internal/types"
)

type LogicalSequence uint64
type CapabilityRef string
type SchedulerEvent string
type StateName string
type DataValue any

type LifecyclePhase string

const (
	PhaseMount   LifecyclePhase = "mount"
	PhaseDispose LifecyclePhase = "dispose"
	PhaseBefore  LifecyclePhase = "before"
	PhaseAfter   LifecyclePhase = "after"
	PhaseError   LifecyclePhase = "error"
)

type SchedulerPhase string

const (
	SchedulerPhaseEnqueue           SchedulerPhase = "enqueue"
	SchedulerPhaseTransition        SchedulerPhase = "transition"
	SchedulerPhaseLifecycleMount    SchedulerPhase = "lifecycle_mount"
	SchedulerPhaseLifecycleDispose  SchedulerPhase = "lifecycle_dispose"
	SchedulerPhaseLifecycleBefore   SchedulerPhase = "lifecycle_before"
	SchedulerPhaseLifecycleAfter    SchedulerPhase = "lifecycle_after"
	SchedulerPhaseLifecycleError    SchedulerPhase = "lifecycle_error"
	SchedulerPhaseExternalOperation SchedulerPhase = "external_operation"
)

var (
	ErrInvalidEventName  = errors.New("scheduler event name must start with @")
	ErrInvalidPayload    = errors.New("event payload must be a serializable Nova data value")
	ErrInvalidStateValue = errors.New("state transition produced value outside state type")
	ErrQueueOverloaded   = errors.New("scheduler queue overloaded")
)

type StateKey struct {
	Owner CapabilityRef
	Name  StateName
}

func Key(owner CapabilityRef, name StateName) StateKey {
	return StateKey{Owner: owner, Name: name}
}

type EventEnvelope struct {
	Sequence LogicalSequence
	Source   CapabilityRef
	Name     SchedulerEvent
	Payload  DataValue
}

type EventToEmit struct {
	Source  CapabilityRef
	Name    SchedulerEvent
	Payload DataValue
}

func Emit(source CapabilityRef, name SchedulerEvent, payload DataValue) EventToEmit {
	return EventToEmit{Source: source, Name: name, Payload: payload}
}

type Snapshot struct {
	values map[StateKey]DataValue
}

func (s Snapshot) Value(key StateKey) (DataValue, bool) {
	value, ok := s.values[key]
	return value, ok
}

func (s Snapshot) MustValue(key StateKey) DataValue {
	value, ok := s.Value(key)
	if !ok {
		panic(fmt.Sprintf("missing scheduler state %s.%s", key.Owner, key.Name))
	}
	return value
}

func (s Snapshot) Values() map[StateKey]DataValue {
	out := make(map[StateKey]DataValue, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out
}

type TransitionFunc func(Snapshot, EventEnvelope) (DataValue, error)

type TransitionRule struct {
	Event SchedulerEvent
	Apply TransitionFunc
}

func On(event SchedulerEvent, apply TransitionFunc) TransitionRule {
	return TransitionRule{Event: event, Apply: apply}
}

type StateCell struct {
	Owner       CapabilityRef
	Name        StateName
	Type        string
	Value       DataValue
	Transitions []TransitionRule
}

func NewStateCell(owner CapabilityRef, name StateName, typ string, value DataValue, transitions ...TransitionRule) StateCell {
	return StateCell{
		Owner:       owner,
		Name:        name,
		Type:        typ,
		Value:       value,
		Transitions: copyTransitions(transitions),
	}
}

type LifecycleContext struct {
	Snapshot Snapshot
	Event    EventEnvelope
	Error    *SchedulerError
}

type LifecycleFunc func(LifecycleContext) (LifecycleOutput, error)

type ExternalOperationRequest struct {
	Source     CapabilityRef
	Capability string
	Operation  string
	Input      map[string]DataValue
	OutputType string
	OnSuccess  SchedulerEvent
	OnFailure  SchedulerEvent
}

type ExternalOperationResult struct {
	Output DataValue
	Emit   []EventToEmit
}

type LifecycleOutput struct {
	Emit     []EventToEmit
	External []ExternalOperationRequest
}

func ExternalOperation(source CapabilityRef, capability string, operation string, input map[string]DataValue, outputType string, onSuccess SchedulerEvent, onFailure SchedulerEvent) ExternalOperationRequest {
	return ExternalOperationRequest{
		Source:     source,
		Capability: capability,
		Operation:  operation,
		Input:      cloneInput(input),
		OutputType: outputType,
		OnSuccess:  onSuccess,
		OnFailure:  onFailure,
	}
}

type LifecycleHandler struct {
	Owner CapabilityRef
	Phase LifecyclePhase
	Event SchedulerEvent
	Run   LifecycleFunc
}

func Before(owner CapabilityRef, event SchedulerEvent, run LifecycleFunc) LifecycleHandler {
	return LifecycleHandler{Owner: owner, Phase: PhaseBefore, Event: event, Run: run}
}

func After(owner CapabilityRef, event SchedulerEvent, run LifecycleFunc) LifecycleHandler {
	return LifecycleHandler{Owner: owner, Phase: PhaseAfter, Event: event, Run: run}
}

func Mount(owner CapabilityRef, run LifecycleFunc) LifecycleHandler {
	return LifecycleHandler{Owner: owner, Phase: PhaseMount, Run: run}
}

func Dispose(owner CapabilityRef, run LifecycleFunc) LifecycleHandler {
	return LifecycleHandler{Owner: owner, Phase: PhaseDispose, Run: run}
}

func OnError(owner CapabilityRef, run LifecycleFunc) LifecycleHandler {
	return LifecycleHandler{Owner: owner, Phase: PhaseError, Run: run}
}

type SchedulerError struct {
	Source  CapabilityRef
	Phase   SchedulerPhase
	Event   SchedulerEvent
	State   *StateKey
	Message string
	Cause   error
}

type StateChange struct {
	Key    StateKey
	Before DataValue
	After  DataValue
}

type StateCommit struct {
	Sequence      LogicalSequence
	Event         SchedulerEvent
	Committed     bool
	Changes       []StateChange
	Invalidations []StateKey
}

type LifecycleInvocation struct {
	Owner  CapabilityRef
	Phase  LifecyclePhase
	Event  SchedulerEvent
	Error  *SchedulerError
	Output LifecycleOutput
	Failed error
}

type StepResult struct {
	Event                EventEnvelope
	Commit               StateCommit
	LifecycleInvocations []LifecycleInvocation
	Emitted              []EventEnvelope
	ExternalOperations   []ExternalOperationRequest
	Errors               []SchedulerError
}

type LifecycleResult struct {
	Phase                LifecyclePhase
	LifecycleInvocations []LifecycleInvocation
	Emitted              []EventEnvelope
	ExternalOperations   []ExternalOperationRequest
	Errors               []SchedulerError
}

type StateValueValidator func(typeRef string, value DataValue) error

type Config struct {
	MaxQueue           int
	ValidateStateValue StateValueValidator
}

type Runtime struct {
	cells        []StateCell
	lifecycles   []LifecycleHandler
	queue        []EventEnvelope
	nextSequence LogicalSequence
	config       Config
}

func NewRuntime(cells []StateCell, lifecycles []LifecycleHandler, configs ...Config) Runtime {
	config := Config{}
	if len(configs) > 0 {
		config = configs[0]
	}
	config = normalizeConfig(config)
	return Runtime{
		cells:        copyCells(cells),
		lifecycles:   copyLifecycles(lifecycles),
		queue:        nil,
		nextSequence: 1,
		config:       config,
	}
}

func (r Runtime) Queue() []EventEnvelope {
	return copyQueue(r.queue)
}

func (r Runtime) State(key StateKey) (DataValue, bool) {
	for _, cell := range r.cells {
		if cellKey(cell) == key {
			return cell.Value, true
		}
	}
	return nil, false
}

func (r Runtime) Cells() []StateCell {
	return copyCells(r.cells)
}

func (r Runtime) NextSequence() LogicalSequence {
	return r.nextSequence
}

func (r Runtime) LastSequence() LogicalSequence {
	if r.nextSequence <= 1 {
		return 0
	}
	return r.nextSequence - 1
}

func NewRuntimeFromState(cells []StateCell, lifecycles []LifecycleHandler, values map[StateKey]DataValue, nextSequence LogicalSequence, configs ...Config) (Runtime, []SchedulerError) {
	runtime := NewRuntime(cells, lifecycles, configs...)
	if nextSequence < 1 {
		nextSequence = 1
	}
	runtime.nextSequence = nextSequence

	pending := make(map[StateKey]DataValue, len(values))
	for key, value := range values {
		pending[key] = value
	}

	validate := stateValueValidator(runtime.config)
	errs := make([]SchedulerError, 0)
	for i, cell := range runtime.cells {
		key := cellKey(cell)
		value, ok := pending[key]
		if !ok {
			continue
		}
		delete(pending, key)
		if err := validate(cell.Type, value); err != nil {
			stateKey := key
			errs = append(errs, SchedulerError{
				Source:  cell.Owner,
				Phase:   SchedulerPhaseTransition,
				State:   &stateKey,
				Message: err.Error(),
				Cause:   err,
			})
			continue
		}
		runtime.cells[i].Value = value
	}

	for key := range pending {
		stateKey := key
		errs = append(errs, SchedulerError{
			Source:  key.Owner,
			Phase:   SchedulerPhaseTransition,
			State:   &stateKey,
			Message: fmt.Sprintf("restored state %s.%s is not present in scheduler schema", key.Owner, key.Name),
		})
	}

	if len(errs) > 0 {
		return NewRuntime(cells, lifecycles, configs...), errs
	}
	return runtime, nil
}

func Enqueue(runtime Runtime, source CapabilityRef, name SchedulerEvent, payload DataValue) (Runtime, EventEnvelope, error) {
	if !isSchedulerEvent(name) {
		return runtime, EventEnvelope{}, ErrInvalidEventName
	}
	if !isSerializable(payload) {
		return runtime, EventEnvelope{}, ErrInvalidPayload
	}
	if runtime.config.MaxQueue > 0 && len(runtime.queue) >= runtime.config.MaxQueue {
		return runtime, EventEnvelope{}, ErrQueueOverloaded
	}

	event := EventEnvelope{
		Sequence: runtime.nextSequence,
		Source:   source,
		Name:     name,
		Payload:  payload,
	}
	next := runtime
	next.queue = appendQueue(runtime.queue, event)
	next.nextSequence = runtime.nextSequence + 1
	return next, event, nil
}

func Step(runtime Runtime) (Runtime, StepResult, bool) {
	if len(runtime.queue) == 0 {
		return runtime, StepResult{}, false
	}

	event := runtime.queue[0]
	base := runtime
	base.queue = copyQueue(runtime.queue[1:])
	result := StepResult{
		Event: event,
		Commit: StateCommit{
			Sequence:  event.Sequence,
			Event:     event.Name,
			Committed: true,
		},
	}
	pending := make([]EventToEmit, 0)

	beforeSnapshot := snapshotFromCells(runtime.cells)
	beforeInvocations, beforeEmits, beforeExternal, beforeErrors := runLifecycles(runtime.lifecycles, PhaseBefore, event, nil, beforeSnapshot)
	result.LifecycleInvocations = append(result.LifecycleInvocations, beforeInvocations...)
	result.ExternalOperations = append(result.ExternalOperations, beforeExternal...)
	result.Errors = append(result.Errors, beforeErrors...)
	pending = append(pending, beforeEmits...)
	if len(beforeErrors) > 0 {
		errorInvocations, errorEmits, errorExternal, errorErrors := routeErrors(runtime.lifecycles, beforeSnapshot, event, beforeErrors)
		result.LifecycleInvocations = append(result.LifecycleInvocations, errorInvocations...)
		result.ExternalOperations = append(result.ExternalOperations, errorExternal...)
		result.Errors = append(result.Errors, errorErrors...)
		pending = append(pending, errorEmits...)
	}

	nextCells, commit, transitionErrors := planAndCommit(runtime.cells, beforeSnapshot, event, stateValueValidator(runtime.config))
	result.Commit = commit
	result.Errors = append(result.Errors, transitionErrors...)

	if len(transitionErrors) > 0 {
		result.Commit.Committed = false
		errorInvocations, errorEmits, errorExternal, errorErrors := routeErrors(runtime.lifecycles, beforeSnapshot, event, transitionErrors)
		result.LifecycleInvocations = append(result.LifecycleInvocations, errorInvocations...)
		result.ExternalOperations = append(result.ExternalOperations, errorExternal...)
		result.Errors = append(result.Errors, errorErrors...)
		pending = append(pending, errorEmits...)
		base.cells = copyCells(runtime.cells)
		return enqueuePending(base, pending, result)
	}

	afterSnapshot := snapshotFromCells(nextCells)
	afterInvocations, afterEmits, afterExternal, afterErrors := runLifecycles(runtime.lifecycles, PhaseAfter, event, nil, afterSnapshot)
	result.LifecycleInvocations = append(result.LifecycleInvocations, afterInvocations...)
	result.ExternalOperations = append(result.ExternalOperations, afterExternal...)
	result.Errors = append(result.Errors, afterErrors...)
	pending = append(pending, afterEmits...)
	if len(afterErrors) > 0 {
		errorInvocations, errorEmits, errorExternal, errorErrors := routeErrors(runtime.lifecycles, afterSnapshot, event, afterErrors)
		result.LifecycleInvocations = append(result.LifecycleInvocations, errorInvocations...)
		result.ExternalOperations = append(result.ExternalOperations, errorExternal...)
		result.Errors = append(result.Errors, errorErrors...)
		pending = append(pending, errorEmits...)
	}

	base.cells = nextCells
	return enqueuePending(base, pending, result)
}

func RunLifecycle(runtime Runtime, phase LifecyclePhase, source CapabilityRef) (Runtime, LifecycleResult) {
	event := EventEnvelope{Source: source}
	snapshot := snapshotFromCells(runtime.cells)
	invocations, pending, external, errs := runLifecycles(runtime.lifecycles, phase, event, nil, snapshot)
	result := LifecycleResult{
		Phase:                phase,
		LifecycleInvocations: invocations,
		ExternalOperations:   external,
		Errors:               errs,
	}

	if phase != PhaseError && len(errs) > 0 {
		errorInvocations, errorEmits, errorExternal, errorErrors := routeErrors(runtime.lifecycles, snapshot, event, errs)
		result.LifecycleInvocations = append(result.LifecycleInvocations, errorInvocations...)
		result.ExternalOperations = append(result.ExternalOperations, errorExternal...)
		result.Errors = append(result.Errors, errorErrors...)
		pending = append(pending, errorEmits...)
	}

	return enqueueLifecyclePending(runtime, pending, result)
}

func Drain(runtime Runtime) (Runtime, []StepResult) {
	results := make([]StepResult, 0)
	next := runtime
	for {
		advanced, result, ok := Step(next)
		if !ok {
			return next, results
		}
		next = advanced
		results = append(results, result)
	}
}

func runLifecycles(lifecycles []LifecycleHandler, phase LifecyclePhase, event EventEnvelope, routed *SchedulerError, snapshot Snapshot) ([]LifecycleInvocation, []EventToEmit, []ExternalOperationRequest, []SchedulerError) {
	invocations := make([]LifecycleInvocation, 0)
	emits := make([]EventToEmit, 0)
	external := make([]ExternalOperationRequest, 0)
	errs := make([]SchedulerError, 0)

	for _, handler := range lifecycles {
		if !matchesLifecycle(handler, phase, event) {
			continue
		}
		invocation := LifecycleInvocation{
			Owner: handler.Owner,
			Phase: handler.Phase,
			Event: handler.Event,
			Error: routed,
		}
		if handler.Run != nil {
			output, err := handler.Run(LifecycleContext{
				Snapshot: snapshot,
				Event:    event,
				Error:    routed,
			})
			invocation.Output = output
			invocation.Failed = err
			if err != nil {
				errs = append(errs, lifecycleError(handler, event, err))
			} else {
				emits = append(emits, output.Emit...)
				external = append(external, cloneExternalRequests(output.External)...)
			}
		}
		invocations = append(invocations, invocation)
	}

	return invocations, emits, external, errs
}

func routeErrors(lifecycles []LifecycleHandler, snapshot Snapshot, event EventEnvelope, errorsToRoute []SchedulerError) ([]LifecycleInvocation, []EventToEmit, []ExternalOperationRequest, []SchedulerError) {
	invocations := make([]LifecycleInvocation, 0)
	emits := make([]EventToEmit, 0)
	external := make([]ExternalOperationRequest, 0)
	errs := make([]SchedulerError, 0)

	for i := range errorsToRoute {
		routed := errorsToRoute[i]
		errorInvocations, errorEmits, errorExternal, errorErrors := runLifecycles(lifecycles, PhaseError, event, &routed, snapshot)
		invocations = append(invocations, errorInvocations...)
		emits = append(emits, errorEmits...)
		external = append(external, errorExternal...)
		for _, err := range errorErrors {
			err.Phase = SchedulerPhaseLifecycleError
			errs = append(errs, err)
		}
	}

	return invocations, emits, external, errs
}

func planAndCommit(cells []StateCell, snapshot Snapshot, event EventEnvelope, validate StateValueValidator) ([]StateCell, StateCommit, []SchedulerError) {
	if validate == nil {
		validate = DefaultStateValueValidator
	}
	commit := StateCommit{
		Sequence:  event.Sequence,
		Event:     event.Name,
		Committed: true,
	}
	type candidate struct {
		cell  StateCell
		key   StateKey
		value DataValue
	}
	candidates := make([]candidate, 0)
	errs := make([]SchedulerError, 0)

	for _, cell := range cells {
		for _, transition := range cell.Transitions {
			if transition.Event != event.Name {
				continue
			}
			if transition.Apply == nil {
				continue
			}
			value, err := transition.Apply(snapshot, event)
			if err != nil {
				key := cellKey(cell)
				errs = append(errs, SchedulerError{
					Source:  cell.Owner,
					Phase:   SchedulerPhaseTransition,
					Event:   event.Name,
					State:   &key,
					Message: err.Error(),
					Cause:   err,
				})
				continue
			}
			candidates = append(candidates, candidate{cell: cell, key: cellKey(cell), value: value})
		}
	}

	if len(errs) > 0 {
		commit.Committed = false
		return copyCells(cells), commit, errs
	}
	for _, candidate := range candidates {
		if err := validate(candidate.cell.Type, candidate.value); err != nil {
			key := candidate.key
			errs = append(errs, SchedulerError{
				Source:  candidate.cell.Owner,
				Phase:   SchedulerPhaseTransition,
				Event:   event.Name,
				State:   &key,
				Message: err.Error(),
				Cause:   err,
			})
		}
	}
	if len(errs) > 0 {
		commit.Committed = false
		return copyCells(cells), commit, errs
	}

	next := copyCells(cells)
	for _, candidate := range candidates {
		for i, cell := range next {
			if cellKey(cell) != candidate.key {
				continue
			}
			before := cell.Value
			if reflect.DeepEqual(before, candidate.value) {
				continue
			}
			next[i].Value = candidate.value
			commit.Changes = append(commit.Changes, StateChange{
				Key:    candidate.key,
				Before: before,
				After:  candidate.value,
			})
			commit.Invalidations = append(commit.Invalidations, candidate.key)
			break
		}
	}

	return next, commit, nil
}

func enqueuePending(runtime Runtime, pending []EventToEmit, result StepResult) (Runtime, StepResult, bool) {
	next := runtime
	for _, event := range pending {
		var envelope EventEnvelope
		var err error
		next, envelope, err = Enqueue(next, event.Source, event.Name, event.Payload)
		if err != nil {
			result.Errors = append(result.Errors, SchedulerError{
				Source:  event.Source,
				Phase:   SchedulerPhaseEnqueue,
				Event:   event.Name,
				Message: err.Error(),
				Cause:   err,
			})
			continue
		}
		result.Emitted = append(result.Emitted, envelope)
	}
	return next, result, true
}

func enqueueLifecyclePending(runtime Runtime, pending []EventToEmit, result LifecycleResult) (Runtime, LifecycleResult) {
	next := runtime
	for _, event := range pending {
		var envelope EventEnvelope
		var err error
		next, envelope, err = Enqueue(next, event.Source, event.Name, event.Payload)
		if err != nil {
			result.Errors = append(result.Errors, SchedulerError{
				Source:  event.Source,
				Phase:   SchedulerPhaseEnqueue,
				Event:   event.Name,
				Message: err.Error(),
				Cause:   err,
			})
			continue
		}
		result.Emitted = append(result.Emitted, envelope)
	}
	return next, result
}

func lifecycleError(handler LifecycleHandler, event EventEnvelope, err error) SchedulerError {
	phase := SchedulerPhaseLifecycleError
	switch handler.Phase {
	case PhaseMount:
		phase = SchedulerPhaseLifecycleMount
	case PhaseDispose:
		phase = SchedulerPhaseLifecycleDispose
	case PhaseBefore:
		phase = SchedulerPhaseLifecycleBefore
	case PhaseAfter:
		phase = SchedulerPhaseLifecycleAfter
	case PhaseError:
		phase = SchedulerPhaseLifecycleError
	}
	return SchedulerError{
		Source:  handler.Owner,
		Phase:   phase,
		Event:   event.Name,
		Message: err.Error(),
		Cause:   err,
	}
}

func matchesLifecycle(handler LifecycleHandler, phase LifecyclePhase, event EventEnvelope) bool {
	if handler.Phase != phase {
		return false
	}
	switch phase {
	case PhaseBefore, PhaseAfter:
		return handler.Event == event.Name
	case PhaseError:
		return handler.Event == "" || handler.Event == event.Name
	default:
		return true
	}
}

func snapshotFromCells(cells []StateCell) Snapshot {
	values := make(map[StateKey]DataValue, len(cells))
	for _, cell := range cells {
		values[cellKey(cell)] = cell.Value
	}
	return Snapshot{values: values}
}

func cellKey(cell StateCell) StateKey {
	return Key(cell.Owner, cell.Name)
}

func isSchedulerEvent(name SchedulerEvent) bool {
	return strings.HasPrefix(string(name), "@") && len(name) > 1
}

func normalizeConfig(config Config) Config {
	if config.ValidateStateValue == nil {
		config.ValidateStateValue = DefaultStateValueValidator
	}
	return config
}

func stateValueValidator(config Config) StateValueValidator {
	if config.ValidateStateValue != nil {
		return config.ValidateStateValue
	}
	return DefaultStateValueValidator
}

func DefaultStateValueValidator(typeRef string, value DataValue) error {
	typeRef = strings.TrimSpace(typeRef)
	if typeRef == "" || typeRef == "unknown" {
		return nil
	}
	if err := novatypes.ValidateValueText(typeRef, value); err == nil {
		return nil
	}
	return fmt.Errorf("%w: expected %s", ErrInvalidStateValue, typeRef)
}

func matchesStateType(typeRef string, value DataValue) bool {
	for _, candidate := range strings.Split(typeRef, "|") {
		if matchesSingleStateType(strings.TrimSpace(candidate), value) {
			return true
		}
	}
	return false
}

func matchesSingleStateType(typeRef string, value DataValue) bool {
	switch typeRef {
	case "", "unknown":
		return true
	case "void", "null":
		return value == nil
	}
	if value == nil {
		return false
	}
	if strings.HasSuffix(typeRef, "[]") {
		return isArrayValue(reflect.ValueOf(value))
	}

	valueRef := unwrapValue(reflect.ValueOf(value))
	switch typeRef {
	case "string":
		return valueRef.IsValid() && valueRef.Kind() == reflect.String
	case "number":
		return isNumberValue(valueRef)
	case "boolean":
		return valueRef.IsValid() && valueRef.Kind() == reflect.Bool
	default:
		return isSerializable(value)
	}
}

func unwrapValue(value reflect.Value) reflect.Value {
	for value.IsValid() && value.Kind() == reflect.Interface {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func isNumberValue(value reflect.Value) bool {
	value = unwrapValue(value)
	if !value.IsValid() {
		return false
	}
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func isArrayValue(value reflect.Value) bool {
	value = unwrapValue(value)
	if !value.IsValid() {
		return false
	}
	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		return true
	default:
		return false
	}
}

func isSerializable(value DataValue) bool {
	if value == nil {
		return true
	}
	return isSerializableValue(reflect.ValueOf(value))
}

func isSerializableValue(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}
	switch value.Kind() {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	case reflect.Interface:
		if value.IsNil() {
			return true
		}
		return isSerializableValue(value.Elem())
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if !isSerializableValue(value.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return false
		}
		for _, key := range value.MapKeys() {
			if !isSerializableValue(value.MapIndex(key)) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func appendQueue(queue []EventEnvelope, event EventEnvelope) []EventEnvelope {
	out := make([]EventEnvelope, len(queue)+1)
	copy(out, queue)
	out[len(queue)] = event
	return out
}

func copyQueue(queue []EventEnvelope) []EventEnvelope {
	out := make([]EventEnvelope, len(queue))
	copy(out, queue)
	return out
}

func copyCells(cells []StateCell) []StateCell {
	out := make([]StateCell, len(cells))
	for i, cell := range cells {
		out[i] = cell
		out[i].Transitions = copyTransitions(cell.Transitions)
	}
	return out
}

func copyTransitions(transitions []TransitionRule) []TransitionRule {
	out := make([]TransitionRule, len(transitions))
	copy(out, transitions)
	return out
}

func copyLifecycles(lifecycles []LifecycleHandler) []LifecycleHandler {
	out := make([]LifecycleHandler, len(lifecycles))
	copy(out, lifecycles)
	return out
}

func cloneExternalRequests(requests []ExternalOperationRequest) []ExternalOperationRequest {
	out := make([]ExternalOperationRequest, len(requests))
	for i, request := range requests {
		out[i] = request
		out[i].Input = cloneInput(request.Input)
	}
	return out
}

func cloneInput(input map[string]DataValue) map[string]DataValue {
	if input == nil {
		return nil
	}
	out := make(map[string]DataValue, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
