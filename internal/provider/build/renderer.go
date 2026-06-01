package build

import (
	"fmt"
	"sort"

	"github.com/dwlhm/nova/internal/packages"
	"github.com/dwlhm/nova/internal/project"
	"github.com/dwlhm/nova/internal/provider/standard"
)

const (
	rendererUnknownKindError          = "error"
	rendererUnknownKindWarn           = "warn"
	rendererUnknownKindPassthroughWeb = "passthrough_web"
)

type rendererSource string

const (
	rendererSourceBuiltin rendererSource = "built-in"
	rendererSourceLocal   rendererSource = "local dictionary"
	rendererSourcePackage rendererSource = "renderer package"
)

type rendererEntry struct {
	primitive RendererPrimitive
	source    rendererSource
}

func resolveRendererPlan(config project.RendererConfig, graph packages.ResolvedGraph, target string) (RendererPlan, []Diagnostic) {
	policy := string(config.UnknownKind)
	if policy == "" {
		policy = rendererUnknownKindError
	}
	entries := make(map[string]rendererEntry)
	diagnostics := make([]Diagnostic, 0)

	for _, primitive := range standard.RendererPrimitives() {
		entries[primitive.Name] = rendererEntry{primitive: rendererPrimitiveFromStandard(primitive), source: rendererSourceBuiltin}
	}

	for _, primitive := range config.Dictionary {
		if primitive.Kind == "" {
			continue
		}
		if existing, ok := entries[primitive.Kind]; ok && existing.source == rendererSourceBuiltin {
			diagnostics = append(diagnostics, rendererDiagnostic("NVA-RENDER-003", fmt.Sprintf("local renderer dictionary kind %s conflicts with built-in primitive", primitive.Kind)))
			continue
		}
		entries[primitive.Kind] = rendererEntry{primitive: rendererPrimitiveFromProject(primitive), source: rendererSourceLocal}
	}

	extensions := make([]RendererExtension, 0, len(graph.RendererExtensions))
	for _, extension := range graph.RendererExtensions {
		primitiveNames := make([]string, 0, len(extension.Primitives))
		for _, primitive := range extension.Primitives {
			kind := primitive.Kind
			if kind == "" {
				continue
			}
			primitiveNames = append(primitiveNames, kind)
			if adapterRequired(primitive, target) && extension.TargetAdapter == "" && primitiveAdapter(primitive, target) == "" {
				diagnostics = append(diagnostics, rendererDiagnostic("NVA-RENDER-004", fmt.Sprintf("renderer package %s primitive %s has no adapter for target %s", extension.Name, kind, target)))
			}
			if existing, ok := entries[kind]; ok {
				switch existing.source {
				case rendererSourceBuiltin:
					diagnostics = append(diagnostics, rendererDiagnostic("NVA-RENDER-003", fmt.Sprintf("renderer package %s kind %s conflicts with built-in primitive", extension.Name, kind)))
				case rendererSourcePackage:
					diagnostics = append(diagnostics, rendererDiagnostic("NVA-RENDER-003", fmt.Sprintf("renderer package %s kind %s conflicts with another renderer package", extension.Name, kind)))
				}
				continue
			}
			entries[kind] = rendererEntry{primitive: rendererPrimitiveFromPackage(primitive), source: rendererSourcePackage}
		}
		sort.Strings(primitiveNames)
		extensions = append(extensions, RendererExtension{
			Package:        extension.Name,
			Version:        extension.Version,
			AdapterPath:    extension.TargetAdapter,
			AdapterContent: extension.TargetAdapterContent,
			Primitives:     primitiveNames,
		})
	}

	kinds := make([]string, 0, len(entries))
	for kind := range entries {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	primitives := make([]RendererPrimitive, 0, len(kinds))
	for _, kind := range kinds {
		primitives = append(primitives, entries[kind].primitive)
	}
	return RendererPlan{UnknownKind: policy, Primitives: primitives, Extensions: extensions}, diagnostics
}

func rendererPrimitiveFromStandard(primitive standard.RendererPrimitive) RendererPrimitive {
	return RendererPrimitive{
		Package:     primitive.Package,
		Kind:        primitive.Name,
		Description: primitive.Description,
		Props:       rendererFieldsFromStandard(primitive.Props),
		Events:      rendererEventsFromStandard(primitive.Events),
		Targets: map[string]RendererTarget{
			"web":     {Strategy: "adapter"},
			"android": {Strategy: "adapter"},
		},
	}
}

func rendererPrimitiveFromProject(primitive project.RendererPrimitive) RendererPrimitive {
	return RendererPrimitive{
		Package:       "local",
		Kind:          primitive.Kind,
		Description:   primitive.Description,
		Props:         rendererFieldsFromProject(primitive.Props),
		Events:        rendererEventsFromProject(primitive.Events),
		AllowOverride: primitive.AllowOverride,
		Targets:       rendererTargetsFromProject(primitive.Targets),
	}
}

func rendererPrimitiveFromPackage(primitive packages.RendererPrimitive) RendererPrimitive {
	return RendererPrimitive{
		Package:       primitive.Package,
		Kind:          primitive.Kind,
		Description:   primitive.Description,
		Props:         rendererFieldsFromPackage(primitive.Props),
		Events:        rendererEventsFromPackage(primitive.Events),
		AllowOverride: primitive.AllowOverride,
		Targets:       rendererTargetsFromPackage(primitive.Targets),
	}
}

func rendererFieldsFromStandard(fields []standard.PrimitiveField) []RendererField {
	out := make([]RendererField, 0, len(fields))
	for _, field := range fields {
		out = append(out, RendererField{Name: field.Name, Type: field.Type, Optional: field.Optional, Description: field.Description})
	}
	return out
}

func rendererEventsFromStandard(events []standard.PrimitiveEvent) []RendererEvent {
	out := make([]RendererEvent, 0, len(events))
	for _, event := range events {
		out = append(out, RendererEvent{Name: event.Name, Payload: event.Payload, Description: event.Description})
	}
	return out
}

func rendererFieldsFromProject(fields []project.RendererField) []RendererField {
	out := make([]RendererField, 0, len(fields))
	for _, field := range fields {
		out = append(out, RendererField{Name: field.Name, Type: field.Type, Optional: field.Optional, Description: field.Description})
	}
	return out
}

func rendererEventsFromProject(events []project.RendererEvent) []RendererEvent {
	out := make([]RendererEvent, 0, len(events))
	for _, event := range events {
		out = append(out, RendererEvent{Name: event.Name, Payload: event.Payload, Description: event.Description})
	}
	return out
}

func rendererFieldsFromPackage(fields []packages.RendererField) []RendererField {
	out := make([]RendererField, 0, len(fields))
	for _, field := range fields {
		out = append(out, RendererField{Name: field.Name, Type: field.Type, Optional: field.Optional, Description: field.Description})
	}
	return out
}

func rendererEventsFromPackage(events []packages.RendererEvent) []RendererEvent {
	out := make([]RendererEvent, 0, len(events))
	for _, event := range events {
		out = append(out, RendererEvent{Name: event.Name, Payload: event.Payload, Description: event.Description})
	}
	return out
}

func rendererTargetsFromProject(targets map[string]project.RendererTarget) map[string]RendererTarget {
	out := make(map[string]RendererTarget, len(targets))
	for target, entry := range targets {
		out[target] = RendererTarget{Strategy: entry.Strategy, Adapter: entry.Adapter, Tag: entry.Tag, Delegate: entry.Delegate}
	}
	return out
}

func rendererTargetsFromPackage(targets map[string]packages.RendererTarget) map[string]RendererTarget {
	out := make(map[string]RendererTarget, len(targets))
	for target, entry := range targets {
		out[target] = RendererTarget{Strategy: entry.Strategy, Adapter: entry.Adapter, Tag: entry.Tag, Delegate: entry.Delegate}
	}
	return out
}

func adapterRequired(primitive packages.RendererPrimitive, target string) bool {
	targetEntry := primitive.Targets[target]
	strategy := targetEntry.Strategy
	if strategy == "" {
		strategy = "adapter"
	}
	return strategy == "adapter"
}

func primitiveAdapter(primitive packages.RendererPrimitive, target string) string {
	if primitive.Targets == nil {
		return ""
	}
	return primitive.Targets[target].Adapter
}

func rendererDiagnostic(code string, message string) Diagnostic {
	return Diagnostic{Code: code, Message: message}
}
