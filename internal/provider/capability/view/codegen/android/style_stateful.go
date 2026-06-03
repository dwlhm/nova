package androidcodegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/core/style"
)

type androidStatefulSkinSpec struct {
	pathKey   string
	targetVar string
	nodeKind  string
	base      androidResolvedStyle
	states    map[string]androidResolvedStyle
}

func isLayoutProp(name string) bool {
	switch name {
	case "font-size", "font-weight", "text-transform", "text-align", "line-height", "padding", "min-height", "align-content":
		return true
	default:
		return false
	}
}

func layoutPropsOverriddenInStates(base androidResolvedStyle, states map[string]androidResolvedStyle) map[string]bool {
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

func androidStatesNeedLayoutRefresh(base androidResolvedStyle, states map[string]androidResolvedStyle) bool {
	return len(layoutPropsOverriddenInStates(base, states)) > 0
}

func filterLayoutProperties(style androidResolvedStyle, skip map[string]bool) androidResolvedStyle {
	if len(skip) == 0 {
		return style
	}
	out := androidResolvedStyle{Properties: make(map[string]string)}
	for name, value := range style.Properties {
		if skip[name] {
			continue
		}
		out.Properties[name] = value
	}
	return out
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
		builder.WriteString(renderStatefulSkinMethods(renderer.statefulSkins[target]))
	}
	return builder.String()
}

func renderStatefulSkinMethods(spec androidStatefulSkinSpec) string {
	overridden := layoutPropsOverriddenInStates(spec.base, spec.states)
	if len(overridden) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("    private void bindStatefulLayout_%s(View view) {\n", spec.targetVar))
	builder.WriteString(fmt.Sprintf("        Runnable refresh = () -> applyLayoutSkin_%s(view);\n", spec.targetVar))
	builder.WriteString("        view.setOnTouchListener((v, event) -> {\n")
	builder.WriteString("            v.post(refresh);\n")
	builder.WriteString("            return false;\n")
	builder.WriteString("        });\n")
	builder.WriteString("        view.setOnFocusChangeListener((v, hasFocus) -> refresh.run());\n")
	builder.WriteString("        refresh.run();\n")
	builder.WriteString("    }\n\n")

	builder.WriteString(fmt.Sprintf("    private void applyLayoutSkin_%s(View view) {\n", spec.targetVar))
	builder.WriteString(androidJavaLayoutSkinLocals(spec, overridden, "        "))
	builder.WriteString(androidJavaLayoutSkinStateBranches(spec, overridden, "        "))
	builder.WriteString(androidJavaLayoutSkinApply("view", spec.nodeKind, overridden, "        "))
	builder.WriteString("    }\n\n")
	return builder.String()
}

type androidLayoutSkinField struct {
	name         string
	decl         string
	assign       func(androidResolvedStyle) string
	alwaysAssign bool
}

func androidJavaLayoutSkinLocals(spec androidStatefulSkinSpec, overridden map[string]bool, indent string) string {
	fields := androidLayoutSkinFields(spec.nodeKind, overridden)
	if len(fields) == 0 {
		return ""
	}
	var builder strings.Builder
	base := androidEffectiveLayoutStyle(spec.base, map[string]androidResolvedStyle{})
	for _, field := range fields {
		builder.WriteString(indent + field.decl + "\n")
		if assign := field.assign(base); assign != "" {
			builder.WriteString(indent + assign + "\n")
		}
	}
	builder.WriteString("\n")
	return builder.String()
}

func androidJavaLayoutSkinStateBranches(spec androidStatefulSkinSpec, overridden map[string]bool, indent string) string {
	fields := androidLayoutSkinFields(spec.nodeKind, overridden)
	if len(fields) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString(indent + "if (!view.isEnabled()) {\n")
	builder.WriteString(androidJavaLayoutSkinBranchAssignments(spec, "disabled", overridden, indent+"    "))
	builder.WriteString(indent + "} else if (view.isPressed()) {\n")
	builder.WriteString(androidJavaLayoutSkinBranchAssignments(spec, "active", overridden, indent+"    "))
	builder.WriteString(indent + "} else if (view.isFocused()) {\n")
	builder.WriteString(androidJavaLayoutSkinBranchAssignments(spec, "focus", overridden, indent+"    "))
	builder.WriteString(indent + "} else if (view.isSelected()) {\n")
	builder.WriteString(androidJavaLayoutSkinBranchAssignments(spec, "checked", overridden, indent+"    "))
	builder.WriteString(indent + "}\n\n")
	return builder.String()
}

