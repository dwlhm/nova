package scheduler

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
