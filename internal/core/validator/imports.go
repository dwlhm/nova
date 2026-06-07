package validator

import (
	"strings"

	"github.com/dwlhm/nova/internal/core/parser"
	novatypes "github.com/dwlhm/nova/internal/core/types"
)

func buildTypeEnvironment(file parser.File, imports []parser.File) (novatypes.Environment, []Diagnostic) {
	env, typeDiagnostics := novatypes.BuildEnvironment(file)
	diagnostics := convertTypeDiagnostics(typeDiagnostics)
	for _, imported := range imports {
		importEnv, importTypeDiagnostics := novatypes.BuildEnvironment(imported)
		diagnostics = append(diagnostics, convertTypeDiagnostics(importTypeDiagnostics)...)
		env = novatypes.MergeEnvironments(env, importEnv)
	}
	return env, diagnostics
}

// CapabilityImportsForModule resolves capability import sources to parsed module files.
func CapabilityImportsForModule(modulePath string, file parser.File, modules map[string]parser.File) []parser.File {
	if len(modules) == 0 {
		return nil
	}
	imports := make([]parser.File, 0)
	seen := make(map[string]bool)
	for _, decl := range file.Imports {
		if decl.Kind != parser.ImportCapability {
			continue
		}
		resolved := resolveCapabilityImportPath(modulePath, decl.From)
		if resolved == "" || seen[resolved] {
			continue
		}
		imported, ok := modules[resolved]
		if !ok {
			continue
		}
		seen[resolved] = true
		imports = append(imports, imported)
	}
	return imports
}

func resolveCapabilityImportPath(modulePath string, from string) string {
	from = strings.TrimSpace(strings.Trim(from, `"'`))
	if from == "" {
		return ""
	}
	if strings.Contains(from, ":") || strings.HasPrefix(from, "/") {
		return strings.ReplaceAll(from, "\\", "/")
	}
	dir := modulePath
	if index := strings.LastIndex(modulePath, "/"); index >= 0 {
		dir = modulePath[:index]
	}
	if strings.HasPrefix(from, "./") {
		from = from[2:]
	}
	if dir == "" {
		return from
	}
	return dir + "/" + from
}
