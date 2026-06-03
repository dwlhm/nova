package androidcodegen

import (
	"strings"

	"github.com/dwlhm/nova/internal/provider/build"
	androidtarget "github.com/dwlhm/nova/internal/provider/capability/view/target/android"
	"github.com/dwlhm/nova/internal/provider/shared"
)

func PrimitiveRegistryJava(config androidtarget.Config) string {
	return "package " + config.Namespace + ";\n\n" +
		"import java.util.LinkedHashMap;\n" +
		"import java.util.Map;\n\n" +
		"public final class NovaPrimitiveRegistry {\n" +
		"    private final Map<String, Object> primitives = new LinkedHashMap<>();\n\n" +
		"    public void definePrimitive(String kind, Object adapter) {\n" +
		"        if (kind == null || kind.isEmpty()) return;\n" +
		"        primitives.put(kind, adapter);\n" +
		"    }\n\n" +
		"    public boolean hasPrimitive(String kind) {\n" +
		"        return primitives.containsKey(kind);\n" +
		"    }\n" +
		"}\n"
}

func RendererExtensionsJava(extensions []build.RendererExtension, config androidtarget.Config) string {
	var builder strings.Builder
	builder.WriteString("package " + config.Namespace + ";\n\n")
	builder.WriteString("public final class NovaRendererExtensions {\n")
	builder.WriteString("    private NovaRendererExtensions() {}\n\n")
	builder.WriteString("    public static NovaPrimitiveRegistry register(NovaPrimitiveRegistry registry) {\n")
	builder.WriteString("        if (registry == null) registry = new NovaPrimitiveRegistry();\n")
	for _, extension := range extensions {
		builder.WriteString("        // " + extension.Package + " " + extension.AdapterPath + "\n")
		for _, primitive := range extension.Primitives {
			builder.WriteString("        registry.definePrimitive(" + shared.QuoteCodeString(primitive) + ", " + shared.QuoteCodeString(extension.Package) + ");\n")
		}
	}
	builder.WriteString("        return registry;\n")
	builder.WriteString("    }\n")
	builder.WriteString("}\n")
	return builder.String()
}

func ExternalBindingsJava(operations []build.ResolvedExternalOperation, config androidtarget.Config) string {
	names := shared.ExternalOperationNames(operations)
	var builder strings.Builder
	builder.WriteString("package " + config.Namespace + ";\n\nimport java.util.Arrays;\nimport java.util.List;\n\npublic final class NovaExternalBindings {\n    private NovaExternalBindings() {}\n    public static final List<String> OPERATIONS = Arrays.asList(\n")
	for _, name := range names {
		builder.WriteString("        ")
		builder.WriteString(shared.QuoteCodeString(name))
		builder.WriteString(",\n")
	}
	builder.WriteString("        \"\"\n    );\n}\n")
	return builder.String()
}
