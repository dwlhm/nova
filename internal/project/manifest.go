package project

import (
	"fmt"
	"strings"
)

func ParseManifest(input string) (Manifest, []Diagnostic) {
	manifest := Manifest{
		Targets:          make(map[string]Target),
		Renderer:         RendererConfig{UnknownKind: RendererUnknownKindError},
		Permissions:      make(PermissionMap),
		PermissionScopes: make(map[string][]string),
	}
	diagnostics := make([]Diagnostic, 0)
	section := ""
	dictionaryIndex := -1

	for lineNumber, raw := range strings.Split(input, "\n") {
		line := strings.TrimSpace(stripComment(raw))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[[") && strings.HasSuffix(line, "]]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "[["), "]]"))
			if section == "renderer.dictionary" {
				manifest.Renderer.Dictionary = append(manifest.Renderer.Dictionary, RendererPrimitive{Targets: make(map[string]RendererTarget)})
				dictionaryIndex = len(manifest.Renderer.Dictionary) - 1
			}
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
		manifest = assignManifestValue(manifest, section, dictionaryIndex, key, value, &diagnostics, lineNumber+1)
	}

	return manifest, diagnostics
}
func assignManifestValue(manifest Manifest, section string, dictionaryIndex int, key string, value string, diagnostics *[]Diagnostic, lineNumber int) Manifest {
	switch section {
	case "project":
		assignProjectValue(&manifest.Project, key, value, diagnostics, lineNumber)
	case "renderer":
		if key == "unknown_kind" {
			manifest.Renderer.UnknownKind = parseRendererUnknownKind(value, diagnostics, lineNumber)
		}
	case "renderer.extensions":
		if key == "packages" {
			manifest.Renderer.ExtensionPackages = parseRendererPackages(value, diagnostics, lineNumber)
		}
	case "renderer.dictionary":
		if dictionaryIndex >= 0 && dictionaryIndex < len(manifest.Renderer.Dictionary) {
			assignRendererDictionaryValue(&manifest.Renderer.Dictionary[dictionaryIndex], key, value, diagnostics, lineNumber)
		}
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
	case "dependencies":
		manifest.Dependencies = append(manifest.Dependencies, Dependency{
			Name:       parseString(key, diagnostics, lineNumber),
			Constraint: parseString(value, diagnostics, lineNumber),
		})
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
		if target, ok := strings.CutPrefix(section, "renderer.dictionary."); ok {
			if dictionaryIndex >= 0 && dictionaryIndex < len(manifest.Renderer.Dictionary) {
				assignRendererDictionaryTarget(&manifest.Renderer.Dictionary[dictionaryIndex], target, key, value, diagnostics, lineNumber)
			}
		}
	}
	return manifest
}
