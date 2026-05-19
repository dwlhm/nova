package standard

import (
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/view"
)

func TestOfficialPackagesExposeMVPStandardSurface(t *testing.T) {
	manifests := OfficialPackages()

	storage, ok := findPackage(manifests, "@env/storage")
	if !ok {
		t.Fatalf("missing @env/storage in official packages: %+v", manifests)
	}
	if !storage.Permissions["storage.read"] || !storage.Permissions["storage.write"] {
		t.Fatalf("@env/storage permissions = %+v, want read/write", storage.Permissions)
	}
	if storage.Targets["web"].Adapter == "" || storage.Targets["android"].Adapter == "" {
		t.Fatalf("@env/storage target adapters = %+v, want web and android", storage.Targets)
	}

	ui, ok := findPackage(manifests, "@nova/ui")
	if !ok {
		t.Fatalf("missing @nova/ui")
	}
	if _, ok := ui.Exports["button"]; !ok {
		t.Fatalf("@nova/ui exports = %+v, want button primitive", ui.Exports)
	}
}

func TestValidateAccessibilityRequiresInteractiveLabel(t *testing.T) {
	nodes := []view.Node{
		{
			Kind:   "button",
			Props:  map[string]view.Binding{},
			Events: map[string]view.EventRoute{"on_press": {}},
		},
	}

	diagnostics := ValidateAccessibility(nodes)
	if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, "accessible label") {
		t.Fatalf("diagnostics = %+v, want missing accessible label", diagnostics)
	}

	labeled := []view.Node{
		{
			Kind:  "button",
			Props: map[string]view.Binding{"label": {Text: "Save"}},
		},
	}
	if diagnostics := ValidateAccessibility(labeled); len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics for labeled button: %+v", diagnostics)
	}
}
