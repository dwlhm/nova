// Package compile is the target-neutral Nova compiler core.
// Edge tooling such as internal/cli should call only [Compile] to run the core pipeline.
package compile

import (
	"sort"

	"github.com/dwlhm/nova/internal/core/ast"
	"github.com/dwlhm/nova/internal/core/ir"
	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/parser"
	"github.com/dwlhm/nova/internal/core/plan"
	"github.com/dwlhm/nova/internal/core/security"
	"github.com/dwlhm/nova/internal/core/semantic"
	"github.com/dwlhm/nova/internal/core/style"
)

// Diagnostic is a compiler diagnostic from any core pipeline stage.
type Diagnostic struct {
	Stage   Stage
	Code    string
	Message string
	Token   lexer.Token
}

// Stage identifies which core pipeline step produced a diagnostic.
type Stage string

const (
	StageLexer    Stage = "lexer"
	StageParser   Stage = "parser"
	StageSemantic Stage = "semantic"
	StagePlan     Stage = "plan"
	StageIR       Stage = "ir"
)

// Program is the output of the core compiler pipeline for a project slice.
type Program struct {
	Raw          []ast.RawModule
	Checked      []ast.CheckedModule
	Plan         plan.Core
	StyleImports []style.ImportRef
	NovaIR       ir.NovaIR
}

// SourceModule is an input Nova source unit for the core pipeline.
type SourceModule struct {
	Path    string
	Content string
}

// CompileInput configures end-to-end core compilation from lexer to core plan.
type CompileInput struct {
	Profile        string
	Entry          string
	Sources        []SourceModule
	PackageExports map[string]string
	Permissions    []security.Permission
	Externals      []ir.ResolvedExternal
}

// LowerInput configures IR lowering and core build planning.
type LowerInput struct {
	Profile        string
	Entry          string
	Modules        []ast.CheckedModule
	PackageExports map[string]string
	Permissions    []security.Permission
	Externals      []ir.ResolvedExternal
}

// Compile is the single entry point for core compilation.
// It runs lexer -> parser -> raw AST -> semantic -> checked AST -> IR lowering -> Nova IR -> core build plan.
func Compile(input CompileInput) (Program, []Diagnostic) {
	sources := make([]SourceModule, len(input.Sources))
	copy(sources, input.Sources)
	sort.Slice(sources, func(i int, j int) bool {
		return sources[i].Path < sources[j].Path
	})

	raw := make([]ast.RawModule, 0, len(sources))
	diagnostics := make([]Diagnostic, 0)
	for _, source := range sources {
		module, parseDiagnostics := ParseSource(source.Path, source.Content)
		diagnostics = append(diagnostics, parseDiagnostics...)
		if len(parseDiagnostics) > 0 {
			continue
		}
		raw = append(raw, module)
	}
	if len(diagnostics) > 0 {
		return Program{}, diagnostics
	}

	checked, checkDiagnostics := CheckModules(raw)
	if len(checkDiagnostics) > 0 {
		return Program{}, checkDiagnostics
	}

	lowered, lowerDiagnostics := Lower(LowerInput{
		Profile:        input.Profile,
		Entry:          input.Entry,
		Modules:        checked,
		PackageExports: input.PackageExports,
		Permissions:    input.Permissions,
		Externals:      input.Externals,
	})
	if len(lowerDiagnostics) > 0 {
		return Program{}, lowerDiagnostics
	}
	lowered.Raw = raw
	return lowered, nil
}

// ParseSource runs lexer and parser to produce a raw AST module.
func ParseSource(path string, content string) (ast.RawModule, []Diagnostic) {
	tokens := lexer.Tokenize(content)
	diagnostics := make([]Diagnostic, 0)
	for _, tok := range tokens {
		if tok.Type == lexer.ILLEGAL {
			diagnostics = append(diagnostics, Diagnostic{
				Stage:   StageLexer,
				Code:    "NVA-LEXER-001",
				Message: "illegal token " + tok.Literal,
				Token:   tok,
			})
		}
	}
	file, parserDiagnostics := parser.Parse(tokens)
	for _, item := range parserDiagnostics {
		diagnostics = append(diagnostics, Diagnostic{
			Stage:   StageParser,
			Code:    "NVA-PARSE-001",
			Message: item.Message,
			Token:   item.Token,
		})
	}
	if len(diagnostics) > 0 {
		return ast.RawModule{}, diagnostics
	}
	return ast.RawModule{Path: path, File: file}, nil
}

