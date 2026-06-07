package conformance

import (
	"path/filepath"
	"testing"
)

func TestAndroidFinanceArtifactFixture(t *testing.T) {
	root := filepath.Join("..", "..", "tests", "conformance", "android", "finance")
	result := RunFixtureDir(root)
	if !result.Passed() {
		t.Fatalf("fixture diagnostics = %+v", result.Diagnostics)
	}
	if result.Target != "android" {
		t.Fatalf("target = %q, want android", result.Target)
	}
}
