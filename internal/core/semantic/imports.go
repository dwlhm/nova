package semantic

import (
	"github.com/dwlhm/nova/internal/core/parser"
	"github.com/dwlhm/nova/internal/core/validator"
)

// CapabilityImports resolves capability import sources for a module.
func CapabilityImports(modulePath string, file parser.File, modules map[string]parser.File) []parser.File {
	return validator.CapabilityImportsForModule(modulePath, file, modules)
}

// ValidateWithImports runs semantic validation with merged types from imported modules.
func ValidateWithImports(file parser.File, imports []parser.File) []Diagnostic {
	return convertDiagnostics(validator.ValidateWithImports(file, imports))
}
