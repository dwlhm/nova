package effect

import (
	"context"

	"github.com/dwlhm/nova/internal/core/scheduler"
)

type ExternalOperationPort interface {
	Invoke(context.Context, scheduler.ExternalOperationRequest) (scheduler.ExternalOperationResult, error)
}
