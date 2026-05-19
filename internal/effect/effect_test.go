package effect

import (
	"context"
	"errors"
	"testing"

	"github.com/dwlhm/nova/internal/scheduler"
)

type fakeExternalPort struct {
	result scheduler.ExternalOperationResult
	err    error
}

func (p fakeExternalPort) Invoke(context.Context, scheduler.ExternalOperationRequest) (scheduler.ExternalOperationResult, error) {
	return p.result, p.err
}

func TestCompleteExternalOperationsEnqueuesValidatedSuccessEvent(t *testing.T) {
	runtime := scheduler.NewRuntime(nil, nil)
	request := scheduler.ExternalOperation(
		"counter",
		"storage",
		"load",
		map[string]scheduler.DataValue{"key": "counter"},
		"number",
		"@loaded",
		"@load_failed",
	)

	next, result := CompleteExternalOperations(context.Background(), runtime, fakeExternalPort{
		result: scheduler.ExternalOperationResult{Output: 42},
	}, []scheduler.ExternalOperationRequest{request})

	if len(result.Errors) != 0 {
		t.Fatalf("unexpected errors: %+v", result.Errors)
	}
	queue := next.Queue()
	if len(queue) != 1 || queue[0].Name != "@loaded" || queue[0].Payload != 42 {
		t.Fatalf("queue = %+v, want @loaded(42)", queue)
	}
}

func TestCompleteExternalOperationsRoutesAdapterFailureToFailureEvent(t *testing.T) {
	runtime := scheduler.NewRuntime(nil, nil)
	request := scheduler.ExternalOperation(
		"counter",
		"storage",
		"load",
		nil,
		"number",
		"@loaded",
		"@load_failed",
	)

	next, result := CompleteExternalOperations(context.Background(), runtime, fakeExternalPort{
		err: errors.New("disk offline"),
	}, []scheduler.ExternalOperationRequest{request})

	if len(result.Errors) != 0 {
		t.Fatalf("failure event should handle adapter error, got %+v", result.Errors)
	}
	queue := next.Queue()
	if len(queue) != 1 || queue[0].Name != "@load_failed" || queue[0].Payload != "disk offline" {
		t.Fatalf("queue = %+v, want @load_failed message", queue)
	}
}

func TestCompleteExternalOperationsValidatesBoundaryOutput(t *testing.T) {
	runtime := scheduler.NewRuntime(nil, nil)
	request := scheduler.ExternalOperation(
		"counter",
		"storage",
		"load",
		nil,
		"number",
		"@loaded",
		"@load_failed",
	)

	next, result := CompleteExternalOperations(context.Background(), runtime, fakeExternalPort{
		result: scheduler.ExternalOperationResult{Output: "not a number"},
	}, []scheduler.ExternalOperationRequest{request})

	if len(result.Errors) != 0 {
		t.Fatalf("failure event should handle validation error, got %+v", result.Errors)
	}
	queue := next.Queue()
	if len(queue) != 1 || queue[0].Name != "@load_failed" {
		t.Fatalf("queue = %+v, want @load_failed", queue)
	}
}
