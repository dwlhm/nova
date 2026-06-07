package androidcodegen

import (
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/core/security"
	"github.com/dwlhm/nova/internal/provider/build"
	androidtarget "github.com/dwlhm/nova/internal/provider/capability/view/target/android"
)

func TestExternalBindingsJavaIncludesOperationSpec(t *testing.T) {
	source := ExternalBindingsJava([]build.ResolvedExternalOperation{{
		CapabilitySource: "@env/storage",
		Operation:        "set",
		Output:           "void",
		Permissions:      []security.Permission{"storage.write"},
	}}, []security.Permission{"storage.write"}, androidtarget.Config{Namespace: "nova.generated"})
	if !strings.Contains(source, "PROJECT_PERMISSIONS") {
		t.Fatalf("missing project permissions: %q", source)
	}
	if !strings.Contains(source, "OperationSpec spec") {
		t.Fatalf("missing spec helper: %q", source)
	}
	if !strings.Contains(source, `"@env/storage#set"`) {
		t.Fatalf("missing operation id: %q", source)
	}
}
