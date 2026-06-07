package androidcodegen

import (
	"strconv"
	"strings"

	"github.com/dwlhm/nova/internal/core/style"
	"github.com/dwlhm/nova/internal/provider/shared"
)

func androidJavaBlockContainerKind(kind string) bool {
	switch kind {
	case "surface", "page", "column", "row", "scroll":
		return true
	default:
		return false
	}
}

func androidJavaFillHorizontal(indent string, viewName string) string {
	return indent + "{\n" +
		indent + "    ViewGroup.LayoutParams raw = " + viewName + ".getLayoutParams();\n" +
		indent + "    ViewGroup.MarginLayoutParams params;\n" +
		indent + "    if (raw instanceof ViewGroup.MarginLayoutParams) {\n" +
		indent + "        params = (ViewGroup.MarginLayoutParams) raw;\n" +
		indent + "    } else if (raw != null) {\n" +
		indent + "        params = new ViewGroup.MarginLayoutParams(raw);\n" +
		indent + "    } else {\n" +
		indent + "        params = new ViewGroup.MarginLayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT);\n" +
		indent + "    }\n" +
		indent + "    params.width = ViewGroup.LayoutParams.MATCH_PARENT;\n" +
		indent + "    " + viewName + ".setLayoutParams(params);\n" +
		indent + "}\n"
}

func androidJavaRoutePatternTable(patterns []string) string {
	var builder strings.Builder
	builder.WriteString("    private List<String> routePatterns() {\n")
	if len(patterns) == 0 {
		builder.WriteString("        return Collections.emptyList();\n")
	} else {
		builder.WriteString("        return Arrays.asList(\n")
		for _, pattern := range patterns {
			builder.WriteString("            " + shared.QuoteCodeString(pattern) + ",\n")
		}
		builder.WriteString("            \"\"\n")
		builder.WriteString("        );\n")
	}
	builder.WriteString("    }\n\n")
	return builder.String()
}

func androidJavaInitialValue(expression string) string {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return "null"
	}
	if strings.HasPrefix(expression, "({") && strings.HasSuffix(expression, "})") {
		return androidJavaInitialRecord(expression)
	}
	if strings.HasPrefix(expression, "\"") && strings.HasSuffix(expression, "\"") {
		return shared.QuoteCodeString(androidJavaStringLiteralValue(expression))
	}
	if _, err := strconv.ParseFloat(expression, 64); err == nil {
		if strings.Contains(expression, ".") {
			return expression
		}
		return expression + ".0"
	}
	switch expression {
	case "true":
		return "Boolean.TRUE"
	case "false":
		return "Boolean.FALSE"
	case "null", "undefined":
		return "null"
	default:
		return "null"
	}
}

func androidJavaStringLiteralValue(expression string) string {
	value, err := strconv.Unquote(expression)
	if err != nil {
		return strings.Trim(expression, "\"")
	}
	return value
}

func androidJavaInitialRecord(expression string) string {
	body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(expression, "({"), "})"))
	if body == "" {
		return "record()"
	}
	fields := splitAndroidRecordFields(body)
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		name, value, ok := strings.Cut(field, ":")
		if !ok {
			continue
		}
		parts = append(parts, "entry("+shared.QuoteCodeString(strings.TrimSpace(name))+", "+androidJavaInitialValue(strings.TrimSpace(value))+")")
	}
	if len(parts) == 0 {
		return "record()"
	}
	return "record(" + strings.Join(parts, ", ") + ")"
}

func splitAndroidRecordFields(body string) []string {
	fields := make([]string, 0)
	start := 0
	depth := 0
	inString := false
	escaped := false
	for index, ch := range body {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch ch {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				fields = append(fields, strings.TrimSpace(body[start:index]))
				start = index + 1
			}
		}
	}
	fields = append(fields, strings.TrimSpace(body[start:]))
	return fields
}

func androidJavaStringList(values []string) string {
	if len(values) == 0 {
		return "Collections.emptyList()"
	}
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, shared.QuoteCodeString(value))
	}
	return "Arrays.asList(" + strings.Join(quoted, ", ") + ")"
}

