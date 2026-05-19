package packages

import (
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/security"
)

func TestValidateManifestRequiresPackageTypeAndExternalPermissions(t *testing.T) {
	diagnostics := ValidateManifest(Manifest{Name: "@env/storage", Version: "1.0.0"})
	assertPackageDiagnostic(t, diagnostics, "NVA-PKG-001")

	diagnostics = ValidateManifest(Manifest{
		Name:    "@env/storage",
		Version: "1.0.0",
		Types:   []PackageType{PackageExternalCapability},
	})
	assertPackageDiagnostic(t, diagnostics, "NVA-PKG-003")
}

func TestResolvePackagesUsesLockfileDeterministicallyAndSurfacesPermissions(t *testing.T) {
	manifests := []Manifest{
		{
			Name:        "@app/root",
			Version:     "1.0.0",
			Types:       []PackageType{PackageSource},
			ContentHash: "root-hash",
			Dependencies: []Dependency{
				{Name: "@nova/ui", Constraint: ">=1.0.0 <2.0.0"},
				{Name: "@env/storage", Constraint: "1.0.0"},
			},
		},
		{
			Name:        "@nova/ui",
			Version:     "1.1.0",
			Types:       []PackageType{PackageRenderer},
			ContentHash: "ui-hash",
			Exports:     map[string]string{"button": "src/button.nova"},
			Targets:     map[string]TargetAdapter{"web": {Adapter: "platform/web/ui.web.js"}},
		},
		{
			Name:        "@env/storage",
			Version:     "1.0.0",
			Types:       []PackageType{PackageExternalCapability},
			ContentHash: "storage-hash",
			Permissions: security.PermissionMap{
				"storage.read":  true,
				"storage.write": true,
			},
			Targets: map[string]TargetAdapter{"web": {Adapter: "platform/web/storage.web.js"}},
		},
	}

	graph, diagnostics := Resolve(ResolutionInput{
		Roots:      []Dependency{{Name: "@app/root", Constraint: "1.0.0"}},
		Packages:   manifests,
		Target:     "web",
		Production: true,
		Lockfile: Lockfile{Entries: []LockEntry{
			{Name: "@app/root", Version: "1.0.0", ContentHash: "root-hash"},
			{Name: "@env/storage", Version: "1.0.0", ContentHash: "storage-hash", TargetAdapters: map[string]string{"web": "platform/web/storage.web.js"}},
			{Name: "@nova/ui", Version: "1.1.0", ContentHash: "ui-hash", TargetAdapters: map[string]string{"web": "platform/web/ui.web.js"}},
		}},
	})
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}

	wantOrder := []string{"@app/root", "@env/storage", "@nova/ui"}
	if len(graph.Packages) != len(wantOrder) {
		t.Fatalf("packages = %+v, want %d", graph.Packages, len(wantOrder))
	}
	for i, name := range wantOrder {
		if graph.Packages[i].Name != name {
			t.Fatalf("package %d = %s, want %s in %+v", i, graph.Packages[i].Name, name, graph.Packages)
		}
	}
	if sources := graph.PermissionSources["storage.read"]; len(sources) != 1 || sources[0].Package != "@env/storage" {
		t.Fatalf("permission sources = %+v, want @env/storage", graph.PermissionSources)
	}
}

func TestResolvePackagesRejectsUnsupportedTargetsAndLockfileDrift(t *testing.T) {
	graph, diagnostics := Resolve(ResolutionInput{
		Roots:  []Dependency{{Name: "@env/storage", Constraint: "1.0.0"}},
		Target: "android",
		Packages: []Manifest{{
			Name:        "@env/storage",
			Version:     "1.0.0",
			Types:       []PackageType{PackageExternalCapability},
			ContentHash: "new-hash",
			Permissions: security.PermissionMap{
				"storage.read": true,
			},
			Targets: map[string]TargetAdapter{"web": {Adapter: "platform/web/storage.web.js"}},
		}},
		Production: true,
		Lockfile: Lockfile{Entries: []LockEntry{
			{Name: "@env/storage", Version: "1.0.0", ContentHash: "old-hash"},
		}},
	})

	if len(graph.Packages) != 1 {
		t.Fatalf("resolver should still return inspectable graph, got %+v", graph)
	}
	assertPackageDiagnostic(t, diagnostics, "NVA-PKG-004")
	assertPackageDiagnostic(t, diagnostics, "NVA-PKG-006")
}

func TestResolvePackagesReportsVersionConflictsWithDependencyChain(t *testing.T) {
	_, diagnostics := Resolve(ResolutionInput{
		Roots: []Dependency{
			{Name: "@app/root", Constraint: "1.0.0"},
			{Name: "@scope/feature", Constraint: "1.0.0"},
		},
		Packages: []Manifest{
			{
				Name:         "@app/root",
				Version:      "1.0.0",
				Types:        []PackageType{PackageSource},
				Dependencies: []Dependency{{Name: "@scope/shared", Constraint: ">=1.0.0 <2.0.0"}},
			},
			{
				Name:         "@scope/feature",
				Version:      "1.0.0",
				Types:        []PackageType{PackageSource},
				Dependencies: []Dependency{{Name: "@scope/shared", Constraint: ">=2.0.0 <3.0.0"}},
			},
			{Name: "@scope/shared", Version: "1.5.0", Types: []PackageType{PackageSource}},
			{Name: "@scope/shared", Version: "2.1.0", Types: []PackageType{PackageSource}},
		},
	})

	assertPackageDiagnostic(t, diagnostics, "NVA-PKG-002")
	assertPackageDiagnostic(t, diagnostics, "@scope/feature -> @scope/shared")
}

func assertPackageDiagnostic(t *testing.T, diagnostics []Diagnostic, want string) {
	t.Helper()

	for _, diagnostic := range diagnostics {
		if diagnostic.Code == want || strings.Contains(diagnostic.Message, want) {
			return
		}
	}
	t.Fatalf("missing diagnostic %q in %+v", want, diagnostics)
}
