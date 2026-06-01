package architecture

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/dwlhm/nova"

func TestInternalPackageDependenciesStayLayered(t *testing.T) {
	importsByPackage, err := collectProductionImports(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}

	allowed := allowedInternalImports()
	failures := make([]string, 0)
	for importer, imports := range importsByPackage {
		allowedImports, ok := allowed[importer]
		if !ok {
			failures = append(failures, importer+" is not registered in the architecture dependency map")
			continue
		}
		for imported := range imports {
			if !strings.HasPrefix(imported, modulePath+"/internal/") {
				continue
			}
			relativeImport := strings.TrimPrefix(imported, modulePath+"/")
			if !allowedImports[relativeImport] {
				failures = append(failures, importer+" imports "+relativeImport+" outside its architecture boundary")
			}
		}
	}

	sort.Strings(failures)
	for _, failure := range failures {
		t.Error(failure)
	}
}

func TestHostIOImportsStayAtEdges(t *testing.T) {
	importsByPackage, err := collectProductionImports(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}

	hostImports := set("net", "net/http", "os", "os/exec", "path/filepath")
	hostPackages := set("cmd/nova", "internal/cli", "internal/bundler", "internal/conformance", "internal/packageio")
	failures := make([]string, 0)
	for importer, imports := range importsByPackage {
		if hostPackages[importer] {
			continue
		}
		for imported := range imports {
			if hostImports[imported] {
				failures = append(failures, importer+" imports host IO package "+imported)
			}
		}
	}

	sort.Strings(failures)
	for _, failure := range failures {
		t.Error(failure)
	}
}

func TestCorePackagesDoNotImportProvider(t *testing.T) {
	importsByPackage, err := collectProductionImports(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}

	failures := make([]string, 0)
	for importer, imports := range importsByPackage {
		if !strings.HasPrefix(importer, "internal/core/") {
			continue
		}
		for imported := range imports {
			if strings.HasPrefix(imported, modulePath+"/internal/provider/") {
				failures = append(failures, importer+" imports provider package "+strings.TrimPrefix(imported, modulePath+"/"))
			}
		}
	}

	sort.Strings(failures)
	for _, failure := range failures {
		t.Error(failure)
	}
}

func TestProviderTargetsStaySandboxed(t *testing.T) {
	importsByPackage, err := collectProductionImports(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}

	failures := make([]string, 0)
	for importer, imports := range importsByPackage {
		switch importer {
		case "internal/provider/web":
			if imports[modulePath+"/internal/provider/android"] {
				failures = append(failures, importer+" imports internal/provider/android")
			}
		case "internal/provider/android":
			if imports[modulePath+"/internal/provider/web"] {
				failures = append(failures, importer+" imports internal/provider/web")
			}
		}
	}

	sort.Strings(failures)
	for _, failure := range failures {
		t.Error(failure)
	}
}

