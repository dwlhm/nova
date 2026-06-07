package style

import (
	"fmt"
	"strings"
)

// PropertySpec documents one portable CSS property and how values are interpreted.
type PropertySpec struct {
	// Affects describes what the property controls in the box/text model.
	Affects  string
	Validate func(value string) bool
}

// PortablePropertySpecs maps web-canonical property names to semantics and validators.
var PortablePropertySpecs = map[string]PropertySpec{
	"color": {
		Affects: "foreground (text) color inside the element",
		Validate: func(value string) bool {
			return ParseColorField(strings.TrimSpace(value))
		},
	},
	"font-size": {
		Affects:  "text size (prefer sp in portable files for Android parity)",
		Validate: parseFontSize,
	},
	"font-weight": {
		Affects:  "stroke weight of glyphs (normal, bold, or 100–900)",
		Validate: parseFontWeight,
	},
	"text-align": {
		Affects:  "horizontal alignment of text within its line box",
		Validate: parseTextAlign,
	},
	"text-transform": {
		Affects:  "casing of text (none, uppercase, lowercase, capitalize)",
		Validate: parseTextTransform,
	},
	"line-height": {
		Affects:  "line box height as a unitless multiplier (no px in portable v1)",
		Validate: parseLineHeight,
	},
	"padding": {
		Affects: "inner spacing between content edge and border (1–4 lengths, CSS shorthand: top [right bottom left])",
		Validate: func(value string) bool {
			_, ok := ParseBoxShorthand(value)
			return ok
		},
	},
	"margin": {
		Affects: "outer spacing outside the border box (1–4 lengths, CSS shorthand: top [right bottom left])",
		Validate: func(value string) bool {
			_, ok := ParseBoxShorthand(value)
			return ok
		},
	},
	"min-height": {
		Affects: "minimum block height of the element",
		Validate: func(value string) bool {
			_, ok := ParseLength(value)
			return ok
		},
	},
	"background": {
		Affects:  "fill behind content (portable v1: solid color only, not images/gradients)",
		Validate: validateBackgroundValue,
	},
	"background-color": {
		Affects: "fill color behind content",
		Validate: func(value string) bool {
			return ParseColorField(strings.TrimSpace(value))
		},
	},
	"border": {
		Affects:  "shorthand for border-width plus border-color (e.g. \"1 #ccc\" or \"1 token.name\")",
		Validate: parseBorderShorthand,
	},
	"border-color": {
		Affects: "color of the border stroke",
		Validate: func(value string) bool {
			return ParseColorField(strings.TrimSpace(value))
		},
	},
	"border-width": {
		Affects: "thickness of the border stroke",
		Validate: func(value string) bool {
			_, ok := parseBorderWidthField(value)
			return ok
		},
	},
	"border-radius": {
		Affects: "corner rounding radius of the border box",
		Validate: func(value string) bool {
			_, ok := ParseLength(value)
			return ok
		},
	},
	"align-content": {
		Affects:  "cross-axis alignment in flex-like containers (limited Android support)",
		Validate: parseAlignContent,
	},
}

func portablePropertyNames() map[string]bool {
	out := make(map[string]bool, len(PortablePropertySpecs))
	for name := range PortablePropertySpecs {
		out[name] = true
	}
	return out
}

func validateBackgroundValue(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	fields := strings.Fields(trimmed)
	if len(fields) == 1 {
		return ParseColorField(fields[0])
	}
	for _, field := range fields {
		if ParseColorField(field) {
			return true
		}
	}
	return false
}

// ValidatePropertyValues checks resolved property values against portable semantics.
func ValidatePropertyValues(context string, properties map[string]string) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	for _, name := range names {
		spec, ok := PortablePropertySpecs[name]
		if !ok {
			continue
		}
		value := properties[name]
		if spec.Validate(value) {
			continue
		}
		diagnostics = append(diagnostics, Diagnostic{
			Code: "NVA-STYLE-010",
			Message: fmt.Sprintf(
				"%s has invalid %s: %q (%s)",
				context,
				name,
				value,
				spec.Affects,
			),
		})
	}
	return diagnostics
}
