package target

import (
	"fmt"

	"github.com/dwlhm/nova/internal/core/diagnostic"
	"github.com/dwlhm/nova/internal/core/security"
)

type RuntimeContract struct {
	ID                  string
	RuntimePackage      string
	RendererAdapter     string
	HostAdapter         string
	ExternalAdapter     string
	ArtifactRoot        string
	PermissionMappings  security.PermissionMap
	SupportsHydration   bool
	SupportsRestoration bool
}

type ArtifactMetadata struct {
	Target             string
	EntryCapability    string
	LanguageVersion    string
	ABIVersion         string
	SchedulerVersion   string
	ViewIRVersion      string
	RuntimeVersion     string
	Permissions        []security.Permission
	ExternalOperations []string
}

type Diagnostic = diagnostic.Diagnostic

func WebContract() RuntimeContract {
	return RuntimeContract{
		ID:                "web",
		RuntimePackage:    "nova-web-runtime",
		RendererAdapter:   "nova-web-dom-renderer",
		HostAdapter:       "nova-web-host-adapter",
		ExternalAdapter:   "nova-web-external-adapter",
		ArtifactRoot:      "build/web",
		SupportsHydration: true,
		PermissionMappings: security.PermissionSet(
			"storage.read",
			"storage.write",
			"network.request",
			"clipboard.read",
			"clipboard.write",
			"notification.send",
			"device.info",
		),
	}
}

func AndroidContract() RuntimeContract {
	return RuntimeContract{
		ID:                  "android",
		RuntimePackage:      "nova-android-runtime",
		RendererAdapter:     "nova-android-view-renderer",
		HostAdapter:         "nova-android-host-adapter",
		ExternalAdapter:     "nova-android-env-adapters",
		ArtifactRoot:        "build/android",
		SupportsRestoration: true,
		PermissionMappings: security.PermissionSet(
			"storage.read",
			"storage.write",
			"network.request",
			"clipboard.read",
			"clipboard.write",
			"notification.send",
			"device.info",
		),
	}
}

func ValidateArtifact(contract RuntimeContract, artifact ArtifactMetadata) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	if artifact.Target != contract.ID {
		diagnostics = append(diagnostics, targetDiagnostic("NVA-TARGET-016", fmt.Sprintf("artifact target %s does not match runtime target %s", artifact.Target, contract.ID)))
	}
	required := []struct {
		name  string
		value string
	}{
		{"entry capability", artifact.EntryCapability},
		{"language version", artifact.LanguageVersion},
		{"ABI version", artifact.ABIVersion},
		{"scheduler version", artifact.SchedulerVersion},
		{"ViewIR version", artifact.ViewIRVersion},
		{"runtime version", artifact.RuntimeVersion},
	}
	for _, field := range required {
		if field.value == "" {
			diagnostics = append(diagnostics, targetDiagnostic("NVA-TARGET-017", fmt.Sprintf("artifact metadata must include %s", field.name)))
		}
	}
	for _, permission := range artifact.Permissions {
		if !contract.PermissionMappings[permission] {
			diagnostics = append(diagnostics, targetDiagnostic("NVA-TARGET-018", fmt.Sprintf("permission %s is not mapped by target %s", permission, contract.ID)))
		}
	}
	return diagnostics
}

func targetDiagnostic(code string, message string) Diagnostic {
	return Diagnostic{Code: code, Severity: diagnostic.SeverityError, Message: message}
}
