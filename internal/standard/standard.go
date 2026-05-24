package standard

import (
	"fmt"
	"sort"

	"github.com/dwlhm/nova/internal/diagnostic"
	"github.com/dwlhm/nova/internal/packages"
	"github.com/dwlhm/nova/internal/security"
	"github.com/dwlhm/nova/internal/view"
)

type Diagnostic = diagnostic.Diagnostic

type RendererPrimitive struct {
	Package     string
	Name        string
	Description string
	Props       []PrimitiveField
	Events      []PrimitiveEvent
}

type PrimitiveField struct {
	Name        string
	Type        string
	Optional    bool
	Description string
}

type PrimitiveEvent struct {
	Name        string
	Payload     string
	Description string
}

func OfficialPackages() []packages.Manifest {
	return []packages.Manifest{
		purePackage("@nova/core", map[string]string{
			"Option":    "src/Option.nova",
			"Result":    "src/Result.nova",
			"JsonValue": "src/JsonValue.nova",
		}),
		rendererPackage("@nova/ui", map[string]string{
			"text":    "src/text.nova",
			"button":  "src/button.nova",
			"image":   "src/image.nova",
			"list":    "src/list.nova",
			"surface": "src/surface.nova",
			"row":     "src/row.nova",
			"column":  "src/column.nova",
			"stack":   "src/stack.nova",
			"scroll":  "src/scroll.nova",
		}),
		rendererPackage("@nova/forms", map[string]string{
			"text_input":   "src/text_input.nova",
			"number_input": "src/number_input.nova",
			"toggle":       "src/toggle.nova",
			"slider":       "src/slider.nova",
			"select":       "src/select.nova",
			"form":         "src/form.nova",
		}),
		frameworkPackage("@nova/navigation", map[string]string{
			"Route":            "src/Route.nova",
			"NavigationAction": "src/NavigationAction.nova",
		}),
		frameworkPackage("@nova/app", map[string]string{
			"events": "src/AppLifecycle.nova",
		}),
		envPackage("@env/storage", security.PermissionSet("storage.read", "storage.write"), "storage"),
		envPackage("@env/network", security.PermissionSet("network.request"), "network"),
		envPackage("@env/clipboard", security.PermissionSet("clipboard.read", "clipboard.write"), "clipboard"),
		envPackage("@env/notify", security.PermissionSet("notification.send"), "notify"),
		envPackage("@env/device", security.PermissionSet("device.info"), "device"),
	}
}