// CheckModules runs semantic analysis on raw modules.
func CheckModules(raw []ast.RawModule) ([]ast.CheckedModule, []Diagnostic) {
	modulesByPath := make(map[string]parser.File, len(raw))
	for _, module := range raw {
		modulesByPath[module.Path] = module.File
	}
	checked := make([]ast.CheckedModule, 0, len(raw))
	diagnostics := make([]Diagnostic, 0)
	for _, module := range raw {
		item, moduleDiagnostics := semantic.AnalyzeModule(module, modulesByPath)
		for _, diagnostic := range moduleDiagnostics {
			diagnostics = append(diagnostics, Diagnostic{
				Stage:   StageSemantic,
				Code:    "NVA-SEMANTIC-001",
				Message: diagnostic.Message,
				Token:   diagnostic.Token,
			})
		}
		if len(moduleDiagnostics) > 0 {
			continue
		}
		checked = append(checked, item)
	}
	if len(diagnostics) > 0 {
		return nil, diagnostics
	}
	return checked, nil
}

// Lower runs core IR lowering and assembles the core build plan.
func Lower(input LowerInput) (Program, []Diagnostic) {
	resolution := plan.Resolve(plan.ResolveInput{
		Profile:        input.Profile,
		Entry:          input.Entry,
		Modules:        input.Modules,
		PackageExports: input.PackageExports,
	})
	diagnostics := planDiagnostics(resolution.Diagnostics)
	if len(diagnostics) > 0 {
		return Program{}, diagnostics
	}

	irInput := irLowerInput(input, resolution.Plan)
	bundle, irDiags := ir.Lower(irInput)
	diagnostics = append(diagnostics, irDiagnostics(irDiags)...)
	if len(irDiags) > 0 {
		return Program{}, diagnostics
	}

	styleImports, styleDiagnostics := style.CollectImports(resolution.Plan, ast.FileMap(input.Modules))
	diagnostics = append(diagnostics, styleDiagnosticsToCompile(styleDiagnostics)...)
	if len(styleDiagnostics) > 0 {
		return Program{}, diagnostics
	}

	return Program{
		Checked: input.Modules,
		Plan: plan.Core{
			Profile:  resolution.Plan.Profile,
			Entry:    resolution.Plan.Entry,
			Modules:  resolution.Plan.Modules,
			Template: resolution.Plan.Template,
		},
		StyleImports: styleImports,
		NovaIR:       bundle,
	}, nil
}

func styleDiagnosticsToCompile(items []style.Diagnostic) []Diagnostic {
	out := make([]Diagnostic, 0, len(items))
	for _, item := range items {
		out = append(out, Diagnostic{
			Stage:   StagePlan,
			Code:    item.Code,
			Message: item.Message,
		})
	}
	return out
}

func irLowerInput(input LowerInput, partial plan.Partial) ir.LowerInput {
	sources := make([]ir.SourceFile, 0, len(input.Modules))
	for _, module := range input.Modules {
		sources = append(sources, ir.SourceFile{Path: module.Path, File: module.File.File})
	}
	modules := make([]ir.ModuleRef, 0, len(partial.Modules))
	for _, module := range partial.Modules {
		modules = append(modules, ir.ModuleRef{Path: module.Path})
	}
	return ir.LowerInput{
		Profile:     partial.Profile,
		Entry:       partial.Entry,
		Modules:     modules,
		Template:    ir.TemplateRef{SourceFile: partial.Template.SourceFile, Index: partial.Template.Index},
		Sources:     sources,
		Permissions: input.Permissions,
		Externals:   input.Externals,
	}
}

func planDiagnostics(items []plan.Diagnostic) []Diagnostic {
	out := make([]Diagnostic, 0, len(items))
	for _, item := range items {
		out = append(out, Diagnostic{
			Stage:   StagePlan,
			Code:    item.Code,
			Message: item.Message,
		})
	}
	return out
}

func irDiagnostics(items []ir.Diagnostic) []Diagnostic {
	out := make([]Diagnostic, 0, len(items))
	for _, item := range items {
		out = append(out, Diagnostic{
			Stage:   StageIR,
			Code:    item.Code,
			Message: item.Message,
		})
	}
	return out
}
