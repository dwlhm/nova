package scheduler

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
