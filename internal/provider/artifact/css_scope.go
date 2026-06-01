package artifact

import "strings"

func scopeCSS(input string, scopeSelector string) string {
	if strings.TrimSpace(input) == "" || strings.TrimSpace(scopeSelector) == "" {
		return input
	}

	var out strings.Builder
	for pos := 0; pos < len(input); {
		open := strings.IndexByte(input[pos:], '{')
		if open < 0 {
			out.WriteString(input[pos:])
			break
		}
		open += pos
		close := matchingCSSBrace(input, open)
		if close < 0 {
			out.WriteString(input[pos:])
			break
		}

		prelude := input[pos:open]
		body := input[open+1 : close]
		out.WriteString(scopedCSSRule(prelude, body, scopeSelector))
		pos = close + 1
	}
	return out.String()
}

func scopedCSSRule(prelude string, body string, scopeSelector string) string {
	trimmed := strings.TrimSpace(prelude)
	if trimmed == "" {
		return prelude + "{" + body + "}"
	}
	if scopesNestedRules(trimmed) {
		return prelude + "{" + scopeCSS(body, scopeSelector) + "}"
	}
	if strings.HasPrefix(trimmed, "@") {
		return prelude + "{" + body + "}"
	}
	return scopeSelectorList(prelude, scopeSelector) + "{" + body + "}"
}

func scopesNestedRules(prelude string) bool {
	return strings.HasPrefix(prelude, "@media") ||
		strings.HasPrefix(prelude, "@supports") ||
		strings.HasPrefix(prelude, "@container") ||
		strings.HasPrefix(prelude, "@layer")
}

func scopeSelectorList(prelude string, scopeSelector string) string {
	leading := leadingWhitespace(prelude)
	trailing := trailingWhitespace(prelude)
	body := strings.TrimSpace(prelude)
	selectors := splitSelectorList(body)
	scoped := make([]string, 0, len(selectors))
	for _, selector := range selectors {
		trimmed := strings.TrimSpace(selector)
		if trimmed == "" {
			continue
		}
		scoped = append(scoped, scopedSelector(trimmed, scopeSelector))
	}
	return leading + strings.Join(scoped, ", ") + trailing
}

func scopedSelector(selector string, scopeSelector string) string {
	if selector == ":root" {
		return scopeSelector
	}
	if strings.HasPrefix(selector, scopeSelector) {
		return selector
	}
	return scopeSelector + " " + selector
}

func splitSelectorList(input string) []string {
	parts := make([]string, 0)
	start := 0
	depth := 0
	for i, ch := range input {
		switch ch {
		case '(', '[':
			depth++
		case ')', ']':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, input[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, input[start:])
	return parts
}

func matchingCSSBrace(input string, open int) int {
	depth := 0
	for i := open; i < len(input); i++ {
		switch input[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func leadingWhitespace(input string) string {
	return input[:len(input)-len(strings.TrimLeft(input, " \t\r\n"))]
}

func trailingWhitespace(input string) string {
	return input[len(strings.TrimRight(input, " \t\r\n")):]
}
