package shared

import "strings"

func SafeRendererPackagePath(name string) string {
	name = strings.TrimPrefix(name, "@")
	replacer := strings.NewReplacer("/", "_", "\\", "_", ".", "_", ":", "_")
	if name == "" {
		return "local"
	}
	return replacer.Replace(name)
}

func SafeRendererAdapterName(path string) string {
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
