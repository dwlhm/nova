package security

import "fmt"

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
