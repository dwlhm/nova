package scheduler

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
