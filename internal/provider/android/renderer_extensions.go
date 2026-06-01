package android

import (
	"strings"

	"github.com/dwlhm/nova/internal/provider/build"
	"github.com/dwlhm/nova/internal/provider/shared"
)

func primitiveRegistry(config targetConfig) string {
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

func rendererExtensionsJava(extensions []build.RendererExtension, config targetConfig) string {
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

func rendererAdapterFiles(extensions []build.RendererExtension) []shared.File {
	files := make([]shared.File, 0, len(extensions))
	for _, extension := range extensions {
		if strings.TrimSpace(extension.AdapterContent) == "" {
			continue
		}
		files = append(files, shared.File{
			Path:    "build/android/renderer/" + shared.SafeRendererPackagePath(extension.Package) + "/" + shared.SafeRendererAdapterName(extension.AdapterPath),
			Content: extension.AdapterContent,
		})
	}
	return files
}
