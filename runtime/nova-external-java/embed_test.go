package externaljava

import (
	"strings"
	"testing"
)

func TestSourceIncludesInvokeAndPermissionHelpers(t *testing.T) {
	source := Source("nova.generated.demo")
	if !strings.Contains(source, "checkPermission") {
		t.Fatal("NovaExternal must expose checkPermission")
	}
	if !strings.Contains(source, "validateOutput") {
		t.Fatal("NovaExternal must expose validateOutput")
	}
	if !strings.Contains(source, "invokeSync") {
		t.Fatal("NovaExternal must expose invokeSync")
	}
}
