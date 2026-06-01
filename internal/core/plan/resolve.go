package plan

import (
	"fmt"
	"path"
	"strings"

	"github.com/dwlhm/nova/internal/core/ast"
	"github.com/dwlhm/nova/internal/core/capability"
	"github.com/dwlhm/nova/internal/core/parser"
	"github.com/dwlhm/nova/internal/project"
)

type Diagnostic struct {
	Code           string
	Message        string
	RequestingFile string
	ImportSource   string
}

func Resolve(input ResolveInput) ResolveResult {
	entry := ast.NormalizePath(input.Entry)
	sourceMap := ast.FileMap(input.Modules)
	diagnostics := make([]Diagnostic, 0)

	modules, moduleDiagnostics := resolveModuleGraph(entry, sourceMap, input.PackageExports)
	diagnostics = append(diagnostics, moduleDiagnostics...)
	diagnostics = append(diagnostics, validateModuleGraph(modules, sourceMap)...)

	partial := Partial{Profile: input.Profile, Entry: entry, Modules: moduleRefs(modules)}
	if entryFile, ok := sourceMap[entry]; ok {
		template, templateDiagnostics := selectTemplate(entry, input.Profile, entryFile.Templates)
		partial.Template = template
		diagnostics = append(diagnostics, templateDiagnostics...)
	}

	return ResolveResult{Plan: partial, Diagnostics: diagnostics}
}

func resolveModuleGraph(entry string, sources map[string]ast.RawFile, packageExports map[string]string) ([]string, []Diagnostic) {
	visited := make(map[string]bool)
	modules := make([]string, 0)
	diagnostics := make([]Diagnostic, 0)

	var visit func(string)
	visit = func(modulePath string) {
		modulePath = ast.NormalizePath(modulePath)
		if visited[modulePath] {
			return
		}
		visited[modulePath] = true

		file, ok := sources[modulePath]
		if !ok {
			diagnostics = append(diagnostics, Diagnostic{
				Code:           "NVA-TARGET-001",
				RequestingFile: modulePath,
				Message:        fmt.Sprintf("source module %s was not found", modulePath),
			})
			return
		}
		modules = append(modules, modulePath)
		for _, decl := range file.Imports {
			if isLocalImport(decl.From) {
				visit(ast.NormalizePath(path.Join(path.Dir(modulePath), decl.From)))
				continue
			}
			if project.IsPackageImport(decl.From) {
				resolved, ok := resolvePackageImport(decl.From, packageExports)
				if !ok {
					diagnostics = append(diagnostics, Diagnostic{
						Code:           "NVA-PKG-011",
						RequestingFile: modulePath,
						ImportSource:   decl.From,
						Message:        fmt.Sprintf("package import %s was not resolved requested by %s", decl.From, modulePath),
					})
					continue
				}
				visit(resolved)
			}
		}
	}

	visit(entry)
	return modules, diagnostics
}

func resolvePackageImport(from string, exports map[string]string) (string, bool) {
	if exports == nil {
		return "", false
	}
	if resolved, ok := exports[from]; ok {
		return ast.NormalizePath(resolved), true
	}
	return "", false
}

func validateModuleGraph(modules []string, sources map[string]ast.RawFile) []Diagnostic {
	moduleSet := make(map[string]bool, len(modules))
	for _, modulePath := range modules {
		moduleSet[modulePath] = true
	}
	graph := capability.ModuleGraph{}
	for _, modulePath := range modules {
		file, ok := sources[modulePath]
		if !ok {
			continue
		}
		deps := make([]capability.CapabilityRef, 0)
		for _, decl := range file.Imports {
			if !isLocalImport(decl.From) {
				continue
			}
			dep := ast.NormalizePath(path.Join(path.Dir(modulePath), decl.From))
			if moduleSet[dep] {
				deps = append(deps, capability.CapabilityRef(dep))
			}
		}
		graph[capability.CapabilityRef(modulePath)] = deps
	}
	diagnostics := make([]Diagnostic, 0)
	for _, cycle := range capability.FindModuleGraphCycles(graph) {
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "NVA-MODULE-001",
			Message: fmt.Sprintf("runtime module graph contains cycle %s", joinCapabilityCycle(cycle)),
		})
	}
	return diagnostics
}

func joinCapabilityCycle(cycle []capability.CapabilityRef) string {
	parts := make([]string, 0, len(cycle))
	for _, ref := range cycle {
		parts = append(parts, string(ref))
	}
	return strings.Join(parts, " -> ")
}

func selectTemplate(entry string, target string, templates []parser.TemplateDecl) (TemplateRef, []Diagnostic) {
	for i, template := range templates {
		if template.Target == target {
			return TemplateRef{
				SourceFile: entry,
				Target:     template.Target,
				Selection:  SelectionExactTarget,
				Index:      i,
			}, nil
		}
	}
	for i, template := range templates {
		if template.Target == "" {
			return TemplateRef{
				SourceFile: entry,
				Selection:  SelectionPolymorphic,
				Index:      i,
			}, nil
		}
	}
	return TemplateRef{}, []Diagnostic{{
		Code:           "NVA-TEMPLATE-001",
		RequestingFile: entry,
		Message:        fmt.Sprintf("no template for target %s requested by %s", target, entry),
	}}
}

func moduleRefs(modules []string) []ModuleRef {
	refs := make([]ModuleRef, 0, len(modules))
	for _, module := range modules {
		refs = append(refs, ModuleRef{Path: module})
	}
	return refs
}

func isLocalImport(source string) bool {
	return strings.HasPrefix(source, ".")
}
