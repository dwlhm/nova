package style

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/core/ast"
	"github.com/dwlhm/nova/internal/core/plan"
)

// CollectImports resolves style imports reachable from the module graph.
func CollectImports(partial plan.Partial, sources map[string]ast.RawFile) ([]ImportRef, []Diagnostic) {
	moduleSet := make(map[string]bool, len(partial.Modules))
	for _, module := range partial.Modules {
		moduleSet[module.Path] = true
	}

	refs := make([]ImportRef, 0)
	seen := make(map[string]ImportKind)
	diagnostics := make([]Diagnostic, 0)

	for _, module := range partial.Modules {
		file, ok := sources[module.Path]
		if !ok {
			continue
		}
		for _, decl := range file.StyleImports {
			if err := validateImportFrom(decl.From); err != nil {
				diagnostics = append(diagnostics, Diagnostic{
					Code:    err.code,
					Message: fmt.Sprintf("%s (requested by %s)", err.message, module.Path),
				})
				continue
			}
			resolved := ast.NormalizePath(path.Join(path.Dir(module.Path), decl.From))
			kind := ImportKind(decl.Kind)
			if err := validateStylePath(resolved, kind); err != nil {
				diagnostics = append(diagnostics, Diagnostic{
					Code:    err.code,
					Message: fmt.Sprintf("%s (requested by %s)", err.message, module.Path),
				})
				continue
			}
			if prior, ok := seen[resolved]; ok {
				if prior != kind {
					diagnostics = append(diagnostics, Diagnostic{
						Code:    "NVA-STYLE-003",
						Message: fmt.Sprintf("conflicting style registrations for %s", resolved),
					})
				}
				continue
			}
			seen[resolved] = kind
			refs = append(refs, ImportRef{
				Kind:           kind,
				Path:           resolved,
				RequestingFile: module.Path,
			})
		}
	}

	sort.Slice(refs, func(i, j int) bool {
		return refs[i].Path < refs[j].Path
	})
	return refs, diagnostics
}

type pathError struct {
	code    string
	message string
}

func validateImportFrom(from string) *pathError {
	from = strings.ReplaceAll(strings.TrimSpace(from), "\\", "/")
	if from == "" || strings.HasPrefix(from, "/") {
		return &pathError{code: "NVA-STYLE-001", message: "refusing unsafe stylesheet path " + from}
	}
	for _, segment := range strings.Split(path.Clean(from), "/") {
		if segment == ".." {
			return &pathError{code: "NVA-STYLE-001", message: "refusing unsafe stylesheet path " + from}
		}
	}
	return nil
}

func validateStylePath(cleanPath string, kind ImportKind) *pathError {
	if cleanPath == "" || strings.HasPrefix(cleanPath, "..") || path.IsAbs(cleanPath) {
		return &pathError{code: "NVA-STYLE-001", message: "refusing unsafe stylesheet path " + cleanPath}
	}
	switch kind {
	case ImportStyle:
		if !strings.HasSuffix(cleanPath, ".nova-style") {
			return &pathError{code: "NVA-STYLE-010", message: "style import must reference a .nova-style file: " + cleanPath}
		}
	case ImportStylesheet:
		if !strings.HasSuffix(cleanPath, ".css") {
			return &pathError{code: "NVA-STYLE-010", message: "stylesheet import must reference a .css file: " + cleanPath}
		}
	default:
		return &pathError{code: "NVA-STYLE-010", message: "unknown style import kind"}
	}
	return nil
}
