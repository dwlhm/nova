package androidcodegen

import (
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/core/style"
	"github.com/dwlhm/nova/internal/provider/shared"
)

func NovaStyleRulesFromBundle(bundle style.Bundle) string {
	rules := style.MergedClassRules(style.NormalizeBundle(bundle, "android"))
	if len(rules) == 0 {
		return ""
	}
	names := make([]string, 0, len(rules))
	for name := range rules {
		names = append(names, name)
	}
	sort.Strings(names)

	var builder strings.Builder
	builder.WriteString("public final class NovaStyleRules {\n")
	builder.WriteString("    public static final java.util.Map<String, java.util.Map<String, String>> CLASS_RULES = buildClassRules();\n\n")
	builder.WriteString("    private NovaStyleRules() {}\n\n")
	builder.WriteString("    private static java.util.Map<String, java.util.Map<String, String>> buildClassRules() {\n")
	builder.WriteString("        java.util.Map<String, java.util.Map<String, String>> rules = new java.util.LinkedHashMap<>();\n")
	for _, name := range names {
		rule := rules[name]
		builder.WriteString("        rules.put(" + shared.QuoteCodeString(name) + ", ")
		builder.WriteString(androidJavaPropertyMapLiteral(rule.Properties))
		builder.WriteString(");\n")
	}
	builder.WriteString("        return rules;\n")
	builder.WriteString("    }\n")
	builder.WriteString("}\n")
	return builder.String()
}

func androidJavaPropertyMapLiteral(properties map[string]string) string {
	if len(properties) == 0 {
		return "java.util.Collections.emptyMap()"
	}
	if len(properties) == 1 {
		for name, value := range properties {
			return "java.util.Collections.singletonMap(" + shared.QuoteCodeString(name) + ", " + shared.QuoteCodeString(value) + ")"
		}
	}
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)
	var builder strings.Builder
	builder.WriteString("new java.util.LinkedHashMap<String, String>() {{ ")
	for index, name := range names {
		if index > 0 {
			builder.WriteString(" ")
		}
		builder.WriteString("put(" + shared.QuoteCodeString(name) + ", " + shared.QuoteCodeString(properties[name]) + ");")
	}
	builder.WriteString(" }}")
	return builder.String()
}

func androidJavaStatesMapLiteral(states map[string]style.ResolvedStyle) string {
	if len(states) == 0 {
		return "java.util.Collections.emptyMap()"
	}
	pseudos := make([]string, 0, len(states))
	for pseudo := range states {
		pseudos = append(pseudos, pseudo)
	}
	sort.Strings(pseudos)
	var builder strings.Builder
	builder.WriteString("new java.util.LinkedHashMap<String, java.util.Map<String, String>>() {{ ")
	for index, pseudo := range pseudos {
		if index > 0 {
			builder.WriteString(" ")
		}
		builder.WriteString("put(" + shared.QuoteCodeString(pseudo) + ", ")
		builder.WriteString(androidJavaPropertyMapLiteral(states[pseudo].Properties))
		builder.WriteString(");")
	}
	builder.WriteString(" }}")
	return builder.String()
}
