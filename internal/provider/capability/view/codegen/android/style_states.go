package androidcodegen

import (
	"strings"

	"github.com/dwlhm/nova/internal/core/style"
)

func androidJavaStyleApplication(target string, nodeKind string, base androidResolvedStyle, states map[string]androidResolvedStyle, indent string) string {
	var builder strings.Builder
	if androidJavaTextStyleTarget(nodeKind) {
		builder.WriteString(androidJavaTextColorWithStates(target, base, states, indent))
	}
	overridden := layoutPropsOverriddenInStates(base, states)
	builder.WriteString(androidJavaStaticLayout(target, nodeKind, base, overridden, indent))
	builder.WriteString(androidJavaBackgroundWithStates(target, base, states, indent))
	return builder.String()
}

func androidJavaTextColorWithStates(target string, base androidResolvedStyle, states map[string]androidResolvedStyle, indent string) string {
	baseColor, hasBase := androidCSSColorProperty(base, "color")
	type stateColor struct {
		attrs string
		color string
	}
	entries := make([]stateColor, 0)
	for _, pseudo := range style.AndroidPseudoOrder {
		stateStyle, ok := states[pseudo]
		if !ok {
			continue
		}
		color, ok := androidCSSColorProperty(stateStyle, "color")
		if !ok {
			continue
		}
		attrs, ok := style.AndroidViewStateAttrs(pseudo)
		if !ok {
			continue
		}
		entries = append(entries, stateColor{attrs: attrs, color: color})
	}
	if len(entries) == 0 {
		if !hasBase {
			return ""
		}
		return indent + target + ".setTextColor(" + baseColor + ");\n"
	}
	var builder strings.Builder
	builder.WriteString(indent + target + ".setTextColor(new ColorStateList(\n")
	builder.WriteString(indent + "    new int[][] {\n")
	for _, entry := range entries {
		builder.WriteString(indent + "        new int[] { " + entry.attrs + " },\n")
	}
	builder.WriteString(indent + "        new int[] { }\n")
	builder.WriteString(indent + "    },\n")
	builder.WriteString(indent + "    new int[] {\n")
	for _, entry := range entries {
		builder.WriteString(indent + "        " + entry.color + ",\n")
	}
	if hasBase {
		builder.WriteString(indent + "        " + baseColor + "\n")
	} else {
		builder.WriteString(indent + "        " + entries[len(entries)-1].color + "\n")
	}
	builder.WriteString(indent + "    }\n")
	builder.WriteString(indent + "));\n")
	return builder.String()
}

func androidJavaBackgroundWithStates(target string, base androidResolvedStyle, states map[string]androidResolvedStyle, indent string) string {
	type stateEntry struct {
		pseudo string
		style  androidResolvedStyle
	}
	entries := make([]stateEntry, 0)
	for _, pseudo := range style.AndroidPseudoOrder {
		stateStyle, ok := states[pseudo]
		if !ok || !androidResolvedStyleHasBackground(stateStyle) {
			continue
		}
		entries = append(entries, stateEntry{pseudo: pseudo, style: stateStyle})
	}
	if len(entries) == 0 {
		return androidJavaBackgroundDrawable(target, base, indent)
	}

	listVar := target + "BgStates"
	var builder strings.Builder
	builder.WriteString(indent + "StateListDrawable " + listVar + " = new StateListDrawable();\n")
	for _, entry := range entries {
		attrs, ok := style.AndroidViewStateAttrs(entry.pseudo)
		if !ok {
			continue
		}
		drawableVar := target + "Bg" + androidPseudoJavaSuffix(entry.pseudo)
		builder.WriteString(androidJavaGradientDrawable(drawableVar, mergeAndroidStyle(base, entry.style), indent))
		builder.WriteString(indent + listVar + ".addState(new int[] { " + attrs + " }, " + drawableVar + ");\n")
	}
	defaultVar := target + "BgDefault"
	builder.WriteString(androidJavaGradientDrawable(defaultVar, base, indent))
	builder.WriteString(indent + listVar + ".addState(new int[] { }, " + defaultVar + ");\n")
	builder.WriteString(indent + target + ".setBackground(" + listVar + ");\n")
	return builder.String()
}

func androidJavaBackgroundDrawable(target string, style androidResolvedStyle, indent string) string {
	if !androidResolvedStyleHasBackground(style) {
		return ""
	}
	styleVar := target + "Style"
	var builder strings.Builder
	builder.WriteString(androidJavaGradientDrawable(styleVar, style, indent))
	builder.WriteString(indent + target + ".setBackground(" + styleVar + ");\n")
	return builder.String()
}

func androidJavaGradientDrawable(varName string, style androidResolvedStyle, indent string) string {
	background, hasBackground := androidCSSColorProperty(style, "background-color", "background")
	borderColor, hasBorderColor := androidCSSBorderColor(style)
	borderWidth, hasBorderWidth := androidCSSBorderWidth(style)
	radius, hasRadius := androidCSSIntProperty(style, "border-radius")
	if !hasBackground {
		background = "Color.TRANSPARENT"
	}
	if !hasBorderColor {
		borderColor = "Color.TRANSPARENT"
	}
	if !hasBorderWidth {
		borderWidth = 0
	}
	var builder strings.Builder
	builder.WriteString(indent + "GradientDrawable " + varName + " = new GradientDrawable();\n")
	builder.WriteString(indent + varName + ".setColor(" + background + ");\n")
	if hasBorderColor || hasBorderWidth {
		builder.WriteString(indent + varName + ".setStroke(dp(" + javaInt(borderWidth) + "), " + borderColor + ");\n")
	}
	if hasRadius {
		builder.WriteString(indent + varName + ".setCornerRadius(dp(" + javaInt(radius) + "));\n")
	}
	return builder.String()
}

func androidResolvedStyleHasBackground(style androidResolvedStyle) bool {
	if _, ok := androidCSSColorProperty(style, "background-color", "background"); ok {
		return true
	}
	if _, ok := androidCSSBorderColor(style); ok {
		return true
	}
	if _, ok := androidCSSBorderWidth(style); ok {
		return true
	}
	if _, ok := androidCSSIntProperty(style, "border-radius"); ok {
		return true
	}
	return false
}

func androidPseudoJavaSuffix(pseudo string) string {
	switch pseudo {
	case "disabled":
		return "Disabled"
	case "active":
		return "Active"
	case "focus":
		return "Focus"
	case "checked":
		return "Checked"
	default:
		return strings.ToUpper(pseudo[:1]) + pseudo[1:]
	}
}

func androidJavaTextStyleTarget(kind string) bool {
	return kind == "text" || kind == "#text" || kind == "button" || kind == "text_input" || kind == "number_input"
}

func androidJavaLinearStyleTarget(kind string) bool {
	return kind == "surface" || kind == "column"
}
