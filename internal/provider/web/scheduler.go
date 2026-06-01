package web

import (
	"fmt"

	"github.com/dwlhm/nova/internal/provider/shared"
	rendererjs "github.com/dwlhm/nova/runtime/nova-renderer-js"
	schedulerjs "github.com/dwlhm/nova/runtime/nova-scheduler-js"
)

func schedulerModule() string {
	return schedulerjs.Source
}

func rendererModule() string {
	return rendererjs.Source
}

func schedulerVersionCheck() error {
	if schedulerjs.Version != shared.SchedulerVersion {
		return fmt.Errorf("web scheduler library version %s does not match provider schedulerVersion %s", schedulerjs.Version, shared.SchedulerVersion)
	}
	if rendererjs.Version != shared.RuntimeVersion {
		return fmt.Errorf("web renderer library version %s does not match provider runtimeVersion %s", rendererjs.Version, shared.RuntimeVersion)
	}
	return nil
}
