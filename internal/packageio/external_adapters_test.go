package packageio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dwlhm/nova/internal/packages"
)

func TestCollectExternalAdapterContentsPrefersProjectOverride(t *testing.T) {
	root := t.TempDir()
	adapterPath := filepath.Join(root, "platform", "web")
	if err := os.MkdirAll(adapterPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	custom := `export function register(NovaExternal) { NovaExternal.define("@env/storage", {}); } // project override`
	if err := os.WriteFile(filepath.Join(adapterPath, "storage.web.js"), []byte(custom), 0o644); err != nil {
		t.Fatalf("write adapter: %v", err)
	}

	requests := []ExternalAdapterRequest{{
		CapabilitySource: "@env/storage",
		Path:             "platform/web/storage.web.js",
	}}
	manifests := []packages.Manifest{{
		Name:  "@env/storage",
		Types: []packages.PackageType{packages.PackageExternalCapability},
		Targets: map[string]packages.TargetAdapter{
			"web": {Adapter: "platform/web/storage.web.js", Content: "// package content"},
		},
	}}

	contents := CollectExternalAdapterContents(root, "web", requests, manifests)
	if contents["platform/web/storage.web.js"] != custom {
		t.Fatalf("contents = %q, want project override", contents["platform/web/storage.web.js"])
	}
}

func TestCollectExternalAdapterContentsUsesPackageContentWhenProjectMissing(t *testing.T) {
	root := t.TempDir()
	requests := []ExternalAdapterRequest{{
		CapabilitySource: "@env/storage",
		Path:             "platform/web/storage.web.js",
	}}
	packageContent := `export function register(NovaExternal) { NovaExternal.define("@env/storage", {}); } // package`
	manifests := []packages.Manifest{{
		Name:  "@env/storage",
		Types: []packages.PackageType{packages.PackageExternalCapability},
		Targets: map[string]packages.TargetAdapter{
			"web": {Adapter: "platform/web/storage.web.js", Content: packageContent},
		},
	}}

	contents := CollectExternalAdapterContents(root, "web", requests, manifests)
	if contents["platform/web/storage.web.js"] != packageContent {
		t.Fatalf("contents = %q, want package content", contents["platform/web/storage.web.js"])
	}
}
