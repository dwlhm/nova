package androidcodegen

import (
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/core/security"
	"github.com/dwlhm/nova/internal/provider/build"
	androidtarget "github.com/dwlhm/nova/internal/provider/capability/view/target/android"
	"github.com/dwlhm/nova/internal/provider/shared"
)

func RendererExtensionsJava(extensions []build.RendererExtension, config androidtarget.Config) string {
	var builder strings.Builder
	builder.WriteString("package " + config.Namespace + ";\n\n")
	builder.WriteString("public final class NovaRendererExtensions {\n")
	builder.WriteString("    private NovaRendererExtensions() {}\n\n")
	builder.WriteString("    public static NovaRenderer register(NovaRenderer renderer) {\n")
	builder.WriteString("        if (renderer == null) renderer = NovaRenderer.create();\n")
	for _, extension := range extensions {
		builder.WriteString("        // " + extension.Package + " " + extension.AdapterPath + "\n")
		for _, primitive := range extension.Primitives {
			builder.WriteString("        renderer.definePrimitive(" + shared.QuoteCodeString(primitive) + ", " + shared.QuoteCodeString(extension.Package) + ");\n")
		}
	}
	builder.WriteString("        return renderer;\n")
	builder.WriteString("    }\n")
	builder.WriteString("}\n")
	return builder.String()
}

func ExternalBindingsJava(operations []build.ResolvedExternalOperation, permissions []security.Permission, config androidtarget.Config) string {
	names := shared.ExternalOperationNames(operations)
	var builder strings.Builder
	builder.WriteString("package " + config.Namespace + ";\n\n")
	builder.WriteString("import java.util.Arrays;\n")
	builder.WriteString("import java.util.Collections;\n")
	builder.WriteString("import java.util.List;\n\n")
	builder.WriteString("public final class NovaExternalBindings {\n")
	builder.WriteString("    public static final class OperationSpec {\n")
	builder.WriteString("        public final String output;\n")
	builder.WriteString("        public final List<String> permissions;\n\n")
	builder.WriteString("        public OperationSpec(String output, List<String> permissions) {\n")
	builder.WriteString("            this.output = output;\n")
	builder.WriteString("            this.permissions = permissions == null ? Collections.emptyList() : permissions;\n")
	builder.WriteString("        }\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    private NovaExternalBindings() {}\n\n")
	builder.WriteString("    public static final List<String> PROJECT_PERMISSIONS = ")
	builder.WriteString(androidJavaPermissionList(permissions))
	builder.WriteString(";\n\n")
	builder.WriteString("    public static final List<String> OPERATIONS = Arrays.asList(\n")
	for _, name := range names {
		builder.WriteString("        ")
		builder.WriteString(shared.QuoteCodeString(name))
		builder.WriteString(",\n")
	}
	builder.WriteString("        \"\"\n")
	builder.WriteString("    );\n\n")
	builder.WriteString("    public static OperationSpec spec(String effectId) {\n")
	builder.WriteString("        if (effectId == null) return null;\n")
	builder.WriteString("        switch (effectId) {\n")
	seen := make(map[string]bool)
	for _, operation := range operations {
		id := operation.CapabilitySource + "#" + operation.Operation
		if seen[id] {
			continue
		}
		seen[id] = true
		builder.WriteString("            case " + shared.QuoteCodeString(id) + ":\n")
		builder.WriteString("                return new OperationSpec(")
		builder.WriteString(shared.QuoteCodeString(operation.Output) + ", ")
		builder.WriteString(androidJavaPermissionList(operation.Permissions))
		builder.WriteString(");\n")
	}
	builder.WriteString("            default:\n")
	builder.WriteString("                return null;\n")
	builder.WriteString("        }\n")
	builder.WriteString("    }\n")
	builder.WriteString("}\n")
	return builder.String()
}

func androidJavaPermissionList(permissions []security.Permission) string {
	if len(permissions) == 0 {
		return "Collections.emptyList()"
	}
	names := make([]string, 0, len(permissions))
	for _, permission := range permissions {
		names = append(names, string(permission))
	}
	sort.Strings(names)
	values := make([]string, 0, len(names))
	for _, name := range names {
		values = append(values, shared.QuoteCodeString(name))
	}
	return "Arrays.asList(" + strings.Join(values, ", ") + ")"
}