func androidJavaLayoutSkinBranchAssignments(spec androidStatefulSkinSpec, pseudo string, overridden map[string]bool, indent string) string {
	stateStyle, ok := spec.states[pseudo]
	if !ok {
		return ""
	}
	effective := mergeAndroidStyle(spec.base, stateStyle)
	fields := androidLayoutSkinFields(spec.nodeKind, overridden)
	var builder strings.Builder
	for _, field := range fields {
		if assign := field.assign(effective); assign != "" {
			builder.WriteString(indent + assign + "\n")
		}
	}
	return builder.String()
}

func androidEffectiveLayoutStyle(base androidResolvedStyle, states map[string]androidResolvedStyle) androidResolvedStyle {
	out := androidResolvedStyle{Properties: make(map[string]string)}
	for name, value := range base.Properties {
		if isLayoutProp(name) {
			out.Properties[name] = value
		}
	}
	for _, pseudo := range style.AndroidPseudoOrder {
		state, ok := states[pseudo]
		if !ok {
			continue
		}
		for name, value := range state.Properties {
			if isLayoutProp(name) {
				out.Properties[name] = value
			}
		}
	}
	return out
}

func androidLayoutSkinFields(nodeKind string, overridden map[string]bool) []androidLayoutSkinField {
	fields := make([]androidLayoutSkinField, 0)
	if androidJavaTextStyleTarget(nodeKind) {
		if overridden["font-size"] {
			fields = append(fields, androidLayoutSkinField{
				name:   "font-size",
				decl:   "float fontSizeSp = -1f;",
				assign: androidLayoutAssignFontSize,
			})
		}
		if overridden["font-weight"] {
			fields = append(fields, androidLayoutSkinField{
				name:   "font-weight",
				decl:   "boolean bold = false;",
				assign: androidLayoutAssignFontWeight,
			})
		}
		if overridden["text-transform"] {
			fields = append(fields, androidLayoutSkinField{
				name:   "text-transform",
				decl:   "boolean allCaps = false;",
				assign: androidLayoutAssignTextTransform,
			})
		}
		if overridden["text-align"] {
			fields = append(fields, androidLayoutSkinField{
				name:         "text-align",
				decl:         "boolean textCenter = false;",
				assign:       androidLayoutAssignTextAlign,
				alwaysAssign: true,
			})
		}
		if overridden["line-height"] {
			fields = append(fields, androidLayoutSkinField{
				name:   "line-height",
				decl:   "float lineHeight = -1f;",
				assign: androidLayoutAssignLineHeight,
			})
		}
	}
	if overridden["padding"] {
		fields = append(fields, androidLayoutSkinField{
			name:   "padding",
			decl:   "int padLeft = 0, padTop = 0, padRight = 0, padBottom = 0;",
			assign: androidLayoutAssignPadding,
		})
	}
	if overridden["min-height"] {
		fields = append(fields, androidLayoutSkinField{
			name:   "min-height",
			decl:   "int minHeight = -1;",
			assign: androidLayoutAssignMinHeight,
		})
	}
	if androidJavaLinearStyleTarget(nodeKind) && overridden["align-content"] {
		fields = append(fields, androidLayoutSkinField{
			name:         "align-content",
			decl:         "boolean alignCenter = false;",
			assign:       androidLayoutAssignAlignContent,
			alwaysAssign: true,
		})
	}
	return fields
}

func androidLayoutAssignFontSize(style androidResolvedStyle) string {
	if size, ok := androidCSSIntProperty(style, "font-size"); ok {
		return fmt.Sprintf("fontSizeSp = %sf;", javaInt(size))
	}
	return ""
}

func androidLayoutAssignFontWeight(style androidResolvedStyle) string {
	if weight, ok := style.Value("font-weight"); ok {
		if androidCSSBoldWeight(weight) {
			return "bold = true;"
		}
		return "bold = false;"
	}
	return ""
}

func androidLayoutAssignTextTransform(style androidResolvedStyle) string {
	if transform, ok := style.Value("text-transform"); ok {
		if strings.EqualFold(transform, "uppercase") {
			return "allCaps = true;"
		}
		return "allCaps = false;"
	}
	return ""
}

func androidLayoutAssignTextAlign(style androidResolvedStyle) string {
	if align, ok := style.Value("text-align"); ok {
		if strings.EqualFold(align, "center") {
			return "textCenter = true;"
		}
		return "textCenter = false;"
	}
	return ""
}

func androidLayoutAssignLineHeight(style androidResolvedStyle) string {
	if lineHeight, ok := androidCSSLineHeight(style); ok {
		return fmt.Sprintf("lineHeight = %sf;", lineHeight)
	}
	return ""
}

func androidLayoutAssignPadding(style androidResolvedStyle) string {
	if padding, ok := androidCSSBoxProperty(style, "padding"); ok {
		return fmt.Sprintf("padLeft = dp(%s); padTop = dp(%s); padRight = dp(%s); padBottom = dp(%s);",
			javaInt(padding[3]), javaInt(padding[0]), javaInt(padding[1]), javaInt(padding[2]))
	}
	return ""
}

