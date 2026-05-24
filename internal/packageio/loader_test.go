package packageio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/diagnostic"
	"github.com/dwlhm/nova/internal/project"
)

func TestResolveProjectGraphReadsRendererPackageAdapters(t *testing.T) {
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
	writeTestFile(t, root, "packages/charts/platform/web/register.web.js", `export function register(NovaRenderer) {
  NovaRenderer.definePrimitive("sparkline", {});
}
`)

	graph, diagnostics := ResolveProjectGraph(root, "web", project.Manifest{
		Renderer: project.RendererConfig{
			ExtensionPackages: []project.RendererPackageRef{{Name: "@acme/charts", Constraint: "*"}},
		},
	})
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}
	if len(graph.RendererExtensions) != 1 {
		t.Fatalf("renderer extensions = %+v, want one", graph.RendererExtensions)
	}
	extension := graph.RendererExtensions[0]
	if extension.TargetAdapter != "platform/web/register.web.js" || !strings.Contains(extension.TargetAdapterContent, "definePrimitive") {
		t.Fatalf("extension = %+v, want adapter content", extension)
	}
}

func TestResolveProjectGraphReportsMissingRequiredRendererAdapter(t *testing.T) {
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

	_, diagnostics := ResolveProjectGraph(root, "web", project.Manifest{
		Renderer: project.RendererConfig{
			ExtensionPackages: []project.RendererPackageRef{{Name: "@acme/charts", Constraint: "*"}},
		},
	})
	assertPackageIODiagnostic(t, diagnostics, "NVA-RENDER-004")
}

func writeTestFile(t *testing.T, root string, path string, content string) {
	t.Helper()
	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertPackageIODiagnostic(t *testing.T, diagnostics []diagnostic.Diagnostic, want string) {
	t.Helper()
	for _, item := range diagnostics {
		if item.Code == want || strings.Contains(item.Message, want) {
			return
		}
	}
	t.Fatalf("missing diagnostic %q in %+v", want, diagnostics)
}
