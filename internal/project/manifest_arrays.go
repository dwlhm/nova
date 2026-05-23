package project

import (
	"fmt"
	"strings"
)

func parseTargetStyles(value string, diagnostics *[]Diagnostic, lineNumber int) []string {
	styles, ok := parseStringArray(value, diagnostics, lineNumber)
	if !ok {
		*diagnostics = append(*diagnostics, Diagnostic{
			Code:    "NVA-PROJECT-004",
			Message: fmt.Sprintf("target styles must be a string array on line %d", lineNumber),
		})
		return nil
	}
	out := make([]string, 0, len(styles))
	for _, style := range styles {
		normalized := normalizePath(style)
		if normalized != "" {
			out = append(out, normalized)
		}
	}
	return out
}

func parseStringArray(value string, diagnostics *[]Diagnostic, lineNumber int) ([]string, bool) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
		return nil, false
	}

	body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
	if body == "" {
		return nil, true
	}

	parts := strings.Split(body, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		item := parseString(strings.TrimSpace(part), diagnostics, lineNumber)
		if item != "" {
			out = append(out, item)
		}
	}
	return out, true
}