func RendererPrimitives() []RendererPrimitive {
	return []RendererPrimitive{
		{
			Package:     "@nova/ui",
			Name:        "text",
			Description: "Displays text content.",
			Props: []PrimitiveField{
				{Name: "value", Type: "string", Description: "Text content to render."},
				commonClassProp(),
				commonKeyProp(),
			},
		},
		{
			Package:     "@nova/ui",
			Name:        "button",
			Description: "Interactive press target that routes platform press events into scheduler events.",
			Props: []PrimitiveField{
				{Name: "label", Type: "string", Optional: true, Description: "Accessible label when text children are not enough."},
				{Name: "disabled", Type: "boolean", Optional: true, Description: "Disables press interaction."},
				commonClassProp(),
				commonKeyProp(),
			},
			Events: []PrimitiveEvent{
				{Name: "on_press", Payload: "void", Description: "Emitted when the button is pressed."},
			},
		},
		{
			Package:     "@nova/ui",
			Name:        "image",
			Description: "Displays an image asset or URL.",
			Props: []PrimitiveField{
				{Name: "src", Type: "string", Description: "Image source."},
				{Name: "alt", Type: "string", Optional: true, Description: "Accessible alternative text."},
				commonClassProp(),
				commonKeyProp(),
			},
		},
		{
			Package:     "@nova/ui",
			Name:        "list",
			Description: "Projects repeated children from an items binding.",
			Props: []PrimitiveField{
				{Name: "items", Type: "unknown[]", Description: "Array value projected into item scope."},
				commonClassProp(),
				commonKeyProp(),
			},
		},
		layoutPrimitive("surface", "Layout surface container."),
		layoutPrimitive("row", "Horizontal layout container."),
		layoutPrimitive("column", "Vertical layout container."),
		layoutPrimitive("stack", "Layered layout container."),
		layoutPrimitive("scroll", "Scrollable layout container."),
		{
			Package:     "@nova/navigation",
			Name:        "page",
			Description: "Route-projected fragment shown when path matches route state.",
			Props: []PrimitiveField{
				{Name: "path", Type: "string", Description: "Route path for this page fragment."},
				commonClassProp(),
				commonKeyProp(),
			},
		},
		{
			Package:     "@nova/forms",
			Name:        "text_input",
			Description: "Text entry primitive.",
			Props: []PrimitiveField{
				{Name: "value", Type: "string", Optional: true, Description: "Current text value."},
				{Name: "placeholder", Type: "string", Optional: true, Description: "Placeholder text."},
				{Name: "disabled", Type: "boolean", Optional: true, Description: "Disables editing."},
				commonClassProp(),
				commonKeyProp(),
			},
			Events: []PrimitiveEvent{
				{Name: "on_change", Payload: "string", Description: "Emitted with the new text value."},
				{Name: "on_submit", Payload: "string", Description: "Emitted when text entry is submitted."},
			},
		},
		{
			Package:     "@nova/forms",
			Name:        "number_input",
			Description: "Numeric entry primitive.",
			Props: []PrimitiveField{
				{Name: "value", Type: "number", Optional: true, Description: "Current numeric value."},
				{Name: "min", Type: "number", Optional: true, Description: "Minimum allowed value."},
				{Name: "max", Type: "number", Optional: true, Description: "Maximum allowed value."},
				{Name: "disabled", Type: "boolean", Optional: true, Description: "Disables editing."},
				commonClassProp(),
				commonKeyProp(),
			},
			Events: []PrimitiveEvent{
				{Name: "on_change", Payload: "number", Description: "Emitted with the new numeric value."},
			},
		},
		{
			Package:     "@nova/forms",
			Name:        "toggle",
			Description: "Boolean toggle primitive.",
			Props: []PrimitiveField{
				{Name: "checked", Type: "boolean", Optional: true, Description: "Current checked state."},
				{Name: "disabled", Type: "boolean", Optional: true, Description: "Disables interaction."},
				commonClassProp(),
				commonKeyProp(),
			},
			Events: []PrimitiveEvent{
				{Name: "on_change", Payload: "boolean", Description: "Emitted with the next checked state."},
			},
		},
		{
			Package:     "@nova/forms",
			Name:        "slider",
			Description: "Numeric slider primitive.",
			Props: []PrimitiveField{
				{Name: "value", Type: "number", Optional: true, Description: "Current numeric value."},
				{Name: "min", Type: "number", Optional: true, Description: "Minimum value."},
				{Name: "max", Type: "number", Optional: true, Description: "Maximum value."},
				{Name: "step", Type: "number", Optional: true, Description: "Increment step."},
				commonClassProp(),
				commonKeyProp(),
			},
			Events: []PrimitiveEvent{
				{Name: "on_change", Payload: "number", Description: "Emitted with the new slider value."},
			},
		},
		{
			Package:     "@nova/forms",
			Name:        "select",
			Description: "Selection primitive.",
			Props: []PrimitiveField{
				{Name: "value", Type: "unknown", Optional: true, Description: "Current selected value."},
				{Name: "options", Type: "unknown[]", Description: "Available options."},
				{Name: "disabled", Type: "boolean", Optional: true, Description: "Disables interaction."},
				commonClassProp(),
				commonKeyProp(),
			},
			Events: []PrimitiveEvent{
				{Name: "on_change", Payload: "unknown", Description: "Emitted with the selected value."},
			},
		},
		{
			Package:     "@nova/forms",
			Name:        "form",
			Description: "Form grouping primitive.",
			Props: []PrimitiveField{
				commonClassProp(),
				commonKeyProp(),
			},
			Events: []PrimitiveEvent{
				{Name: "on_submit", Payload: "void", Description: "Emitted when the form is submitted."},
				{Name: "on_validate", Payload: "unknown", Description: "Emitted when validation is requested."},
			},
		},
	}
}

