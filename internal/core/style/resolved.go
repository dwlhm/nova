package style

import (
	"strconv"
	"strings"
)

// ResolvedStyle is a validated property map ready for provider lowering.
type ResolvedStyle struct {
	Properties map[string]string
}

// NewResolvedStyle wraps a property map for target lowering.
func NewResolvedStyle(properties map[string]string) ResolvedStyle {
	return ResolvedStyle{Properties: properties}
}

func (style ResolvedStyle) Value(names ...string) (string, bool) {
	for _, name := range names {
		value := strings.TrimSpace(style.Properties[name])
		if value != "" {
			return value, true
		}
	}
	return "", false
}

func (style ResolvedStyle) Empty() bool {
	return len(style.Properties) == 0
}

// ColorLiteral returns the first portable color literal in a property value.
func (style ResolvedStyle) ColorLiteral(names ...string) (string, bool) {
	for _, name := range names {
		value, ok := style.Value(name)
		if !ok {
			continue
		}
		if literal, ok := firstColorLiteral(value); ok {
			return literal, true
		}
	}
	return "", false
}

// Padding expands padding shorthand (inner spacing, CSS top/right/bottom/left).
func (style ResolvedStyle) Padding() (BoxSides, bool) {
	return style.boxSides("padding")
}

// Margin expands margin shorthand (outer spacing, CSS top/right/bottom/left).
func (style ResolvedStyle) Margin() (BoxSides, bool) {
	return style.boxSides("margin")
}

func (style ResolvedStyle) boxSides(name string) (BoxSides, bool) {
	value, ok := style.Value(name)
	if !ok {
		return BoxSides{}, false
	}
	return ParseBoxShorthand(value)
}

// Length reads a single length property.
func (style ResolvedStyle) Length(name string) (int, bool) {
	value, ok := style.Value(name)
	if !ok {
		return 0, false
	}
	return ParseLength(value)
}

// BorderWidth reads border-width or the width field in border shorthand.
func (style ResolvedStyle) BorderWidth() (int, bool) {
	if width, ok := style.Length("border-width"); ok {
		return width, true
	}
	value, ok := style.Value("border")
	if !ok {
		return 0, false
	}
	for _, field := range strings.Fields(value) {
		if width, ok := ParseLength(field); ok {
			return width, true
		}
	}
	return 0, false
}

// BorderColorLiteral reads border-color or a color field in border shorthand.
func (style ResolvedStyle) BorderColorLiteral() (string, bool) {
	if literal, ok := style.ColorLiteral("border-color"); ok {
		return literal, true
	}
	value, ok := style.Value("border")
	if !ok {
		return "", false
	}
	return firstColorLiteral(value)
}

// Bold reports whether font-weight should render bold.
func (style ResolvedStyle) Bold() (bool, bool) {
	value, ok := style.Value("font-weight")
	if !ok {
		return false, false
	}
	return parseFontWeightBold(value), true
}

// LineHeightMultiplier returns a unitless line-height multiplier for portable v1.
func (style ResolvedStyle) LineHeightMultiplier() (float64, bool) {
	value, ok := style.Value("line-height")
	if !ok {
		return 0, false
	}
	cleaned := strings.TrimSpace(strings.TrimSuffix(value, "em"))
	if strings.HasSuffix(cleaned, "px") {
		return 0, false
	}
	return parseLineHeightFloat(cleaned)
}

// HasBackground reports fill or border paint properties.
func (style ResolvedStyle) HasBackground() bool {
	if _, ok := style.ColorLiteral("background-color", "background"); ok {
		return true
	}
	if _, ok := style.BorderColorLiteral(); ok {
		return true
	}
	if _, ok := style.BorderWidth(); ok {
		return true
	}
	if _, ok := style.Length("border-radius"); ok {
		return true
	}
	return false
}

func firstColorLiteral(value string) (string, bool) {
	for _, field := range strings.Fields(strings.TrimSpace(value)) {
		cleaned := strings.Trim(field, ",")
		if ParseColorField(cleaned) {
			return cleaned, true
		}
	}
	return "", false
}

func parseFontWeightBold(value string) bool {
	cleaned := strings.ToLower(strings.TrimSpace(value))
	if cleaned == "bold" || cleaned == "bolder" {
		return true
	}
	weight, err := parseIntString(cleaned)
	return err == nil && weight >= 600
}

func parseLineHeightFloat(cleaned string) (float64, bool) {
	number, err := parseFloatString(cleaned)
	if err != nil || number <= 0 {
		return 0, false
	}
	return number, true
}

// AndroidColorExpr turns a portable color literal into a Java Color expression.
func AndroidColorExpr(literal string) string {
	return "Color.parseColor(" + strconv.Quote(literal) + ")"
}

// AndroidPaddingArray returns Java setPadding arguments in Android order: left, top, right, bottom.
func AndroidPaddingArray(box BoxSides) [4]int {
	return [4]int{box.Left, box.Top, box.Right, box.Bottom}
}
