package format

import "strings"

const indentUnit = "  "

func Nova(source string) string {
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	formatted := make([]string, 0, len(lines))
	indent := 0

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			if len(formatted) > 0 && formatted[len(formatted)-1] != "" {
				formatted = append(formatted, "")
			}
			continue
		}

		indent = closeIndent(indent, line)
		formatted = append(formatted, strings.Repeat(indentUnit, indent)+line)
		indent = openIndent(indent, line)
	}

	if len(formatted) == 0 {
		return ""
	}
	for len(formatted) > 0 && formatted[len(formatted)-1] == "" {
		formatted = formatted[:len(formatted)-1]
	}
	return strings.Join(formatted, "\n") + "\n"
}

func closeIndent(indent int, line string) int {
	if indent == 0 {
		return 0
	}
	if strings.HasPrefix(line, "/|") || strings.HasPrefix(line, "}") {
		return indent - 1
	}
	return indent
}

func openIndent(indent int, line string) int {
	if opensNovaNode(line) || opensBlock(line) {
		return indent + 1
	}
	return indent
}

func opensNovaNode(line string) bool {
	return strings.HasPrefix(line, "<") &&
		strings.HasSuffix(line, ">") &&
		!strings.Contains(line, "/|")
}

func opensBlock(line string) bool {
	return strings.HasSuffix(line, "{")
}