func RendererPrimitivesFromPackages(graph packages.ResolvedGraph) []RendererPrimitive {
	merged := make(map[string]RendererPrimitive)
	for _, primitive := range RendererPrimitives() {
		merged[primitive.Name] = primitive
	}
	for _, extension := range graph.RendererExtensions {
		for _, primitive := range extension.Primitives {
			if primitive.Kind == "" {
				continue
			}
			if _, exists := merged[primitive.Kind]; exists {
				continue
			}
			merged[primitive.Kind] = rendererPrimitiveFromPackage(extension.Name, primitive)
		}
	}
	names := make([]string, 0, len(merged))
	for name := range merged {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]RendererPrimitive, 0, len(names))
	for _, name := range names {
		out = append(out, merged[name])
	}
	return out
}

func ValidateAccessibility(nodes []view.Node) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	for _, node := range nodes {
		diagnostics = append(diagnostics, validateNodeAccessibility(node)...)
	}
	return diagnostics
}

func layoutPrimitive(name string, description string) RendererPrimitive {
	return RendererPrimitive{
		Package:     "@nova/ui",
		Name:        name,
		Description: description,
		Props: []PrimitiveField{
			commonClassProp(),
			commonKeyProp(),
		},
	}
}

func commonClassProp() PrimitiveField {
	return PrimitiveField{Name: "class", Type: "string", Optional: true, Description: "Style class for renderer target styling."}
}

func commonKeyProp() PrimitiveField {
	return PrimitiveField{Name: "key", Type: "unknown", Optional: true, Description: "Stable identity for projected or repeated nodes."}
}

func validateNodeAccessibility(node view.Node) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	if isInteractive(node) && !hasAccessibleLabel(node) {
		diagnostics = append(diagnostics, Diagnostic{
			Code:     "NVA-RENDER-A11Y-001",
			Severity: diagnostic.SeverityWarning,
			Message:  fmt.Sprintf("interactive primitive %s should have accessible label", node.Kind),
		})
	}
	for _, child := range node.Children {
		diagnostics = append(diagnostics, validateNodeAccessibility(child)...)
	}
	return diagnostics
}

func isInteractive(node view.Node) bool {
	switch node.Kind {
	case "button", "input", "text_input", "number_input", "toggle", "slider", "select", "radio_group", "checkbox":
		return true
	}
	for slot := range node.Events {
		switch slot {
		case "on_press", "on_submit", "on_change", "on_validate":
			return true
		}
	}
	return false
}

func hasAccessibleLabel(node view.Node) bool {
	if binding, ok := node.Props["label"]; ok && binding.Text != "" {
		return true
	}
	for _, child := range node.Children {
		if child.Kind == "#text" || child.Kind == "text" {
			if binding, ok := child.Props["value"]; ok && binding.Text != "" {
				return true
			}
		}
	}
	return false
}

func purePackage(name string, exports map[string]string) packages.Manifest {
	return packages.Manifest{
		Name:    name,
		Version: "0.1.0",
		Types:   []packages.PackageType{packages.PackageSource},
		Exports: exports,
	}
}

func rendererPackage(name string, exports map[string]string) packages.Manifest {
	return packages.Manifest{
		Name:     name,
		Version:  "0.1.0",
		Types:    []packages.PackageType{packages.PackageRenderer},
		Exports:  exports,
		Renderer: packages.RendererManifest{Primitives: rendererPrimitiveContracts(name, exports)},
		Targets: map[string]packages.TargetAdapter{
			"web":     {Adapter: "platform/web/index.web.js"},
			"android": {Adapter: "platform/android/Index.android.java"},
		},
	}
}

