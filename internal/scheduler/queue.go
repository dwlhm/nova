package scheduler

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
