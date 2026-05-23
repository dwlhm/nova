package project

import (
	"fmt"
	"strings"
)

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
			if target.Options == nil {
				target.Options = make(map[string]string)
			}
			switch key {
			case "renderer":
				target.Renderer = parseString(value, diagnostics, lineNumber)
			case "styles":
				target.Styles = parseTargetStyles(value, diagnostics, lineNumber)
			case "scoped_styles":
				target.ScopedStyles = parseTargetStyles(value, diagnostics, lineNumber)
			default:
				target.Options[key] = parseScalar(value, diagnostics, lineNumber)
			}
			manifest.Targets[targetID] = target
		}
	}
	return manifest
}
