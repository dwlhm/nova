package semantic

import (
	"github.com/dwlhm/nova/internal/core/ast"
	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/parser"
	"github.com/dwlhm/nova/internal/core/validator"
)

// Diagnostic is a semantic analysis error anchored to source.
type Diagnostic struct {
	Message string
	Token   lexer.Token
}

// Analyze runs semantic analysis on a raw AST and returns a checked AST when valid.
func Analyze(file ast.RawFile) (ast.CheckedFile, []Diagnostic) {
	diagnostics := convertDiagnostics(validator.Validate(file))
	if len(diagnostics) > 0 {
		return ast.CheckedFile{}, diagnostics
	}
	return ast.CheckedFile{File: file}, nil
}

// AnalyzeModule runs semantic analysis on a raw module.
func AnalyzeModule(module ast.RawModule, modules map[string]parser.File) (ast.CheckedModule, []Diagnostic) {
	imports := validator.CapabilityImportsForModule(module.Path, module.File, modules)
	diagnostics := convertDiagnostics(validator.ValidateWithImports(module.File, imports))
	if len(diagnostics) > 0 {
		return ast.CheckedModule{}, diagnostics
	}
	return ast.CheckedModule{Path: module.Path, File: ast.CheckedFile{File: module.File}}, nil
}

func convertDiagnostics(items []validator.Diagnostic) []Diagnostic {
	out := make([]Diagnostic, 0, len(items))
	for _, item := range items {
		out = append(out, Diagnostic{Message: item.Message, Token: item.Token})
	}
	return out
}
