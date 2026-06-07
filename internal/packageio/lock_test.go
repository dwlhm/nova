package packageio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/project"
)

func TestBuildExportIndexResolvesPackagePaths(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "packages/acme/charts/nova.package.toml", `[package]
name = "@acme/charts"
version = "1.0.0"
type = ["source-package"]

[exports]
sparkline = "src/sparkline.nova"
`)
	writeTestFile(t, root, "packages/acme/charts/src/sparkline.nova", `<contract type Point> x: number; y: number; /|`)

	manifests, diagnostics := LoadProjectManifests(root, "web", nil)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}
	index := BuildExportIndex(root, manifests)
	got := index["@acme/charts/sparkline"]
	want := "packages/acme/charts/src/sparkline.nova"
	if got != want {
		t.Fatalf("export = %q, want %q", got, want)
	}
}

func TestGenerateLockfilePinsRendererExtension(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "packages/charts/nova.package.toml", `[package]
name = "@acme/charts"
version = "1.0.0"
type = ["renderer-package"]

[renderer.primitives.sparkline]
props = ["data"]

[renderer.primitives.sparkline.web]
strategy = "adapter"

[targets.web]
adapter = "platform/web/register.web.js"
`)
	writeTestFile(t, root, "packages/charts/platform/web/register.web.js", `export function register() {}`)

	lockfile, diagnostics := GenerateLockfile(root, "web", project.Manifest{
		Renderer: project.RendererConfig{
			ExtensionPackages: []project.RendererPackageRef{{Name: "@acme/charts", Constraint: "*"}},
		},
	})
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}
	if len(lockfile.Entries) != 1 {
		t.Fatalf("entries = %+v", lockfile.Entries)
	}
	entry := lockfile.Entries[0]
	if entry.Name != "@acme/charts" || entry.Version != "1.0.0" {
		t.Fatalf("entry = %+v", entry)
	}
	if entry.ContentHash == "" {
		t.Fatal("expected content hash")
	}
	if entry.TargetAdapters["web"] != "platform/web/register.web.js" {
		t.Fatalf("adapters = %+v", entry.TargetAdapters)
	}

	path := filepath.Join(root, "nova.lock")
	if err := WriteLockfile(root, lockfile); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), entry.ContentHash) {
		t.Fatalf("lock content missing hash:\n%s", content)
	}
}
