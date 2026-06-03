package style

import (
	"fmt"
	"sort"
	"strings"
)

// EmitCSS renders a parsed sheet as standard CSS (before web app scoping).
func EmitCSS(sheet Sheet) string {
	var out strings.Builder
	classNames := make([]string, 0, len(sheet.Classes))
	for name := range sheet.Classes {
		classNames = append(classNames, name)
	}
	sort.Strings(classNames)
	for _, name := range classNames {
		rule := sheet.Classes[name]
		writeRule(&out, "."+name, rule.Properties)
	}
	for _, state := range sheet.States {
		selector := "." + state.Class + ":" + state.Pseudo
		writeRule(&out, selector, state.Properties)
	}
	return out.String()
}

func writeRule(out *strings.Builder, selector string, properties map[string]string) {
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return
	}
	out.WriteString(selector)
	out.WriteString(" {\n")
	for _, name := range names {
		out.WriteString("  ")
		out.WriteString(name)
		out.WriteString(": ")
		out.WriteString(properties[name])
		out.WriteString(";\n")
	}
	out.WriteString("}\n")
}

// ValidatePortableSubset checks properties for cross-target sheets.
func ValidatePortableSubset(sheet Sheet) []Diagnostic {
	allowed := portableProperties()
	diagnostics := make([]Diagnostic, 0)
	check := func(context string, properties map[string]string) {
		for name := range properties {
			if !allowed[name] {
				diagnostics = append(diagnostics, Diagnostic{
					Code:    "NVA-STYLE-009",
					Message: fmt.Sprintf("%s uses unsupported property %q", context, name),
				})
			}
		}
	}
	for className, rule := range sheet.Classes {
		check("class "+className, rule.Properties)
	}
	for _, state := range sheet.States {
		check(fmt.Sprintf("state %s %s", state.Class, state.Pseudo), state.Properties)
	}
	return diagnostics
}

func portableProperties() map[string]bool {
	names := []string{
		"color",
		"font-size",
		"font-weight",
		"text-align",
		"text-transform",
		"line-height",
		"padding",
		"min-height",
		"background",
		"background-color",
		"border",
		"border-color",
		"border-width",
		"border-radius",
		"align-content",
	}
	out := make(map[string]bool, len(names))
	for _, name := range names {
		out[name] = true
	}
	return out
}