func frameworkPackage(name string, exports map[string]string) packages.Manifest {
	return packages.Manifest{
		Name:    name,
		Version: "0.1.0",
		Types:   []packages.PackageType{packages.PackageSource},
		Exports: exports,
		Targets: map[string]packages.TargetAdapter{
			"web":     {Adapter: "platform/web/index.web.js"},
			"android": {Adapter: "platform/android/Index.android.java"},
		},
	}
}

func envPackage(name string, permissions security.PermissionMap, adapter string) packages.Manifest {
	return packages.Manifest{
		Name:        name,
		Version:     "0.1.0",
		Types:       []packages.PackageType{packages.PackageExternalCapability},
		Permissions: permissions,
		Targets: map[string]packages.TargetAdapter{
			"web":     {Adapter: "platform/web/" + adapter + ".web.js"},
			"android": {Adapter: "platform/android/" + adapter + ".android.java"},
		},
	}
}

func findPackage(manifests []packages.Manifest, name string) (packages.Manifest, bool) {
	for _, manifest := range manifests {
		if manifest.Name == name {
			return manifest, true
		}
	}
	return packages.Manifest{}, false
}

func rendererPrimitiveContracts(packageName string, exports map[string]string) map[string]packages.RendererPrimitive {
	contracts := make(map[string]packages.RendererPrimitive, len(exports))
	for _, primitive := range RendererPrimitives() {
		if primitive.Package != packageName {
			continue
		}
		contracts[primitive.Name] = packages.RendererPrimitive{
			Package:     primitive.Package,
			Kind:        primitive.Name,
			Description: primitive.Description,
			Props:       packageRendererFields(primitive.Props),
			Events:      packageRendererEvents(primitive.Events),
			Targets: map[string]packages.RendererTarget{
				"web":     {Strategy: "adapter"},
				"android": {Strategy: "adapter"},
			},
		}
	}
	for name := range exports {
		if _, ok := contracts[name]; ok {
			continue
		}
		contracts[name] = packages.RendererPrimitive{
			Package: packageName,
			Kind:    name,
			Targets: map[string]packages.RendererTarget{
				"web":     {Strategy: "adapter"},
				"android": {Strategy: "adapter"},
			},
		}
	}
	return contracts
}

func packageRendererFields(fields []PrimitiveField) []packages.RendererField {
	out := make([]packages.RendererField, 0, len(fields))
	for _, field := range fields {
		out = append(out, packages.RendererField{
			Name:        field.Name,
			Type:        field.Type,
			Optional:    field.Optional,
			Description: field.Description,
		})
	}
	return out
}

func packageRendererEvents(events []PrimitiveEvent) []packages.RendererEvent {
	out := make([]packages.RendererEvent, 0, len(events))
	for _, event := range events {
		out = append(out, packages.RendererEvent{
			Name:        event.Name,
			Payload:     event.Payload,
			Description: event.Description,
		})
	}
	return out
}

func rendererPrimitiveFromPackage(packageName string, primitive packages.RendererPrimitive) RendererPrimitive {
	return RendererPrimitive{
		Package:     packageName,
		Name:        primitive.Kind,
		Description: primitive.Description,
		Props:       primitiveFieldsFromPackage(primitive.Props),
		Events:      primitiveEventsFromPackage(primitive.Events),
	}
}

func primitiveFieldsFromPackage(fields []packages.RendererField) []PrimitiveField {
	out := make([]PrimitiveField, 0, len(fields))
	for _, field := range fields {
		out = append(out, PrimitiveField{
			Name:        field.Name,
			Type:        field.Type,
			Optional:    field.Optional,
			Description: field.Description,
		})
	}
	return out
}

func primitiveEventsFromPackage(events []packages.RendererEvent) []PrimitiveEvent {
	out := make([]PrimitiveEvent, 0, len(events))
	for _, event := range events {
		out = append(out, PrimitiveEvent{
			Name:        event.Name,
			Payload:     event.Payload,
			Description: event.Description,
		})
	}
	return out
}
