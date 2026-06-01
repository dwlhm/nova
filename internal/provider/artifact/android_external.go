package artifact

import (
	"regexp"
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/provider/build"
	"github.com/dwlhm/nova/internal/provider/standard"
)

var (
	javaPackageDeclRE = regexp.MustCompile(`(?m)^\s*package\s+([a-zA-Z0-9_.]+)\s*;`)
	javaPublicClassRE = regexp.MustCompile(`(?m)^\s*public\s+(?:final\s+)?class\s+([A-Za-z0-9_]+)\b`)
)

func androidExternalAdapterFiles(operations []build.ResolvedExternalOperation, javaSourceRoot string, overrides map[string]string) []File {
	javaSourceRoot = strings.TrimSuffix(strings.TrimSpace(javaSourceRoot), "/")
	if javaSourceRoot == "" {
		javaSourceRoot = "build/android/app/src/main/java/nova/generated"
	}
	paths := make([]string, 0, len(operations))
	seen := make(map[string]bool)
	for _, operation := range operations {
		path := strings.TrimSpace(operation.Implementation.Path)
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		paths = append(paths, path)
	}
	sort.Strings(paths)
	files := make([]File, 0, len(paths)+1)
	namespace := strings.ReplaceAll(strings.TrimPrefix(javaSourceRoot, "build/android/app/src/main/java/"), "/", ".")
	for _, path := range paths {
		content, ok := androidAdapterContent(path, overrides)
		if !ok || strings.TrimSpace(content) == "" {
			continue
		}
		javaPath, ok := androidExternalAdapterJavaPath(path, content)
		if !ok {
			continue
		}
		files = append(files, File{
			Path:    javaPath,
			Content: content,
		})
	}
	if len(files) == 0 {
		return nil
	}
	files = append(files, File{
		Path:    javaSourceRoot + "/NovaExternalAdapters.java",
		Content: androidExternalAdaptersJava(namespace, paths),
	})
	return files
}

func androidExternalAdapterJavaPath(adapterPath, content string) (string, bool) {
	if className, ok := standard.AndroidPlatformAdapterClass(adapterPath); ok {
		return androidJavaSourcePath(className)
	}
	return androidJavaSourcePathFromContent(content)
}

func androidJavaSourcePath(className string) (string, bool) {
	className = strings.TrimSpace(className)
	lastDot := strings.LastIndex(className, ".")
	if lastDot <= 0 || lastDot >= len(className)-1 {
		return "", false
	}
	pkg := strings.ReplaceAll(className[:lastDot], ".", "/")
	simple := className[lastDot+1:]
	return "build/android/app/src/main/java/" + pkg + "/" + simple + ".java", true
}

func androidJavaSourcePathFromContent(content string) (string, bool) {
	pkgMatch := javaPackageDeclRE.FindStringSubmatch(content)
	classMatch := javaPublicClassRE.FindStringSubmatch(content)
	if len(pkgMatch) < 2 || len(classMatch) < 2 {
		return "", false
	}
	return "build/android/app/src/main/java/" + strings.ReplaceAll(pkgMatch[1], ".", "/") + "/" + classMatch[1] + ".java", true
}

func androidExternalAdaptersJava(namespace string, paths []string) string {
	var builder strings.Builder
	builder.WriteString("package " + namespace + ";\n\n")
	builder.WriteString("import android.content.Context;\n")
	builder.WriteString("import java.util.Map;\n")
	for _, path := range paths {
		if className, ok := standard.AndroidPlatformAdapterClass(path); ok {
			builder.WriteString("import " + className + ";\n")
		}
	}
	builder.WriteString("\npublic final class NovaExternalAdapters {\n")
	builder.WriteString("    private NovaExternalAdapters() {}\n\n")
	builder.WriteString("    public static Object invoke(Context context, String effectId, Map<String, Object> input) {\n")
	builder.WriteString("        String[] parts = effectId.split(\"#\", 2);\n")
	builder.WriteString("        if (parts.length != 2) throw new IllegalStateException(\"invalid effect id \" + effectId);\n")
	for _, path := range paths {
		sourceID, ok := standard.AndroidPlatformAdapterSourceID(path)
		className, classOK := standard.AndroidPlatformAdapterClass(path)
		if !ok || !classOK {
			continue
		}
		simple := className
		if index := strings.LastIndex(className, "."); index >= 0 {
			simple = className[index+1:]
		}
		builder.WriteString("        if (\"" + sourceID + "\".equals(parts[0])) {\n")
		builder.WriteString("            return " + simple + ".invoke(context, parts[1], input);\n")
		builder.WriteString("        }\n")
	}
	builder.WriteString("        throw new IllegalStateException(\"unsupported external operation \" + effectId);\n")
	builder.WriteString("    }\n")
	builder.WriteString("}\n")
	return builder.String()
}