func androidCSSColorProperty(resolved style.ResolvedStyle, names ...string) (string, bool) {
	literal, ok := resolved.ColorLiteral(names...)
	if !ok {
		return "", false
	}
	return style.AndroidColorExpr(literal), true
}

func androidCSSIntProperty(resolved style.ResolvedStyle, name string) (int, bool) {
	return resolved.Length(name)
}

func androidCSSBoxProperty(resolved style.ResolvedStyle, name string) ([4]int, bool) {
	var box style.BoxSides
	var ok bool
	switch name {
	case "padding":
		box, ok = resolved.Padding()
	case "margin":
		box, ok = resolved.Margin()
	default:
		return [4]int{}, false
	}
	if !ok {
		return [4]int{}, false
	}
	return style.AndroidPaddingArray(box), true
}

func androidJavaApplyMargins(viewExpr string, sides [4]int, indent string) string {
	return indent + "{\n" +
		indent + "    ViewGroup.LayoutParams raw = " + viewExpr + ".getLayoutParams();\n" +
		indent + "    ViewGroup.MarginLayoutParams params;\n" +
		indent + "    if (raw instanceof ViewGroup.MarginLayoutParams) {\n" +
		indent + "        params = (ViewGroup.MarginLayoutParams) raw;\n" +
		indent + "    } else if (raw != null) {\n" +
		indent + "        params = new ViewGroup.MarginLayoutParams(raw);\n" +
		indent + "    } else {\n" +
		indent + "        params = new ViewGroup.MarginLayoutParams(ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT);\n" +
		indent + "    }\n" +
		indent + "    params.setMargins(dp(" + javaInt(sides[0]) + "), dp(" + javaInt(sides[1]) + "), dp(" + javaInt(sides[2]) + "), dp(" + javaInt(sides[3]) + "));\n" +
		indent + "    " + viewExpr + ".setLayoutParams(params);\n" +
		indent + "}\n"
}

func androidCSSBorderWidth(resolved style.ResolvedStyle) (int, bool) {
	return resolved.BorderWidth()
}

func androidCSSBorderColor(resolved style.ResolvedStyle) (string, bool) {
	literal, ok := resolved.BorderColorLiteral()
	if !ok {
		return "", false
	}
	return style.AndroidColorExpr(literal), true
}

func androidCSSBoldWeight(resolved style.ResolvedStyle) bool {
	bold, ok := resolved.Bold()
	return ok && bold
}

func androidCSSLineHeight(resolved style.ResolvedStyle) (string, bool) {
	number, ok := resolved.LineHeightMultiplier()
	if !ok {
		return "", false
	}
	return strconv.FormatFloat(number, 'f', -1, 64), true
}

func androidPathKey(path []int) string {
	parts := make([]string, 0, len(path))
	for _, value := range path {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, ".")
}

func androidJavaVar(prefix string, path []int) string {
	suffix := androidJavaPathSuffix(path)
	if suffix == "" {
		return prefix
	}
	return prefix + suffix
}

func androidJavaPathSuffix(path []int) string {
	if len(path) == 0 {
		return ""
	}
	parts := make([]string, 0, len(path))
	for _, value := range path {
		parts = append(parts, strconv.Itoa(value))
	}
	return "_" + strings.Join(parts, "_")
}

func androidRouteConstName(path string) string {
	if strings.TrimSpace(path) == "*" || strings.TrimSpace(path) == "/*" {
		return "fallback"
	}
	name := strings.Trim(strings.ToLower(path), "/")
	if name == "" {
		return "root"
	}
	var builder strings.Builder
	lastUnderscore := false
	for _, ch := range name {
		switch {
		case ch >= 'a' && ch <= 'z':
			builder.WriteRune(ch)
			lastUnderscore = false
		case ch >= '0' && ch <= '9':
			builder.WriteRune(ch)
			lastUnderscore = false
		default:
			if !lastUnderscore {
				builder.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	out := strings.Trim(builder.String(), "_")
	if out == "" {
		return "route"
	}
	if out[0] >= '0' && out[0] <= '9' {
		return "route_" + out
	}
	return out
}

func javaInt(value int) string {
	return strconv.Itoa(value)
}
