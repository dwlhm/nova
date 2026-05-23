package app

import (
	"github.com/dwlhm/nova/internal/diagnostic"
	"github.com/dwlhm/nova/internal/scheduler"
)

const (
	EventAppStarted  scheduler.SchedulerEvent = "@app_started"
	EventAppResumed  scheduler.SchedulerEvent = "@app_resumed"
	EventAppPaused   scheduler.SchedulerEvent = "@app_paused"
	EventAppStopped  scheduler.SchedulerEvent = "@app_stopped"
	EventAppRestored scheduler.SchedulerEvent = "@app_restored"

	EventRouteChanged     scheduler.SchedulerEvent = "@route_changed"
	EventNavigate         scheduler.SchedulerEvent = "@navigate"
	EventNavigationFailed scheduler.SchedulerEvent = "@navigation_failed"
)

type InstanceID string
type TargetID string
type Diagnostic = diagnostic.Diagnostic
