package effect

import (
	"context"
	"fmt"

	"github.com/dwlhm/nova/internal/scheduler"
	novatypes "github.com/dwlhm/nova/internal/types"
)

func CompleteExternalOperations(ctx context.Context, runtime scheduler.Runtime, port ExternalOperationPort, requests []scheduler.ExternalOperationRequest) (scheduler.Runtime, CompletionResult) {
	next := runtime
	result := CompletionResult{Completions: make([]ExternalCompletion, 0, len(requests))}

	for _, request := range requests {
		completion := ExternalCompletion{Request: request}
		output, err := port.Invoke(ctx, request)
		if err != nil {
			next, completion = enqueueFailure(next, request, err, completion)
			result.Completions = append(result.Completions, completion)
			if completion.Error != nil {
				result.Errors = append(result.Errors, *completion.Error)
			}
			continue
		}

		if request.OutputType != "" {
			if err := novatypes.ValidateValueText(request.OutputType, output.Output); err != nil {
				next, completion = enqueueFailure(next, request, fmt.Errorf("external operation %s.%s returned invalid output: %w", request.Capability, request.Operation, err), completion)
				result.Completions = append(result.Completions, completion)
				if completion.Error != nil {
					result.Errors = append(result.Errors, *completion.Error)
				}
				continue
			}
		}

		if request.OnSuccess != "" {
			var envelope scheduler.EventEnvelope
			var enqueueErr error
			next, envelope, enqueueErr = scheduler.Enqueue(next, request.Source, request.OnSuccess, output.Output)
			if enqueueErr != nil {
				completion.Error = schedulerError(request, request.OnSuccess, enqueueErr)
				result.Errors = append(result.Errors, *completion.Error)
			} else {
				completion.Emitted = append(completion.Emitted, envelope)
			}
		}
		for _, event := range output.Emit {
			var envelope scheduler.EventEnvelope
			var enqueueErr error
			next, envelope, enqueueErr = scheduler.Enqueue(next, event.Source, event.Name, event.Payload)
			if enqueueErr != nil {
				completion.Error = schedulerError(request, event.Name, enqueueErr)
				result.Errors = append(result.Errors, *completion.Error)
				continue
			}
			completion.Emitted = append(completion.Emitted, envelope)
		}
		result.Completions = append(result.Completions, completion)
	}

	return next, result
}

func enqueueFailure(runtime scheduler.Runtime, request scheduler.ExternalOperationRequest, cause error, completion ExternalCompletion) (scheduler.Runtime, ExternalCompletion) {
	if request.OnFailure == "" {
		completion.Error = schedulerError(request, "", cause)
		return runtime, completion
	}

	next, envelope, err := scheduler.Enqueue(runtime, request.Source, request.OnFailure, cause.Error())
	if err != nil {
		completion.Error = schedulerError(request, request.OnFailure, err)
		return next, completion
	}
	completion.Emitted = append(completion.Emitted, envelope)
	return next, completion
}

func schedulerError(request scheduler.ExternalOperationRequest, event scheduler.SchedulerEvent, cause error) *scheduler.SchedulerError {
	return &scheduler.SchedulerError{
		Source:  request.Source,
		Phase:   scheduler.SchedulerPhaseExternalOperation,
		Event:   event,
		Message: cause.Error(),
		Cause:   cause,
	}
}
