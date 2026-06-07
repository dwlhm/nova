package runtimejava

import (
	"strings"
	"testing"
)

func TestSourceIncludesPackageAndNovaExprInvoke(t *testing.T) {
	source := Source("nova.generated.finance")
	if !strings.HasPrefix(source, "package nova.generated.finance;\n\n") {
		t.Fatalf("unexpected package header:\n%s", source[:80])
	}
	if !strings.Contains(source, "NovaExpr.invoke(name, args)") {
		t.Fatalf("NovaRuntime must delegate pure func calls to NovaExpr.invoke")
	}
	if strings.Contains(source, "NovaExprInvoke") {
		t.Fatalf("embedded runtime must not reference NovaExprInvoke")
	}
}

func TestHydrationSourceIncludesRestoreHelper(t *testing.T) {
	source := HydrationSource("nova.generated.finance")
	if !strings.Contains(source, "restorePersistedState") {
		t.Fatalf("NovaHydration must expose restorePersistedState")
	}
}

func TestRuntimeSourceIncludesRouteValidation(t *testing.T) {
	source := Source("nova.generated.demo")
	if !strings.Contains(source, "validateRouteData") {
		t.Fatalf("NovaRuntime must expose validateRouteData")
	}
}

func TestRuntimeSourceIncludesSnapshotHelpers(t *testing.T) {
	source := Source("nova.generated.demo")
	if !strings.Contains(source, "encodeSnapshotPayload") {
		t.Fatalf("NovaRuntime must expose encodeSnapshotPayload")
	}
	if !strings.Contains(source, "decodeSnapshotPayload") {
		t.Fatalf("NovaRuntime must expose decodeSnapshotPayload")
	}
}
