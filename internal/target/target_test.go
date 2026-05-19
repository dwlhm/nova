package target

import (
	"testing"

	"github.com/dwlhm/nova/internal/security"
)

func TestTargetContractsExposeWebAndAndroidRuntimeBoundaries(t *testing.T) {
	web := WebContract()
	if web.ID != "web" || web.RendererAdapter != "nova-web-dom-renderer" || !web.SupportsHydration {
		t.Fatalf("web contract = %+v", web)
	}
	if !web.PermissionMappings["storage.read"] || !web.PermissionMappings["network.request"] {
		t.Fatalf("web permissions = %+v", web.PermissionMappings)
	}

	android := AndroidContract()
	if android.ID != "android" || android.RendererAdapter != "nova-android-compose-renderer" || !android.SupportsRestoration {
		t.Fatalf("android contract = %+v", android)
	}
	if !android.PermissionMappings["notification.send"] || !android.PermissionMappings["network.request"] {
		t.Fatalf("android permissions = %+v", android.PermissionMappings)
	}
}

func TestValidateArtifactRequiresTargetAbiAndRuntimeMetadata(t *testing.T) {
	artifact := ArtifactMetadata{
		Target:           "web",
		EntryCapability:  "src/App.nova",
		LanguageVersion:  "0.1.0",
		ABIVersion:       "0.1.0",
		SchedulerVersion: "0.1.0",
		ViewIRVersion:    "0.1.0",
		RuntimeVersion:   "0.1.0",
		Permissions:      []security.Permission{"storage.read"},
	}

	if diagnostics := ValidateArtifact(WebContract(), artifact); len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}

	artifact.RuntimeVersion = ""
	artifact.Target = "android"
	diagnostics := ValidateArtifact(WebContract(), artifact)
	if len(diagnostics) != 2 {
		t.Fatalf("diagnostics = %+v, want target mismatch and missing runtime version", diagnostics)
	}
}
