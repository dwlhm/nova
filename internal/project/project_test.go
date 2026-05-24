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
styles = ["src/global.css", "src/counter.css"]
scoped_styles = ["src/App.css"]

[targets.android]
	renderer = "@nova/android"
application_id = "dev.example.audiolab"
compile_sdk = 35

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
	if got := manifest.Targets["web"].Styles; len(got) != 2 || got[0] != "src/global.css" || got[1] != "src/counter.css" {
		t.Fatalf("web styles = %+v, want global/counter css", got)
	}
	if got := manifest.Targets["web"].ScopedStyles; len(got) != 1 || got[0] != "src/App.css" {
		t.Fatalf("web scoped styles = %+v, want App.css", got)
	}
	if got := manifest.Targets["android"].Options["application_id"]; got != "dev.example.audiolab" {
		t.Fatalf("android application_id = %q, want dev.example.audiolab", got)
	}
	if got := manifest.Targets["android"].Options["compile_sdk"]; got != "35" {
		t.Fatalf("android compile_sdk = %q, want 35", got)
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

func TestParseManifestCapturesRendererExtensionsAndLocalDictionary(t *testing.T) {
	input := `[project]
name = "audiolab"
version = "0.1.0"
entry = "src/App.nova"

[renderer]
unknown_kind = "warn"

[renderer.extensions]
packages = ["@acme/charts"]

[[renderer.dictionary]]
kind = "local_meter"
props = ["value"]
events = ["on_press"]
allow_override = true

[renderer.dictionary.web]
strategy = "adapter"
adapter = "platform/web/local_meter.web.js"
`

	manifest, diagnostics := ParseManifest(input)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}
	if manifest.Renderer.UnknownKind != RendererUnknownKindWarn {
		t.Fatalf("unknown_kind = %q, want warn", manifest.Renderer.UnknownKind)
	}
	if len(manifest.Renderer.ExtensionPackages) != 1 || manifest.Renderer.ExtensionPackages[0].Name != "@acme/charts" {
		t.Fatalf("extensions = %+v, want @acme/charts", manifest.Renderer.ExtensionPackages)
	}
	if len(manifest.Renderer.Dictionary) != 1 {
		t.Fatalf("dictionary = %+v, want one entry", manifest.Renderer.Dictionary)
	}
	entry := manifest.Renderer.Dictionary[0]
	if entry.Kind != "local_meter" || !entry.AllowOverride {
		t.Fatalf("dictionary entry = %+v", entry)
	}
	if len(entry.Props) != 1 || entry.Props[0].Name != "value" {
		t.Fatalf("dictionary props = %+v", entry.Props)
	}
	if got := entry.Targets["web"].Adapter; got != "platform/web/local_meter.web.js" {
		t.Fatalf("web adapter = %q", got)
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
