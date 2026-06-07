package plan

import (
	"testing"

	"github.com/dwlhm/nova/internal/core/ast"
	"github.com/dwlhm/nova/internal/core/parser"
)

func TestResolveModuleGraphFollowsPackageImports(t *testing.T) {
	sources := map[string]ast.RawFile{
		"src/App.nova": {
			Imports: []parser.ImportDecl{{
				From: "@acme/charts/sparkline",
			}},
		},
		"packages/acme/charts/src/sparkline.nova": {},
	}
	exports := map[string]string{
		"@acme/charts/sparkline": "packages/acme/charts/src/sparkline.nova",
	}
	modules, diagnostics := resolveModuleGraph("src/App.nova", sources, exports)
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
	if len(modules) != 2 {
		t.Fatalf("modules = %+v", modules)
	}
	if modules[0] != "src/App.nova" || modules[1] != "packages/acme/charts/src/sparkline.nova" {
		t.Fatalf("module order = %+v", modules)
	}
}
