package app

import (
	"context"

	"github.com/dwlhm/nova/internal/effect"
	"github.com/dwlhm/nova/internal/scheduler"
	"github.com/dwlhm/nova/internal/view"
)

type HostPort interface {
	Start(context.Context, AppInstance) error
	Stop(context.Context, AppInstance) error
}

type AppInstance struct {
	ID        InstanceID
	Root      scheduler.CapabilityRef
	Target    TargetID
	Scheduler scheduler.Runtime
	Renderer  view.RendererPort
	Host      HostPort
	External  effect.ExternalOperationPort
}
