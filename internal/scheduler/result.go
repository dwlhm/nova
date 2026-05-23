package scheduler

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
