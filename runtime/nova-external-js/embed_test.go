package externaljs

import (
	"strings"
	"testing"
)

func TestCoreJSIncludesPermissionAndValidationHelpers(t *testing.T) {
	if !strings.Contains(CoreJS(), "checkPermission") {
		t.Fatal("NovaExternal core must expose checkPermission")
	}
	if !strings.Contains(CoreJS(), "validateOutput") {
		t.Fatal("NovaExternal core must expose validateOutput")
	}
	if !strings.Contains(CoreJS(), "isSerializable") {
		t.Fatal("NovaExternal core must expose isSerializable")
	}
}
