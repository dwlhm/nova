package view

import (
	"context"

	"github.com/dwlhm/nova/internal/scheduler"
)

type RendererPort interface {
	Mount(context.Context, IR) error
	Update(context.Context, scheduler.StateCommit, DependencyMetadata) error
	Dispose(context.Context) error
}
