package scheduler

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
