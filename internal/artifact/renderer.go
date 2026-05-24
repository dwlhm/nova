package artifact

import (
	"fmt"
	"strings"

	"github.com/dwlhm/nova/internal/build"
	"github.com/dwlhm/nova/internal/diagnostic"
	"github.com/dwlhm/nova/internal/view"
)

type webRendererExtensionBundle struct {
	Enabled bool
	Content string
}

func validateRendererKinds(nodes []view.Node, renderer build.RendererPlan, target string) []Diagnostic {
	known := make(map[string]bool, len(renderer.Primitives))
	known["#text"] = true
	for _, primitive := range renderer.Primitives {
		known[primitive.Kind] = true
	}
	diagnostics := make([]Diagnostic, 0)
	var visit func([]view.Node)
	visit = func(nodes []view.Node) {
		for _, node := range nodes {
			if !known[node.Kind] {
				diagnostics = append(diagnostics, unknownKindDiagnostic(node.Kind, renderer.UnknownKind, target))
			}
			visit(node.Children)
		}
	}
	visit(nodes)
	return diagnostic.StableSort(diagnostics)
}

func unknownKindDiagnostic(kind string, policy string, target string) Diagnostic {
	if policy == "warn" || policy == "passthrough_web" && target == "web" {
		return Diagnostic{
			Code:     "NVA-RENDER-002",
			Severity: diagnostic.SeverityWarning,
			Message:  fmt.Sprintf("unknown view kind %s is allowed by renderer.unknown_kind policy for target %s", kind, target),
			Target:   target,
		}
	}
	return Diagnostic{
		Code:     "NVA-RENDER-001",
		Severity: diagnostic.SeverityError,
		Message:  fmt.Sprintf("unknown view kind %s for target %s", kind, target),
		Target:   target,
	}
}

func webRendererExtensions(extensions []build.RendererExtension) webRendererExtensionBundle {
	if len(extensions) == 0 {
		return webRendererExtensionBundle{}
	}
	var builder strings.Builder
	builder.WriteString("\"use strict\";\n")
	for _, extension := range extensions {
		if strings.TrimSpace(extension.AdapterContent) == "" {
			continue
		}
		builder.WriteString("\n// " + extension.Package + " " + extension.AdapterPath + "\n")
		builder.WriteString("(function(NovaRenderer) {\n")
		builder.WriteString(rewriteWebRegisterModule(extension.AdapterContent))
		builder.WriteString("\nif (typeof register === \"function\") register(NovaRenderer);\n")
		builder.WriteString("})(window.NovaRenderer);\n")
	}
	return webRendererExtensionBundle{Enabled: true, Content: builder.String()}
}

func rewriteWebRegisterModule(content string) string {
	content = strings.ReplaceAll(content, "export function register", "function register")
	content = strings.ReplaceAll(content, "export default function register", "function register")
	return content
}

func androidPrimitiveRegistry(config androidTargetConfig) string {
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

func androidRendererExtensions(extensions []build.RendererExtension, config androidTargetConfig) string {
	var builder strings.Builder
	builder.WriteString("package " + config.Namespace + ";\n\n")
	builder.WriteString("public final class NovaRendererExtensions {\n")
	builder.WriteString("    private NovaRendererExtensions() {}\n\n")
	builder.WriteString("    public static NovaPrimitiveRegistry register(NovaPrimitiveRegistry registry) {\n")
	builder.WriteString("        if (registry == null) registry = new NovaPrimitiveRegistry();\n")
	for _, extension := range extensions {
		builder.WriteString("        // " + extension.Package + " " + extension.AdapterPath + "\n")
		for _, primitive := range extension.Primitives {
			builder.WriteString("        registry.definePrimitive(" + quoteCodeString(primitive) + ", " + quoteCodeString(extension.Package) + ");\n")
		}
	}
	builder.WriteString("        return registry;\n")
	builder.WriteString("    }\n")
	builder.WriteString("}\n")
	return builder.String()
}

func androidRendererAdapterFiles(extensions []build.RendererExtension) []File {
	files := make([]File, 0, len(extensions))
	for _, extension := range extensions {
		if strings.TrimSpace(extension.AdapterContent) == "" {
			continue
		}
		files = append(files, File{
			Path:    "build/android/renderer/" + safeRendererPackagePath(extension.Package) + "/" + safeRendererAdapterName(extension.AdapterPath),
			Content: extension.AdapterContent,
		})
	}
	return files
}

func safeRendererPackagePath(name string) string {
	name = strings.TrimPrefix(name, "@")
	replacer := strings.NewReplacer("/", "_", "\\", "_", ".", "_", ":", "_")
	if name == "" {
		return "local"
	}
	return replacer.Replace(name)
}

func safeRendererAdapterName(path string) string {
	path = strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
	if path == "" {
		return "Register.android.java"
	}
	parts := strings.Split(path, "/")
	name := parts[len(parts)-1]
	if !strings.HasSuffix(name, ".java") {
		return "Register.android.java"
	}
	return name
}
