package plan

import "github.com/dwlhm/nova/internal/core/ast"

// Core is the target-neutral core build plan (module graph and entry template).
type Core struct {
	Profile  string
	Entry    string
	Modules  []ModuleRef
	Template TemplateRef
}

// ModuleRef identifies a Nova source module in the build graph.
type ModuleRef struct {
	Path string
}

// TemplateRef selects the entry template used for ViewIR projection.
type TemplateRef struct {
	SourceFile string
	Target     string
	Selection  Selection
	Index      int
}

// Selection describes how the entry template was chosen.
type Selection string

const (
	SelectionExactTarget Selection = "exact_target"
	SelectionPolymorphic Selection = "polymorphic"
)

// ResolveInput configures core build planning from checked modules.
type ResolveInput struct {
	Profile        string
	Entry          string
	Modules        []ast.CheckedModule
	PackageExports map[string]string
}

// ResolveResult is the outcome of core build planning before provider enrichment.
type ResolveResult struct {
	Plan        Partial
	Diagnostics []Diagnostic
}

// Partial holds module graph and template selection prior to IR lowering.
type Partial struct {
	Profile  string
	Entry    string
	Modules  []ModuleRef
	Template TemplateRef
}
