package artifact

import (
	"strings"
	"unicode"
)

type androidStyleSheet struct {
	Classes map[string]androidStyleRule
}

type androidStyleRule struct {
	Properties map[string]string
}

type androidResolvedStyle struct {
	Properties map[string]string
}

func newAndroidStyleSheet(assets []StyleAsset) androidStyleSheet {
	sheet := androidStyleSheet{Classes: make(map[string]androidStyleRule)}
	for _, asset := range assets {
		for _, rule := range parseAndroidStyleRules(asset.Content) {
			for _, className := range rule.Classes {
				current := sheet.Classes[className]
				if current.Properties == nil {
					current.Properties = make(map[string]string)
				}
				for name, value := range rule.Properties {
					current.Properties[name] = value
				}
				sheet.Classes[className] = current
			}
		}
	}
	return sheet
}

func (sheet androidStyleSheet) StyleForClassList(classList string) androidResolvedStyle {
	out := androidResolvedStyle{Properties: make(map[string]string)}
	for _, className := range strings.Fields(classList) {
		rule, ok := sheet.Classes[className]
		if !ok {
			continue
		}
		for name, value := range rule.Properties {
			out.Properties[name] = value
		}
	}
	return out
}

func (style androidResolvedStyle) Empty() bool {
	return len(style.Properties) == 0
}

func (style androidResolvedStyle) Value(names ...string) (string, bool) {
	for _, name := range names {
		value := strings.TrimSpace(style.Properties[name])
		if value != "" {
			return value, true
		}
	}
	return "", false
}

type androidParsedStyleRule struct {
	Classes    []string
	Properties map[string]string
}

func parseAndroidStyleRules(input string) []androidParsedStyleRule {
	clean := stripCSSComments(input)
	rules := make([]androidParsedStyleRule, 0)
	for pos := 0; pos < len(clean); {
		open := strings.IndexByte(clean[pos:], '{')
		if open < 0 {
			break
		}
		open += pos
		close := matchingCSSBrace(clean, open)
		if close < 0 {
			break
		}
		classes := androidStyleClasses(clean[pos:open])
		properties := androidStyleDeclarations(clean[open+1 : close])
		if len(classes) > 0 && len(properties) > 0 {
			rules = append(rules, androidParsedStyleRule{Classes: classes, Properties: properties})
		}
		pos = close + 1
	}
	return rules
}

func stripCSSComments(input string) string {
	var out strings.Builder
	for pos := 0; pos < len(input); {
		start := strings.Index(input[pos:], "/*")
		if start < 0 {
			out.WriteString(input[pos:])
			break
		}
		start += pos
		out.WriteString(input[pos:start])
		end := strings.Index(input[start+2:], "*/")
		if end < 0 {
			break
		}
		pos = start + 2 + end + 2
	}
	return out.String()
}

func androidStyleClasses(prelude string) []string {
	classes := make([]string, 0)
	for _, selector := range splitSelectorList(prelude) {
		if className, ok := androidStyleClassSelector(selector); ok {
			classes = append(classes, className)
		}
	}
	return classes
}

func androidStyleClassSelector(selector string) (string, bool) {
	trimmed := strings.TrimSpace(selector)
	if !strings.HasPrefix(trimmed, ".") {
		return "", false
	}
	name := strings.TrimPrefix(trimmed, ".")
	if name == "" {
		return "", false
	}
	for _, ch := range name {
		if ch == '-' || ch == '_' || unicode.IsLetter(ch) || unicode.IsDigit(ch) {
			continue
		}
		return "", false
	}
	return name, true
}

func androidStyleDeclarations(body string) map[string]string {
	out := make(map[string]string)
	for _, declaration := range splitAndroidStyleDeclarations(body) {
		name, value, ok := strings.Cut(declaration, ":")
		if !ok {
			continue
		}
		name = strings.ToLower(strings.TrimSpace(name))
		value = strings.TrimSpace(value)
		if name != "" && value != "" {
			out[name] = value
		}
	}
	return out
}

func splitAndroidStyleDeclarations(body string) []string {
	parts := make([]string, 0)
	start := 0
	depth := 0
	inString := false
	var quote rune
	for index, ch := range body {
		switch {
		case inString:
			if ch == quote {
				inString = false
			}
		case ch == '\'' || ch == '"':
			inString = true
			quote = ch
		case ch == '(' || ch == '[':
			depth++
		case ch == ')' || ch == ']':
			if depth > 0 {
				depth--
			}
		case ch == ';' && depth == 0:
			parts = append(parts, strings.TrimSpace(body[start:index]))
			start = index + 1
		}
	}
	parts = append(parts, strings.TrimSpace(body[start:]))
	return parts
}
