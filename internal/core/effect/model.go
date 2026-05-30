package effect

import "github.com/dwlhm/nova/internal/scheduler"

type ExternalCompletion struct {
	Request scheduler.ExternalOperationRequest
	Emitted []scheduler.EventEnvelope
	Error   *scheduler.SchedulerError
}

type CompletionResult struct {
	Completions []ExternalCompletion
	Errors      []scheduler.SchedulerError
}
