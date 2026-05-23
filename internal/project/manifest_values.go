package project

import (
	"fmt"
	"strconv"
	"strings"
)

func assignProjectValue(project *Project, key string, value string, diagnostics *[]Diagnostic, lineNumber int) {
	switch key {
	case "name":
		project.Name = parseString(value, diagnostics, lineNumber)
	case "version":
		project.Version = parseString(value, diagnostics, lineNumber)
	case "entry":
		project.Entry = normalizePath(parseString(value, diagnostics, lineNumber))
	}
}

func parseString(value string, diagnostics *[]Diagnostic, lineNumber int) string {
	unquoted, err := strconv.Unquote(value)
	if err == nil {
		return unquoted
	}
	*diagnostics = append(*diagnostics, Diagnostic{
		Code:    "NVA-PROJECT-003",
		Message: fmt.Sprintf("manifest value must be quoted string on line %d", lineNumber),
	})
	return strings.Trim(value, `"`)
}

func parseScalar(value string, diagnostics *[]Diagnostic, lineNumber int) string {
	value = strings.TrimSpace(value)
	unquoted, err := strconv.Unquote(value)
	if err == nil {
		return unquoted
	}
	if value == "" {
		*diagnostics = append(*diagnostics, Diagnostic{
			Code:    "NVA-PROJECT-003",
			Message: fmt.Sprintf("manifest value must not be empty on line %d", lineNumber),
		})
	}
	return value
}

func parseBool(value string) (bool, bool) {
	switch value {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
}

func stripComment(line string) string {
	if idx := strings.Index(line, "#"); idx >= 0 {
		return line[:idx]
	}
	return line
}
