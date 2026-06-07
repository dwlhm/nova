package expr

import (
	"fmt"
	"strings"

	"github.com/dwlhm/nova/internal/core/parser"
)

// EmitJSNovaExpr generates a JavaScript NovaExpr helper object for pure funcs.
func EmitJSNovaExpr(registry *Registry) (string, error) {
	if registry == nil || len(registry.funcs) == 0 {
		return "window.NovaExpr = window.NovaExpr || {};\n", nil
	}
	var builder strings.Builder
	builder.WriteString("window.NovaExpr = {\n")
	for _, name := range registry.Names() {
		fn, _ := registry.Lookup(name)
		paramNames := stringSet(fn.Params)
		body, err := LowerJSMethod(fn.Body, registry, paramNames)
		if err != nil {
			return "", fmt.Errorf("lower func %s: %w", name, err)
		}
		args := make([]string, 0, len(fn.Params))
		for _, param := range fn.Params {
			args = append(args, param)
		}
		builder.WriteString("  ")
		builder.WriteString(name)
		builder.WriteString("(")
		builder.WriteString(strings.Join(args, ", "))
		builder.WriteString(") {\n    return ")
		builder.WriteString(body)
		builder.WriteString(";\n  },\n")
	}
	builder.WriteString("};\n")
	return builder.String(), nil
}

// EmitJavaNovaExpr generates a Java NovaExpr class with static pure func methods.
func EmitJavaNovaExpr(packageName string, registry *Registry) (string, error) {
	var builder strings.Builder
	builder.WriteString("package ")
	builder.WriteString(packageName)
	builder.WriteString(";\n\n")
	builder.WriteString("public final class NovaExpr {\n")
	builder.WriteString("    private NovaExpr() {}\n\n")
	builder.WriteString("    public static Object field(Object base, String name) {\n")
	builder.WriteString("        if (!(base instanceof java.util.Map)) return null;\n")
	builder.WriteString("        return ((java.util.Map<?, ?>) base).get(name);\n")
	builder.WriteString("    }\n\n")
	if registry == nil {
		builder.WriteString("}\n")
		return builder.String(), nil
	}
	for _, name := range registry.Names() {
		fn, _ := registry.Lookup(name)
		paramNames := stringSet(fn.Params)
		body, err := LowerJava(fn.Body, registry, nil, paramNames)
		if err != nil {
			return "", fmt.Errorf("lower func %s: %w", name, err)
		}
		javaParams := make([]string, 0, len(fn.Params))
		for _, param := range fn.Params {
			javaParams = append(javaParams, "Object "+param)
		}
		builder.WriteString("    public static Object ")
		builder.WriteString(name)
		builder.WriteString("(")
		builder.WriteString(strings.Join(javaParams, ", "))
		builder.WriteString(") {\n")
		builder.WriteString("        return ")
		builder.WriteString(body)
		builder.WriteString(";\n")
		builder.WriteString("    }\n\n")
	}
	if err := writeJavaNovaExprInvoke(&builder, registry); err != nil {
		return "", err
	}
	builder.WriteString("}\n")
	return builder.String(), nil
}

func writeJavaNovaExprInvoke(builder *strings.Builder, registry *Registry) error {
	names := registry.Names()
	if len(names) == 0 {
		return nil
	}
	builder.WriteString("    public static Object invoke(String name, Object[] args) {\n")
	builder.WriteString("        switch (name) {\n")
	for _, name := range names {
		fn, ok := registry.Lookup(name)
		if !ok {
			continue
		}
		builder.WriteString("            case ")
		builder.WriteString(quoteCodeString(name))
		builder.WriteString(":\n")
		builder.WriteString("                return ")
		builder.WriteString(name)
		builder.WriteString("(")
		argRefs := make([]string, 0, len(fn.Params))
		for index := range fn.Params {
			argRefs = append(argRefs, fmt.Sprintf("args[%d]", index))
		}
		builder.WriteString(strings.Join(argRefs, ", "))
		builder.WriteString(");\n")
	}
	builder.WriteString("            default:\n")
	builder.WriteString("                throw new IllegalStateException(\"unknown NovaExpr.\" + name);\n")
	builder.WriteString("        }\n")
	builder.WriteString("    }\n\n")
	return nil
}

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

// CollectFuncFiles returns parser files that declare funcs from a source map.
func CollectFuncFiles(sources map[string]parser.File) []parser.File {
	files := make([]parser.File, 0)
	for _, file := range sources {
		if len(file.Funcs) > 0 {
			files = append(files, file)
		}
	}
	return files
}
