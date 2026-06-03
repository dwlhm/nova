package androidcodegen

import (
	"strconv"
	"strings"

	"github.com/dwlhm/nova/internal/provider/shared"
)

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

func androidCSSColorProperty(style androidResolvedStyle, names ...string) (string, bool) {
	for _, name := range names {
		value, ok := style.Value(name)
		if !ok {
			continue
		}
		if color, ok := androidCSSColor(value); ok {
			return color, true
		}
	}
	return "", false
}

func androidCSSColor(value string) (string, bool) {
	for _, field := range strings.Fields(strings.TrimSpace(value)) {
		cleaned := strings.Trim(field, ",")
		if strings.HasPrefix(cleaned, "#") || androidCSSNamedColor(cleaned) {
			return "Color.parseColor(" + shared.QuoteCodeString(cleaned) + ")", true
		}
	}
	return "", false
}

func androidCSSNamedColor(value string) bool {
	switch strings.ToLower(value) {
	case "black", "blue", "cyan", "darkgray", "gray", "green", "lightgray", "magenta", "red", "white", "yellow":
		return true
	default:
		return false
	}
}

func androidCSSIntProperty(style androidResolvedStyle, name string) (int, bool) {
	value, ok := style.Value(name)
	if !ok {
		return 0, false
	}
	return androidCSSInt(value)
}

func androidCSSInt(value string) (int, bool) {
	cleaned := strings.TrimSpace(value)
	for _, suffix := range []string{"px", "dp", "sp"} {
		cleaned = strings.TrimSuffix(cleaned, suffix)
	}
	number, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, false
	}
	return int(number + 0.5), true
}

func androidCSSBoxProperty(style androidResolvedStyle, name string) ([4]int, bool) {
	value, ok := style.Value(name)
	if !ok {
		return [4]int{}, false
	}
	fields := strings.Fields(value)
	if len(fields) == 0 || len(fields) > 4 {
		return [4]int{}, false
	}
	values := make([]int, 0, len(fields))
	for _, field := range fields {
		size, ok := androidCSSInt(field)
		if !ok {
			return [4]int{}, false
		}
		values = append(values, size)
	}
	switch len(values) {
	case 1:
		return [4]int{values[0], values[0], values[0], values[0]}, true
	case 2:
		return [4]int{values[0], values[1], values[0], values[1]}, true
	case 3:
		return [4]int{values[0], values[1], values[2], values[1]}, true
	default:
		return [4]int{values[0], values[1], values[2], values[3]}, true
	}
}

func androidCSSBorderWidth(style androidResolvedStyle) (int, bool) {
	if width, ok := androidCSSIntProperty(style, "border-width"); ok {
		return width, true
	}
	value, ok := style.Value("border")
	if !ok {
		return 0, false
	}
	for _, field := range strings.Fields(value) {
		if width, ok := androidCSSInt(field); ok {
			return width, true
		}
	}
	return 0, false
}

func androidCSSBorderColor(style androidResolvedStyle) (string, bool) {
	if color, ok := androidCSSColorProperty(style, "border-color"); ok {
		return color, true
	}
	value, ok := style.Value("border")
	if !ok {
		return "", false
	}
	return androidCSSColor(value)
}

func androidCSSBoldWeight(value string) bool {
	cleaned := strings.TrimSpace(strings.ToLower(value))
	if cleaned == "bold" || cleaned == "bolder" {
		return true
	}
	weight, err := strconv.Atoi(cleaned)
	return err == nil && weight >= 600
}

func androidCSSLineHeight(style androidResolvedStyle) (string, bool) {
	value, ok := style.Value("line-height")
	if !ok {
		return "", false
	}
	cleaned := strings.TrimSpace(strings.TrimSuffix(value, "em"))
	if strings.HasSuffix(cleaned, "px") {
		return "", false
	}
	number, err := strconv.ParseFloat(cleaned, 64)
	if err != nil || number <= 0 {
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