func allowedInternalImports() map[string]map[string]bool {
	return map[string]map[string]bool{
		"cmd/nova": set("internal/cli"),
		"internal/core/app": set(
			"internal/core/diagnostic",
			"internal/core/effect",
			"internal/core/scheduler",
			"internal/core/types",
			"internal/core/view",
		),
		"internal/core/capability": set("internal/core/parser"),
		"internal/core/contract":   set(),
		"internal/core/diagnostic": set(),
		"internal/core/effect": set(
			"internal/core/scheduler",
			"internal/core/types",
		),
		"internal/core/format": set(),
		"internal/core/ir": set(
			"internal/core/capability",
			"internal/core/contract",
			"internal/core/lexer",
			"internal/core/parser",
			"internal/core/routing",
			"internal/core/security",
			"internal/core/view",
		),
		"internal/core/ast": set("internal/core/parser"),
		"internal/core/compile": set(
			"internal/core/ast",
			"internal/core/ir",
			"internal/core/lexer",
			"internal/core/parser",
			"internal/core/plan",
			"internal/core/semantic",
			"internal/core/security",
		),
		"internal/core/lexer": set(),
		"internal/core/parser": set(
			"internal/core/lexer",
		),
		"internal/core/plan": set(
			"internal/core/ast",
			"internal/core/capability",
			"internal/core/parser",
			"internal/project",
		),
		"internal/core/semantic": set(
			"internal/core/ast",
			"internal/core/lexer",
			"internal/core/validator",
		),
		"internal/core/persistence": set(
			"internal/core/diagnostic",
			"internal/core/scheduler",
		),
		"internal/core/routing": set(),
		"internal/core/scheduler": set(
			"internal/core/types",
		),
		"internal/core/security": set(
			"internal/core/parser",
			"internal/core/scheduler",
			"internal/core/types",
		),
		"internal/core/types": set(
			"internal/core/lexer",
			"internal/core/parser",
		),
		"internal/core/validator": set(
			"internal/core/capability",
			"internal/core/lexer",
			"internal/core/parser",
			"internal/core/types",
			"internal/core/view",
		),
		"internal/core/view": set(
			"internal/core/lexer",
			"internal/core/parser",
			"internal/core/scheduler",
		),
		"internal/provider/artifact": set(
			"internal/provider/android",
			"internal/provider/shared",
			"internal/provider/web",
			"internal/core/contract",
			"internal/core/ir",
		),
		"internal/provider/shared": set(
			"internal/core/contract",
			"internal/core/diagnostic",
			"internal/core/ir",
			"internal/project",
			"internal/core/security",
			"internal/provider/build",
			"internal/provider/target",
		),
		"internal/provider/web": set(
			"internal/provider/build",
			"internal/provider/shared",
			"internal/provider/standard",
			"internal/provider/target",
			"runtime/nova-renderer-js",
			"runtime/nova-scheduler-js",
		),
		"internal/provider/android": set(
			"internal/provider/build",
			"internal/core/contract",
			"internal/core/diagnostic",
			"internal/core/ir",
			"internal/project",
			"internal/core/routing",
			"internal/core/security",
			"internal/provider/shared",
			"internal/provider/standard",
			"internal/provider/target",
			"runtime/nova-scheduler-java",
		),
		"internal/provider/build": set(
			"internal/core/ast",
			"internal/core/compile",
			"internal/core/ir",
			"internal/core/plan",
			"internal/packages",
			"internal/core/parser",
			"internal/project",
			"internal/core/security",
			"internal/provider/standard",
			"internal/provider/target",
			"internal/core/view",
		),
		"internal/provider/standard": set(
			"internal/core/diagnostic",
			"internal/packages",
			"internal/core/security",
			"internal/core/view",
		),
		"internal/provider/target": set(
			"internal/core/diagnostic",
			"internal/core/security",
		),
		"internal/bundler": set(),
		"internal/cli": set(
			"internal/provider/artifact",
			"internal/provider/build",
			"internal/bundler",
			"internal/conformance",
			"internal/dev",
			"internal/core/ast",
			"internal/core/compile",
			"internal/core/diagnostic",
			"internal/core/format",
			"internal/core/ir",
			"internal/lsp",
			"internal/packageio",
			"internal/packages",
			"internal/project",
		),
		"internal/conformance": set(
			"internal/provider/artifact",
			"internal/provider/build",
			"internal/core/ast",
			"internal/core/compile",
			"internal/core/contract",
			"internal/core/diagnostic",
			"internal/core/effect",
			"internal/core/ir",
			"internal/core/lexer",
			"internal/packageio",
			"internal/packages",
			"internal/core/parser",
			"internal/project",
			"internal/core/routing",
			"internal/core/scheduler",
			"internal/core/security",
			"internal/core/view",
		),
		"internal/dev": set(),
		"internal/lsp": set(
			"internal/core/format",
			"internal/core/lexer",
			"internal/packages",
			"internal/core/parser",
			"internal/core/semantic",
			"internal/provider/standard",
		),
		"internal/packageio": set(
			"internal/core/diagnostic",
			"internal/packages",
			"internal/project",
			"internal/provider/standard",
		),
		"internal/packages": set(
			"internal/core/diagnostic",
			"internal/core/security",
		),
		"internal/project": set(),
		"internal/tooling": set(),
	}
}

func collectProductionImports(root string) (map[string]map[string]bool, error) {
	importsByPackage := make(map[string]map[string]bool)
	for _, scanRoot := range []string{filepath.Join(root, "cmd"), filepath.Join(root, "internal")} {
		if err := filepath.WalkDir(scanRoot, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			packagePath, err := filepath.Rel(root, filepath.Dir(path))
			if err != nil {
				return err
			}
			packagePath = filepath.ToSlash(packagePath)
			if importsByPackage[packagePath] == nil {
				importsByPackage[packagePath] = make(map[string]bool)
			}

			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, imported := range file.Imports {
				importPath, err := strconv.Unquote(imported.Path.Value)
				if err != nil {
					return err
				}
				importsByPackage[packagePath][importPath] = true
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return importsByPackage, nil
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate architecture test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func set(items ...string) map[string]bool {
	out := make(map[string]bool, len(items))
	for _, item := range items {
		out[item] = true
	}
	return out
}
