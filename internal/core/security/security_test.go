package security

import (
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/core/scheduler"
	novatypes "github.com/dwlhm/nova/internal/core/types"
)

func TestAuditPermissionsDeniesUndeclaredAndUnmappedExternalAccess(t *testing.T) {
	diagnostics := AuditPermissions(AuditInput{
		ProjectPermissions: PermissionSet("storage.read"),
		TargetPermissions:  PermissionSet("storage.read"),
		OperationPermissions: []OperationPermission{
			{
				CapabilitySource: "@env/storage",
				CapabilityName:   "storage",
				Operation:        "load",
				Requires:         []Permission{"storage.read"},
			},
			{
				CapabilitySource: "@env/storage",
				CapabilityName:   "storage",
				Operation:        "set",
				Requires:         []Permission{"storage.write"},
			},
			{
				CapabilitySource: "@env/notify",
				CapabilityName:   "notify",
				Operation:        "send",
				Requires:         []Permission{"notification.send"},
			},
		},
		ExternalCalls: []ExternalCall{
			{
				RequestingFile:   "src/App.nova",
				Lifecycle:        "after @sync",
				CapabilitySource: "@env/storage",
				CapabilityName:   "storage",
				Operation:        "load",
			},
			{
				RequestingFile:   "src/App.nova",
				Lifecycle:        "after @sync",
				CapabilitySource: "@env/storage",
				CapabilityName:   "storage",
				Operation:        "set",
			},
			{
				RequestingFile:   "src/App.nova",
				Lifecycle:        "after @select",
				CapabilitySource: "@env/notify",
				CapabilityName:   "notify",
				Operation:        "send",
			},
		},
	})

	assertSecurityDiagnostic(t, diagnostics, "permission storage.write is required by storage.set")
	assertSecurityDiagnostic(t, diagnostics, "target does not map permission notification.send")
	assertSecurityDiagnostic(t, diagnostics, "project manifest does not grant permission notification.send")
	if hasSecurityDiagnostic(diagnostics, "storage.read") {
		t.Fatalf("storage.read should be granted and mapped: %+v", diagnostics)
	}
}

func TestValidateHostEventEnforcesContractPayloadAndEmitter(t *testing.T) {
	env := novatypes.EmptyEnvironment()
	contracts := []EventContract{
		{
			Name:        "@set",
			PayloadType: "number",
			Emitters:    []scheduler.CapabilityRef{"renderer.CounterButton"},
		},
		{
			Name:        "@reset",
			PayloadType: "void",
			Emitters:    []scheduler.CapabilityRef{"host.Keyboard"},
		},
	}

	valid := ValidateHostEvent(env, contracts, HostEvent{
		Source:  "renderer.CounterButton",
		Name:    "@set",
		Payload: 7,
	})
	if len(valid) != 0 {
		t.Fatalf("unexpected diagnostics for valid event: %+v", valid)
	}

	wrongPayload := ValidateHostEvent(env, contracts, HostEvent{
		Source:  "renderer.CounterButton",
		Name:    "@set",
		Payload: "7",
	})
	assertSecurityDiagnostic(t, wrongPayload, "event @set payload must match number")

	wrongEmitter := ValidateHostEvent(env, contracts, HostEvent{
		Source:  "renderer.Other",
		Name:    "@set",
		Payload: 7,
	})
	assertSecurityDiagnostic(t, wrongEmitter, "renderer.Other cannot emit scheduler event @set")

	undeclared := ValidateHostEvent(env, contracts, HostEvent{
		Source:  "renderer.CounterButton",
		Name:    "@missing",
		Payload: nil,
	})
	assertSecurityDiagnostic(t, undeclared, "undeclared scheduler event @missing")
}

func TestValidateHostEventRejectsNonSerializablePayload(t *testing.T) {
	diagnostics := ValidateHostEvent(novatypes.EmptyEnvironment(), []EventContract{
		{
			Name:        "@debug",
			PayloadType: "unknown",
			Emitters:    []scheduler.CapabilityRef{"devtools"},
		},
	}, HostEvent{
		Source:  "devtools",
		Name:    "@debug",
		Payload: func() {},
	})

	assertSecurityDiagnostic(t, diagnostics, "payload must be serializable Nova data")
}

func assertSecurityDiagnostic(t *testing.T, diagnostics []Diagnostic, want string) {
	t.Helper()

	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, want) {
			return
		}
	}
	t.Fatalf("missing diagnostic %q in %+v", want, diagnostics)
}

func hasSecurityDiagnostic(diagnostics []Diagnostic, want string) bool {
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, want) {
			return true
		}
	}
	return false
}
