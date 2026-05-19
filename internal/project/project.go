package project

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Project struct {
	Name    string
	Version string
	Entry   string
}

type Target struct {
	Renderer string
}

type PermissionMap map[string]bool

type Manifest struct {
	Project          Project
	Targets          map[string]Target
	Permissions      PermissionMap
	PermissionScopes map[string][]string
}

type File struct {
	Path string
}

type Diagnostic struct {
	Code    string
	Message string
	Path    string
}

var adrNamePattern = regexp.MustCompile(`^adr_[0-9]{3}_[a-z0-9]+(?:_[a-z0-9]+)*\.md$`)

func ParseManifest(input string) (Manifest, []Diagnostic) {
	manifest := Manifest{
		Targets:          make(map[string]Target),
		Permissions:      make(PermissionMap),
		PermissionScopes: make(map[string][]string),
	}
	diagnostics := make([]Diagnostic, 0)
	section := ""

	for lineNumber, raw := range strings.Split(input, "\n") {
		line := strings.TrimSpace(stripComment(raw))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "NVA-PROJECT-001",
				Message: fmt.Sprintf("invalid manifest entry on line %d", lineNumber+1),
			})
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		manifest = assignManifestValue(manifest, section, key, value, &diagnostics, lineNumber+1)
	}

	return manifest, diagnostics
}

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

func IsPackageImport(source string) bool {
	if !strings.HasPrefix(source, "@") {
		return false
	}
	parts := strings.Split(source, "/")
	return len(parts) >= 2 && len(parts[0]) > 1 && parts[1] != ""
}

func assignManifestValue(manifest Manifest, section string, key string, value string, diagnostics *[]Diagnostic, lineNumber int) Manifest {
	switch section {
	case "project":
		assignProjectValue(&manifest.Project, key, value, diagnostics, lineNumber)
	case "permissions":
		allowed, ok := parseBool(value)
		if !ok {
			*diagnostics = append(*diagnostics, Diagnostic{
				Code:    "NVA-PROJECT-002",
				Message: fmt.Sprintf("permission %s must be true or false on line %d", key, lineNumber),
			})
			return manifest
		}
		manifest.Permissions[key] = allowed
	default:
		if group, ok := strings.CutPrefix(section, "permissions."); ok {
			permission := group + "." + key
			scopes, ok := parseStringArray(value, diagnostics, lineNumber)
			if ok {
				manifest.Permissions[permission] = true
				manifest.PermissionScopes[permission] = scopes
				return manifest
			}
			allowed, ok := parseBool(value)
			if !ok {
				*diagnostics = append(*diagnostics, Diagnostic{
					Code:    "NVA-PROJECT-002",
					Message: fmt.Sprintf("permission %s must be true, false, or a string array on line %d", permission, lineNumber),
				})
				return manifest
			}
			manifest.Permissions[permission] = allowed
			return manifest
		}
		if targetID, ok := strings.CutPrefix(section, "targets."); ok {
			target := manifest.Targets[targetID]
			if key == "renderer" {
				target.Renderer = parseString(value, diagnostics, lineNumber)
			}
			manifest.Targets[targetID] = target
		}
	}
	return manifest
}

func assignProjectValue(project *Project, key string, value string, diagnostics *[]Diagnostic, lineNumber int) {
	switch key {
	case "name":
		project.Name = parseString(value, diagnostics, lineNumber)
	case "version":
		project.Version = parseString(value, diagnostics, lineNumber)
	case "entry":
		project.Entry = normalizePath(parseString(value, diagnostics, lineNumber))
	}
}

func parseString(value string, diagnostics *[]Diagnostic, lineNumber int) string {
	unquoted, err := strconv.Unquote(value)
	if err == nil {
		return unquoted
	}
	*diagnostics = append(*diagnostics, Diagnostic{
		Code:    "NVA-PROJECT-003",
		Message: fmt.Sprintf("manifest value must be quoted string on line %d", lineNumber),
	})
	return strings.Trim(value, `"`)
}

func parseBool(value string) (bool, bool) {
	switch value {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
}

func parseStringArray(value string, diagnostics *[]Diagnostic, lineNumber int) ([]string, bool) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
		return nil, false
	}

	body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
	if body == "" {
		return nil, true
	}

	parts := strings.Split(body, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		item := parseString(strings.TrimSpace(part), diagnostics, lineNumber)
		if item != "" {
			out = append(out, item)
		}
	}
	return out, true
}

func stripComment(line string) string {
	if idx := strings.Index(line, "#"); idx >= 0 {
		return line[:idx]
	}
	return line
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

func validateSourcePath(filePath string) []Diagnostic {
	if !strings.HasPrefix(filePath, "src/") || !strings.HasSuffix(filePath, ".nova") {
		return nil
	}
	base := path.Base(filePath)
	if isPascalCaseNova(base) {
		return nil
	}
	return []Diagnostic{{
		Code:    "NVA-LAYOUT-004",
		Message: fmt.Sprintf("capability source %s should use PascalCase.nova", filePath),
		Path:    filePath,
	}}
}

func validateGeneratedPath(filePath string) []Diagnostic {
	if !strings.HasPrefix(filePath, "src/build/") {
		return nil
	}
	return []Diagnostic{{
		Code:    "NVA-LAYOUT-005",
		Message: "generated output should stay under build/",
		Path:    filePath,
	}}
}

func validateADRPath(filePath string) []Diagnostic {
	if !strings.HasPrefix(filePath, "docs/adr/") || path.Ext(filePath) != ".md" {
		return nil
	}
	if adrNamePattern.MatchString(path.Base(filePath)) {
		return nil
	}
	return []Diagnostic{{
		Code:    "NVA-LAYOUT-006",
		Message: "ADR docs should use adr_NNN_slug.md",
		Path:    filePath,
	}}
}

func isPascalCaseNova(name string) bool {
	if !strings.HasSuffix(name, ".nova") {
		return false
	}
	stem := strings.TrimSuffix(name, ".nova")
	if stem == "" || stem[0] < 'A' || stem[0] > 'Z' {
		return false
	}
	for _, ch := range stem {
		if ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' {
			continue
		}
		return false
	}
	return true
}

func normalizePath(filePath string) string {
	filePath = strings.ReplaceAll(filePath, "\\", "/")
	return path.Clean(filePath)
}
