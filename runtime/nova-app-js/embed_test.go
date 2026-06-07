package appjs

import (
	"strings"
	"testing"
)

func TestLifecycleJSExportsBinder(t *testing.T) {
	source := LifecycleJS()
	if !strings.Contains(source, "NovaAppLifecycle") {
		t.Fatal("missing NovaAppLifecycle export")
	}
	if !strings.Contains(source, "@app_started") {
		t.Fatal("missing @app_started hook")
	}
	if !strings.Contains(source, "visibilitychange") {
		t.Fatal("missing visibilitychange hook")
	}
	if !strings.Contains(source, "@app_restored") {
		t.Fatal("missing @app_restored restore hook")
	}
	if !strings.Contains(source, "sessionStorage") {
		t.Fatal("missing session snapshot persistence")
	}
}
