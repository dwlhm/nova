package androidcodegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/core/style"
	"github.com/dwlhm/nova/internal/provider/shared"
)

type androidStatefulSkinSpec struct {
	pathKey   string
	targetVar string
	nodeKind  string
	base      style.ResolvedStyle
	states    map[string]style.ResolvedStyle
}

func isLayoutProp(name string) bool {
	switch name {
	case "font-size", "font-weight", "text-transform", "text-align", "line-height", "padding", "margin", "min-height", "align-content":
		return true
	default:
		return false
	}
}

func layoutPropsOverriddenInStates(base style.ResolvedStyle, states map[string]style.ResolvedStyle) map[string]bool {
	overridden := make(map[string]bool)
	for _, state := range states {
		for prop, value := range state.Properties {
			if !isLayoutProp(prop) {
				continue
			}
			baseValue, ok := base.Properties[prop]
			if !ok || baseValue != value {
				overridden[prop] = true
			}
		}
	}
	return overridden
}

func androidStatesNeedLayoutRefresh(base style.ResolvedStyle, states map[string]style.ResolvedStyle) bool {
	return len(layoutPropsOverriddenInStates(base, states)) > 0
}

func (renderer *androidContractRenderer) renderStatefulSkinMethods() string {
	if len(renderer.statefulSkins) == 0 {
		return ""
	}
	targets := make([]string, 0, len(renderer.statefulSkins))
	for target := range renderer.statefulSkins {
		targets = append(targets, target)
	}
	sort.Strings(targets)

	var builder strings.Builder
	for _, target := range targets {
		spec := renderer.statefulSkins[target]
		builder.WriteString(fmt.Sprintf("    private static NovaStyle.StyleLayoutSpec layoutSpec_%s() {\n", spec.targetVar))
		builder.WriteString("        return new NovaStyle.StyleLayoutSpec(\n")
		builder.WriteString("            " + androidJavaPropertyMapLiteral(spec.base.Properties) + ",\n")
		builder.WriteString("            " + androidJavaStatesMapLiteral(spec.states) + "\n")
		builder.WriteString("        );\n")
		builder.WriteString("    }\n\n")
		builder.WriteString(fmt.Sprintf("    private void bindStatefulLayout_%s(View view) {\n", spec.targetVar))
		builder.WriteString("        NovaStyle.bindStatefulLayout(view, " + shared.QuoteCodeString(spec.nodeKind) + ", layoutSpec_" + spec.targetVar + "(), this::dp);\n")
		builder.WriteString("    }\n\n")
	}
	return builder.String()
}
