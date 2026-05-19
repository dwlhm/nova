package standard

import (
	"fmt"

	"github.com/dwlhm/nova/internal/diagnostic"
	"github.com/dwlhm/nova/internal/packages"
	"github.com/dwlhm/nova/internal/security"
	"github.com/dwlhm/nova/internal/view"
)

type Diagnostic = diagnostic.Diagnostic

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

func ValidateAccessibility(nodes []view.Node) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	for _, node := range nodes {
		diagnostics = append(diagnostics, validateNodeAccessibility(node)...)
	}
	return diagnostics
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
		Name:    name,
		Version: "0.1.0",
		Types:   []packages.PackageType{packages.PackageRenderer},
		Exports: exports,
		Targets: map[string]packages.TargetAdapter{
			"web":     {Adapter: "platform/web/index.web.js"},
			"android": {Adapter: "platform/android/Index.android.kt"},
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
			"android": {Adapter: "platform/android/Index.android.kt"},
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
			"android": {Adapter: "platform/android/" + adapter + ".android.kt"},
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
