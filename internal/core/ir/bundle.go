package ir

import (
	"github.com/dwlhm/nova/internal/core/capability"
	"github.com/dwlhm/nova/internal/core/parser"
	"github.com/dwlhm/nova/internal/core/routing"
	"github.com/dwlhm/nova/internal/core/view"
)

// Lower projects parser sources into a target-neutral application bundle.
func Lower(input LowerInput) (Bundle, []Diagnostic) {
	sourceMap := sourceFiles(input.Sources)
	selected, ok := selectedTemplate(input.Template, sourceMap)
	if !ok {
		return Bundle{}, []Diagnostic{{
			Code:    "NVA-RENDER-006",
			Message: "selected template is not available in source graph",
		}}
	}

	viewIR, viewDiagnostics := view.Project(selected, collectStateNames(input.Sources))
	diagnostics := make([]Diagnostic, 0, len(viewDiagnostics))
	for _, viewDiagnostic := range viewDiagnostics {
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "NVA-RENDER-005",
			Message: viewDiagnostic.Message,
		})
	}
	if len(diagnostics) > 0 {
		return Bundle{}, diagnostics
	}

	model := buildLoweredAppModel(input.Modules, sourceMap)
	manifests := capabilityManifests(input.Modules, sourceMap)
	app := materializeAppContract(input, sourceMap, model, viewIR, manifests)

	return Bundle{
		App:                 app,
		ViewIR:              viewIR,
		Modules:             modulePathsFromRefs(input.Modules),
		CapabilityManifests: manifests,
		Routes:              routeModels(viewIR),
	}, diagnostics
}

func selectedTemplate(ref TemplateRef, sources map[string]parser.File) (parser.TemplateDecl, bool) {
	file, ok := sources[ref.SourceFile]
	if !ok || ref.Index < 0 || ref.Index >= len(file.Templates) {
		return parser.TemplateDecl{}, false
	}
	return file.Templates[ref.Index], true
}

func sourceFiles(sources []SourceFile) map[string]parser.File {
	out := make(map[string]parser.File, len(sources))
	for _, source := range sources {
		out[source.Path] = source.File
	}
	return out
}

func capabilityManifests(modules []ModuleRef, sources map[string]parser.File) []capability.Manifest {
	manifests := make([]capability.Manifest, 0, len(modules))
	for _, module := range modules {
		file, ok := sources[module.Path]
		if !ok {
			continue
		}
		manifests = append(manifests, capability.BuildManifest(module.Path, file))
	}
	return manifests
}

func routeModels(viewIR view.IR) []RouteModel {
	routes := make([]RouteModel, 0, len(viewIR.Metadata.Pages))
	for _, page := range viewIR.Metadata.Pages {
		pattern, ok := staticBindingString(page.Path)
		if !ok {
			continue
		}
		description := routing.DescribePattern(pattern)
		routes = append(routes, RouteModel{
			Pattern:  description.Pattern,
			NodePath: cloneIntPath(page.NodePath),
			Params:   description.Params,
			Score:    description.Score,
			Fallback: description.Fallback,
		})
	}
	return routes
}
