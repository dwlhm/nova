package style

import (
	"fmt"
	"strings"
)

// Diagnostic is a portable style parse/validate issue.
type Diagnostic struct {
	Code    string
	Message string
}

// ParseDocument parses a .nova-style file (format v1).
func ParseDocument(sourcePath string, content string) (Sheet, []Diagnostic) {
	sheet := Sheet{
		SourcePath: sourcePath,
		Tokens:     make(map[string]string),
		Classes:    make(map[string]ClassRule),
	}
	diagnostics := make([]Diagnostic, 0)
	scopeCount := 0

	var blockKind string
	var blockName string
	var blockPseudo string
	var blockLines []string

	flushBlock := func() {
		if blockKind == "" {
			return
		}
		props, propDiags := parseProperties(blockLines)
		diagnostics = append(diagnostics, propDiags...)
		switch blockKind {
		case "class":
			if _, exists := sheet.Classes[blockName]; exists {
				diagnostics = append(diagnostics, Diagnostic{
					Code:    "NVA-STYLE-004",
					Message: fmt.Sprintf("duplicate class %q", blockName),
				})
			} else {
				sheet.Classes[blockName] = ClassRule{Properties: props}
			}
		case "state":
			sheet.States = append(sheet.States, StateRule{
				Class:      blockName,
				Pseudo:     blockPseudo,
				Properties: props,
			})
		}
		blockKind = ""
		blockName = ""
		blockPseudo = ""
		blockLines = nil
	}

	lines := strings.Split(content, "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(stripComment(raw))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "scope:") {
			flushBlock()
			scopeCount++
			scope := strings.TrimSpace(strings.TrimPrefix(line, "scope:"))
			switch Scope(scope) {
			case ScopeGlobal, ScopeApp:
				sheet.Scope = Scope(scope)
			default:
				diagnostics = append(diagnostics, Diagnostic{
					Code:    "NVA-STYLE-010",
					Message: fmt.Sprintf("invalid scope %q", scope),
				})
			}
			continue
		}
		if strings.HasPrefix(line, "token ") {
			flushBlock()
			name, value, ok := strings.Cut(strings.TrimSpace(strings.TrimPrefix(line, "token")), "=")
			if !ok {
				diagnostics = append(diagnostics, Diagnostic{
					Code:    "NVA-STYLE-010",
					Message: "invalid token declaration",
				})
				continue
			}
			name = strings.TrimSpace(name)
			value = strings.TrimSpace(value)
			if name == "" || value == "" {
				diagnostics = append(diagnostics, Diagnostic{
					Code:    "NVA-STYLE-010",
					Message: "invalid token declaration",
				})
				continue
			}
			sheet.Tokens[name] = value
			continue
		}
		if strings.HasPrefix(line, "class ") && strings.HasSuffix(line, "{") {
			flushBlock()
			blockKind = "class"
			blockName = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "class"), "{"))
			if blockName == "" {
				diagnostics = append(diagnostics, Diagnostic{
					Code:    "NVA-STYLE-010",
					Message: "class name is required",
				})
			}
			continue
		}
		if strings.HasPrefix(line, "state ") && strings.HasSuffix(line, "{") {
			flushBlock()
			parts := strings.Fields(strings.TrimSuffix(strings.TrimPrefix(line, "state"), "{"))
			if len(parts) < 2 {
				diagnostics = append(diagnostics, Diagnostic{
					Code:    "NVA-STYLE-010",
					Message: "state declaration requires class and pseudo",
				})
				continue
			}
			blockKind = "state"
			blockName = parts[0]
			blockPseudo = parts[1]
			continue
		}
		if line == "}" {
			flushBlock()
			continue
		}
		if blockKind != "" {
			blockLines = append(blockLines, line)
			continue
		}
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "NVA-STYLE-010",
			Message: fmt.Sprintf("unexpected line %q", line),
		})
	}
	flushBlock()

	if scopeCount == 0 {
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "NVA-STYLE-010",
			Message: "scope header is required",
		})
	}
	if scopeCount > 1 {
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "NVA-STYLE-010",
			Message: "only one scope header is allowed per file",
		})
	}

	if len(diagnostics) == 0 {
		diagnostics = append(diagnostics, resolveSheetTokens(&sheet)...)
	}
	return sheet, diagnostics
}

func parseProperties(lines []string) (map[string]string, []Diagnostic) {
	out := make(map[string]string)
	diagnostics := make([]Diagnostic, 0)
	for _, line := range lines {
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "NVA-STYLE-010",
				Message: fmt.Sprintf("invalid property %q", line),
			})
			continue
		}
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		if name == "" || value == "" {
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "NVA-STYLE-010",
				Message: fmt.Sprintf("invalid property %q", line),
			})
			continue
		}
		out[name] = value
	}
	return out, diagnostics
}

func resolveSheetTokens(sheet *Sheet) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	resolving := make(map[string]bool)

	var resolveToken func(name string) (string, error)
	resolveToken = func(name string) (string, error) {
		if resolving[name] {
			return "", fmt.Errorf("cyclic token reference involving %q", name)
		}
		value, ok := sheet.Tokens[name]
		if !ok {
			return "", fmt.Errorf("unknown token %q", name)
		}
		if isLiteralValue(value) {
			return value, nil
		}
		if _, ok := sheet.Tokens[value]; !ok {
			return value, nil
		}
		resolving[name] = true
		defer delete(resolving, name)
		return resolveToken(value)
	}

	for name := range sheet.Tokens {
		resolving = make(map[string]bool)
		resolved, err := resolveToken(name)
		if err != nil {
			code := "NVA-STYLE-005"
			if strings.Contains(err.Error(), "cyclic") {
				code = "NVA-STYLE-006"
			}
			diagnostics = append(diagnostics, Diagnostic{Code: code, Message: err.Error()})
			continue
		}
		sheet.Tokens[name] = resolved
	}

	resolveProperty := func(value string) string {
		if isLiteralValue(value) {
			return value
		}
		if resolved, ok := sheet.Tokens[value]; ok {
			return resolved
		}
		return value
	}

	for className, rule := range sheet.Classes {
		for prop, value := range rule.Properties {
			rule.Properties[prop] = resolveProperty(value)
		}
		sheet.Classes[className] = rule
	}
	for index, state := range sheet.States {
		for prop, value := range state.Properties {
			state.Properties[prop] = resolveProperty(value)
		}
		sheet.States[index] = state
	}
	return diagnostics
}

func isLiteralValue(value string) bool {
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "#") {
		return true
	}
	if value[0] >= '0' && value[0] <= '9' {
		return true
	}
	return false
}

func stripComment(line string) string {
	if index := strings.Index(line, "//"); index >= 0 {
		return line[:index]
	}
	return line
}
