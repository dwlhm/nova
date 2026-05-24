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

func allowedInternalImports() map[string]map[string]bool {
	return map[string]map[string]bool{
		"cmd/nova":     set("internal/cli"),
		"internal/app": set("internal/diagnostic", "internal/effect", "internal/scheduler", "internal/types", "internal/view"),
		"internal/artifact": set(
			"internal/build",
			"internal/capability",
			"internal/diagnostic",
			"internal/lexer",
			"internal/parser",
			"internal/project",
			"internal/routing",
			"internal/security",
			"internal/target",
			"internal/view",
			"runtime/nova-renderer-js",
			"runtime/nova-scheduler-java",
			"runtime/nova-scheduler-js",
		),
		"internal/build":       set("internal/packages", "internal/parser", "internal/project", "internal/security", "internal/standard"),
		"internal/bundler":     set(),
		"internal/capability":  set("internal/parser"),
		"internal/cli":         set("internal/artifact", "internal/build", "internal/bundler", "internal/conformance", "internal/dev", "internal/diagnostic", "internal/format", "internal/lexer", "internal/lsp", "internal/packageio", "internal/packages", "internal/parser", "internal/project", "internal/validator"),
		"internal/conformance": set("internal/artifact", "internal/build", "internal/diagnostic", "internal/lexer", "internal/packageio", "internal/parser", "internal/project", "internal/routing", "internal/scheduler", "internal/security", "internal/validator", "internal/view"),
		"internal/dev":         set(),
		"internal/diagnostic":  set(),
		"internal/effect":      set("internal/scheduler", "internal/types"),
		"internal/format":      set(),
		"internal/lexer":       set(),
		"internal/lsp":         set("internal/format", "internal/lexer", "internal/packages", "internal/parser", "internal/standard", "internal/validator"),
		"internal/packageio":   set("internal/diagnostic", "internal/packages", "internal/project", "internal/standard"),
		"internal/packages":    set("internal/diagnostic", "internal/security"),
		"internal/parser":      set("internal/lexer"),
		"internal/persistence": set("internal/diagnostic", "internal/scheduler"),
		"internal/project":     set(),
		"internal/routing":     set(),
		"internal/scheduler":   set("internal/types"),
		"internal/security":    set("internal/parser", "internal/scheduler", "internal/types"),
		"internal/standard":    set("internal/diagnostic", "internal/packages", "internal/security", "internal/view"),
		"internal/target":      set("internal/diagnostic", "internal/security"),
		"internal/tooling":     set(),
		"internal/types":       set("internal/lexer", "internal/parser"),
		"internal/validator":   set("internal/capability", "internal/lexer", "internal/parser", "internal/types", "internal/view"),
		"internal/view":        set("internal/lexer", "internal/parser", "internal/scheduler"),
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
