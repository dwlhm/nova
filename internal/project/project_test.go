package project

import (
	"strings"
	"testing"
)

func TestParseManifestCapturesProjectTargetsAndPermissions(t *testing.T) {
	input := `[project]
name = "audiolab"
version = "0.1.0"
entry = "src/App.nova"

[targets.web]
renderer = "@nova/web"

[targets.android]
renderer = "@nova/android"

[permissions]
storage.read = true
storage.write = false
`

	manifest, diagnostics := ParseManifest(input)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}
	if manifest.Project.Name != "audiolab" || manifest.Project.Version != "0.1.0" || manifest.Project.Entry != "src/App.nova" {
		t.Fatalf("unexpected project metadata: %+v", manifest.Project)
	}
	if manifest.Targets["web"].Renderer != "@nova/web" {
		t.Fatalf("web renderer = %q, want @nova/web", manifest.Targets["web"].Renderer)
	}
	if !manifest.Permissions["storage.read"] {
		t.Fatalf("storage.read should be granted: %+v", manifest.Permissions)
	}
	if manifest.Permissions["storage.write"] {
		t.Fatalf("storage.write should be denied: %+v", manifest.Permissions)
	}
}

func TestParseManifestCapturesScopedPermissionDeclarations(t *testing.T) {
	input := `[project]
name = "audiolab"
version = "0.1.0"
entry = "src/App.nova"

[permissions.storage]
read = ["settings", "profile"]
write = ["settings"]
`

	manifest, diagnostics := ParseManifest(input)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}
	if !manifest.Permissions["storage.read"] || !manifest.Permissions["storage.write"] {
		t.Fatalf("scoped permissions should be granted: %+v", manifest.Permissions)
	}
	if got := manifest.PermissionScopes["storage.read"]; len(got) != 2 || got[0] != "settings" || got[1] != "profile" {
		t.Fatalf("storage.read scopes = %+v, want settings/profile", got)
	}
	if got := manifest.PermissionScopes["storage.write"]; len(got) != 1 || got[0] != "settings" {
		t.Fatalf("storage.write scopes = %+v, want settings", got)
	}
}

func TestValidateLayoutAllowsRecommendedProjectShape(t *testing.T) {
	manifest := Manifest{
		Project: Project{Name: "audiolab", Version: "0.1.0", Entry: "src/App.nova"},
		Targets: map[string]Target{
			"web": {Renderer: "@nova/web"},
		},
	}
	files := []File{
		{Path: "nova.toml"},
		{Path: "src/App.nova"},
		{Path: "src/components/Button.nova"},
		{Path: "platform/web/storage.web.js"},
		{Path: "tests/fixtures/valid/counter.nova"},
		{Path: "docs/adr/adr_001_language_specification.md"},
		{Path: "build/web/metadata.json"},
	}

	diagnostics := ValidateLayout(manifest, files)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}
}

func TestValidateLayoutReportsMissingRequiredFilesAndNamingDrift(t *testing.T) {
	manifest := Manifest{
		Project: Project{Name: "audiolab", Version: "0.1.0", Entry: "src/App.nova"},
	}
	files := []File{
		{Path: "src/app.nova"},
		{Path: "src/build/generated.nova"},
		{Path: "docs/adr/language.md"},
	}

	diagnostics := ValidateLayout(manifest, files)
	assertProjectDiagnostic(t, diagnostics, "nova.toml")
	assertProjectDiagnostic(t, diagnostics, "entry source src/App.nova")
	assertProjectDiagnostic(t, diagnostics, "capability source src/app.nova should use PascalCase.nova")
	assertProjectDiagnostic(t, diagnostics, "generated output should stay under build/")
	assertProjectDiagnostic(t, diagnostics, "ADR docs should use adr_NNN_slug.md")
}

func TestPackageImportNamesFollowReservedAndScopedConventions(t *testing.T) {
	allowed := []string{"@nova/ui/Button", "@env/storage", "@company/ui", "@dwlhm/audio/filter"}
	for _, source := range allowed {
		if !IsPackageImport(source) {
			t.Fatalf("%s should be accepted as a package import", source)
		}
	}

	discouraged := []string{"nova-ui", "Button", "./Button.nova"}
	for _, source := range discouraged {
		if IsPackageImport(source) {
			t.Fatalf("%s should not be treated as a package import", source)
		}
	}
}

func assertProjectDiagnostic(t *testing.T, diagnostics []Diagnostic, want string) {
	t.Helper()

	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, want) {
			return
		}
	}
	t.Fatalf("missing diagnostic %q in %+v", want, diagnostics)
}
