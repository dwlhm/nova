package project

import (
	"fmt"
	"sort"
)

func ValidateLayout(manifest Manifest, files []File) []Diagnostic {
	paths := normalizeFilePaths(files)
	exists := pathSet(paths)
	diagnostics := make([]Diagnostic, 0)

	if !exists["nova.toml"] {
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "NVA-LAYOUT-001",
			Message: "project manifest nova.toml is required",
			Path:    "nova.toml",
		})
	}
	if manifest.Project.Entry == "" {
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "NVA-LAYOUT-002",
			Message: "project entry source must be declared in nova.toml",
		})
	} else if !exists[normalizePath(manifest.Project.Entry)] {
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "NVA-LAYOUT-003",
			Message: fmt.Sprintf("entry source %s is required", normalizePath(manifest.Project.Entry)),
			Path:    normalizePath(manifest.Project.Entry),
		})
	}

	for _, filePath := range paths {
		diagnostics = append(diagnostics, validateSourcePath(filePath)...)
		diagnostics = append(diagnostics, validateGeneratedPath(filePath)...)
		diagnostics = append(diagnostics, validateADRPath(filePath)...)
	}

	return diagnostics
}
func normalizeFilePaths(files []File) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		if file.Path == "" {
			continue
		}
		paths = append(paths, normalizePath(file.Path))
	}
	sort.Strings(paths)
	return paths
}

func pathSet(paths []string) map[string]bool {
	out := make(map[string]bool, len(paths))
	for _, filePath := range paths {
		out[filePath] = true
	}
	return out
}