func androidLayoutAssignMinHeight(style androidResolvedStyle) string {
	if minHeight, ok := androidCSSIntProperty(style, "min-height"); ok {
		return fmt.Sprintf("minHeight = dp(%s);", javaInt(minHeight))
	}
	return ""
}

func androidLayoutAssignAlignContent(style androidResolvedStyle) string {
	if align, ok := style.Value("align-content"); ok {
		if strings.EqualFold(align, "center") {
			return "alignCenter = true;"
		}
		return "alignCenter = false;"
	}
	return ""
}

func androidJavaLayoutSkinApply(viewExpr, nodeKind string, overridden map[string]bool, indent string) string {
	var builder strings.Builder
	if androidJavaTextStyleTarget(nodeKind) {
		builder.WriteString(indent + "if (" + viewExpr + " instanceof TextView) {\n")
		builder.WriteString(indent + "    TextView text = (TextView) " + viewExpr + ";\n")
		if overridden["font-size"] {
			builder.WriteString(indent + "    if (fontSizeSp >= 0f) text.setTextSize(TypedValue.COMPLEX_UNIT_SP, fontSizeSp);\n")
		}
		if overridden["font-weight"] {
			builder.WriteString(indent + "    text.setTypeface(Typeface.DEFAULT, bold ? Typeface.BOLD : Typeface.NORMAL);\n")
		}
		if overridden["text-transform"] {
			builder.WriteString(indent + "    text.setAllCaps(allCaps);\n")
		}
		if overridden["text-align"] {
			builder.WriteString(indent + "    if (textCenter) text.setGravity(Gravity.CENTER);\n")
		}
		if overridden["line-height"] {
			builder.WriteString(indent + "    if (lineHeight >= 0f) text.setLineSpacing(0f, lineHeight);\n")
		}
		builder.WriteString(indent + "}\n")
	}
	if overridden["padding"] {
		builder.WriteString(indent + viewExpr + ".setPadding(padLeft, padTop, padRight, padBottom);\n")
	}
	if overridden["min-height"] {
		builder.WriteString(indent + "if (minHeight >= 0) " + viewExpr + ".setMinimumHeight(minHeight);\n")
	}
	if androidJavaLinearStyleTarget(nodeKind) && overridden["align-content"] {
		builder.WriteString(indent + "if (alignCenter && " + viewExpr + " instanceof LinearLayout) ((LinearLayout) " + viewExpr + ").setGravity(Gravity.CENTER_VERTICAL);\n")
	}
	return builder.String()
}

func androidJavaStaticLayout(target, nodeKind string, base androidResolvedStyle, skip map[string]bool, indent string) string {
	filtered := filterLayoutProperties(base, skip)
	var builder strings.Builder
	if androidJavaTextStyleTarget(nodeKind) {
		if size, ok := androidCSSIntProperty(filtered, "font-size"); ok {
			builder.WriteString(indent + target + ".setTextSize(TypedValue.COMPLEX_UNIT_SP, " + javaInt(size) + ");\n")
		}
		if weight, ok := filtered.Value("font-weight"); ok && androidCSSBoldWeight(weight) {
			builder.WriteString(indent + target + ".setTypeface(Typeface.DEFAULT, Typeface.BOLD);\n")
		}
		if transform, ok := filtered.Value("text-transform"); ok && strings.EqualFold(transform, "uppercase") {
			builder.WriteString(indent + target + ".setAllCaps(true);\n")
		}
		if align, ok := filtered.Value("text-align"); ok && strings.EqualFold(align, "center") {
			builder.WriteString(indent + target + ".setGravity(Gravity.CENTER);\n")
		}
		if lineHeight, ok := androidCSSLineHeight(filtered); ok {
			builder.WriteString(indent + target + ".setLineSpacing(0f, " + lineHeight + "f);\n")
		}
	}
	if padding, ok := androidCSSBoxProperty(filtered, "padding"); ok {
		builder.WriteString(indent + target + ".setPadding(dp(" + javaInt(padding[3]) + "), dp(" + javaInt(padding[0]) + "), dp(" + javaInt(padding[1]) + "), dp(" + javaInt(padding[2]) + "));\n")
	}
	if minHeight, ok := androidCSSIntProperty(filtered, "min-height"); ok {
		builder.WriteString(indent + target + ".setMinimumHeight(dp(" + javaInt(minHeight) + "));\n")
	}
	if alignContent, ok := filtered.Value("align-content"); ok && strings.EqualFold(alignContent, "center") && androidJavaLinearStyleTarget(nodeKind) {
		builder.WriteString(indent + target + ".setGravity(Gravity.CENTER_VERTICAL);\n")
	}
	return builder.String()
}
