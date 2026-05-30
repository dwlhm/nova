package effect

import (
	"context"

	"github.com/dwlhm/nova/internal/scheduler"
)

type ExternalOperationPort interface {
	Invoke(context.Context, scheduler.ExternalOperationRequest) (scheduler.ExternalOperationResult, error)
}
