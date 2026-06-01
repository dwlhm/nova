package build

import (
	"github.com/dwlhm/nova/internal/core/compile"
)

// SourcesFromProgram converts a core compile result into provider source files.
func SourcesFromProgram(program compile.Program) []SourceFile {
	sources := make([]SourceFile, 0, len(program.Raw))
	for _, module := range program.Raw {
		sources = append(sources, SourceFile{Path: module.Path, File: module.File})
	}
	return sources
}

// FinalizeProgram re-lowers core IR using a resolved provider build plan.
func FinalizeProgram(program compile.Program, plan BuildPlan, packageExports map[string]string) (compile.Program, []compile.Diagnostic) {
	irInput := IRLowerInput(plan, SourcesFromProgram(program))
	lowered, diagnostics := compile.Lower(compile.LowerInput{
		Profile:        irInput.Profile,
		Entry:          irInput.Entry,
		Modules:        program.Checked,
		PackageExports: packageExports,
		Permissions:    irInput.Permissions,
		Externals:      irInput.Externals,
	})
	if len(diagnostics) > 0 {
		return compile.Program{}, diagnostics
	}
	lowered.Raw = program.Raw
	return lowered, nil
}
