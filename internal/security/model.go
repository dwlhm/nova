package security

import "github.com/dwlhm/nova/internal/scheduler"

type Permission string

type PermissionMap map[Permission]bool

type OperationPermission struct {
	CapabilitySource string
	CapabilityName   string
	Operation        string
	Requires         []Permission
}

type ExternalCall struct {
	RequestingFile   string
	Lifecycle        string
	CapabilitySource string
	CapabilityName   string
	Operation        string
}

type AuditInput struct {
	ProjectPermissions   PermissionMap
	TargetPermissions    PermissionMap
	OperationPermissions []OperationPermission
	ExternalCalls        []ExternalCall
}

type EventContract struct {
	Name        scheduler.SchedulerEvent
	PayloadType string
	Emitters    []scheduler.CapabilityRef
}

type HostEvent struct {
	Source  scheduler.CapabilityRef
	Name    scheduler.SchedulerEvent
	Payload scheduler.DataValue
}

type Diagnostic struct {
	Code           string
	Message        string
	RequestingFile string
	Permission     Permission
}
