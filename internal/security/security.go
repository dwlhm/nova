package security

import (
	"fmt"

	"github.com/dwlhm/nova/internal/parser"
	"github.com/dwlhm/nova/internal/scheduler"
	novatypes "github.com/dwlhm/nova/internal/types"
)

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

func PermissionSet(names ...Permission) PermissionMap {
	permissions := make(PermissionMap, len(names))
	for _, name := range names {
		permissions[name] = true
	}
	return permissions
}

func AuditPermissions(input AuditInput) []Diagnostic {
	operations := operationPermissionMap(input.OperationPermissions)
	diagnostics := make([]Diagnostic, 0)

	for _, call := range input.ExternalCalls {
		for _, permission := range operations[operationKey(call.CapabilitySource, call.CapabilityName, call.Operation)] {
			if !input.ProjectPermissions[permission] {
				diagnostics = append(diagnostics, Diagnostic{
					Code:           "NVA-SEC-001",
					Permission:     permission,
					RequestingFile: call.RequestingFile,
					Message: fmt.Sprintf(
						"project manifest does not grant permission %s; permission %s is required by %s.%s requested by %s in %s",
						permission,
						permission,
						call.CapabilityName,
						call.Operation,
						call.RequestingFile,
						call.Lifecycle,
					),
				})
			}
			if !input.TargetPermissions[permission] {
				diagnostics = append(diagnostics, Diagnostic{
					Code:           "NVA-SEC-002",
					Permission:     permission,
					RequestingFile: call.RequestingFile,
					Message: fmt.Sprintf(
						"target does not map permission %s required by %s.%s requested by %s",
						permission,
						call.CapabilityName,
						call.Operation,
						call.RequestingFile,
					),
				})
			}
		}
	}

	return diagnostics
}

func ValidateHostEvent(env novatypes.Environment, contracts []EventContract, event HostEvent) []Diagnostic {
	contract, ok := findEventContract(contracts, event.Name)
	if !ok {
		return []Diagnostic{{
			Code:    "NVA-SEC-010",
			Message: fmt.Sprintf("undeclared scheduler event %s", event.Name),
		}}
	}

	diagnostics := make([]Diagnostic, 0)
	if len(contract.Emitters) > 0 && !canEmit(contract.Emitters, event.Source) {
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "NVA-SEC-011",
			Message: fmt.Sprintf("%s cannot emit scheduler event %s", event.Source, event.Name),
		})
	}
	if !novatypes.IsSerializable(event.Payload) {
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "NVA-SEC-012",
			Message: fmt.Sprintf("event %s payload must be serializable Nova data", event.Name),
		})
		return diagnostics
	}

	return append(diagnostics, validateEventPayload(env, contract, event)...)
}

func validateEventPayload(env novatypes.Environment, contract EventContract, event HostEvent) []Diagnostic {
	payloadType := contract.PayloadType
	if payloadType == "" || payloadType == "unknown" {
		return nil
	}
	if payloadType == "void" {
		if event.Payload == nil {
			return nil
		}
		return []Diagnostic{{
			Code:    "NVA-SEC-013",
			Message: fmt.Sprintf("event %s payload must match void", event.Name),
		}}
	}

	typ, typeDiagnostics := novatypes.ParseRef(env, parser.TypeRef{Text: payloadType})
	if len(typeDiagnostics) > 0 {
		return []Diagnostic{{
			Code:    "NVA-SEC-014",
			Message: fmt.Sprintf("event %s has invalid payload type %s: %s", event.Name, payloadType, typeDiagnostics[0].Message),
		}}
	}
	if err := env.ValidateValue(typ, event.Payload); err != nil {
		return []Diagnostic{{
			Code:    "NVA-SEC-015",
			Message: fmt.Sprintf("event %s payload must match %s", event.Name, payloadType),
		}}
	}
	return nil
}

func operationPermissionMap(permissions []OperationPermission) map[string][]Permission {
	out := make(map[string][]Permission, len(permissions))
	for _, permission := range permissions {
		out[operationKey(permission.CapabilitySource, permission.CapabilityName, permission.Operation)] = append(
			out[operationKey(permission.CapabilitySource, permission.CapabilityName, permission.Operation)],
			permission.Requires...,
		)
	}
	return out
}

func operationKey(source string, capabilityName string, operation string) string {
	return source + "\x00" + capabilityName + "\x00" + operation
}

func findEventContract(contracts []EventContract, name scheduler.SchedulerEvent) (EventContract, bool) {
	for _, contract := range contracts {
		if contract.Name == name {
			return contract, true
		}
	}
	return EventContract{}, false
}

func canEmit(emitters []scheduler.CapabilityRef, source scheduler.CapabilityRef) bool {
	for _, emitter := range emitters {
		if emitter == source {
			return true
		}
	}
	return false
}
