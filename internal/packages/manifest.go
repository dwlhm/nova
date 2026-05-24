package packages

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dwlhm/nova/internal/security"
)

func ParseManifest(input string) (Manifest, []Diagnostic) {
	manifest := Manifest{
		Exports:     make(map[string]string),
		Renderer:    RendererManifest{Primitives: make(map[string]RendererPrimitive)},
		Targets:     make(map[string]TargetAdapter),
		Permissions: make(security.PermissionMap),
	}
	diagnostics := make([]Diagnostic, 0)
	section := ""

	for lineNumber, raw := range strings.Split(input, "\n") {
		line := strings.TrimSpace(stripManifestComment(raw))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			diagnostics = append(diagnostics, pkgDiagnostic("NVA-PKG-013", fmt.Sprintf("invalid package manifest entry on line %d", lineNumber+1)))
			continue
		}
		assignPackageManifestValue(&manifest, section, parseManifestKey(key), strings.TrimSpace(value), &diagnostics, lineNumber+1)
	}

	for kind, primitive := range manifest.Renderer.Primitives {
		if primitive.Kind == "" {
			primitive.Kind = kind
		}
		if primitive.Package == "" {
			primitive.Package = manifest.Name
		}
		manifest.Renderer.Primitives[kind] = primitive
	}
	return manifest, diagnostics
}

func assignPackageManifestValue(manifest *Manifest, section string, key string, value string, diagnostics *[]Diagnostic, lineNumber int) {
	switch section {
	case "package":
		switch key {
		case "name":
			manifest.Name = parseManifestString(value, diagnostics, lineNumber)
		case "version":
			manifest.Version = parseManifestString(value, diagnostics, lineNumber)
		case "type":
			manifest.Types = parsePackageTypes(value, diagnostics, lineNumber)
		case "language":
			manifest.Language = parseManifestString(value, diagnostics, lineNumber)
		case "abi":
			manifest.ABI = parseManifestString(value, diagnostics, lineNumber)
		}
	case "exports":
		manifest.Exports[key] = normalizePackagePath(parseManifestString(value, diagnostics, lineNumber))
	case "dependencies":
		manifest.Dependencies = append(manifest.Dependencies, Dependency{
			Name:       key,
			Constraint: parseManifestString(value, diagnostics, lineNumber),
		})
	case "permissions":
		enabled, ok := parseManifestBool(value)
		if !ok {
			*diagnostics = append(*diagnostics, pkgDiagnostic("NVA-PKG-014", fmt.Sprintf("permission %s must be true or false on line %d", key, lineNumber)))
			return
		}
		manifest.Permissions[security.Permission(key)] = enabled
	default:
		if target, ok := strings.CutPrefix(section, "targets."); ok {
			adapter := manifest.Targets[target]
			if key == "adapter" {
				adapter.Adapter = normalizePackagePath(parseManifestString(value, diagnostics, lineNumber))
			}
			manifest.Targets[target] = adapter
			return
		}
		if primitive, target, ok := rendererPrimitiveTargetSection(section); ok {
			entry := rendererPrimitive(manifest, primitive)
			targetEntry := entry.Targets[target]
			switch key {
			case "strategy":
				targetEntry.Strategy = parseManifestString(value, diagnostics, lineNumber)
			case "adapter":
				targetEntry.Adapter = normalizePackagePath(parseManifestString(value, diagnostics, lineNumber))
			case "tag":
				targetEntry.Tag = parseManifestString(value, diagnostics, lineNumber)
			case "delegate", "delegate_kind":
				targetEntry.Delegate = parseManifestString(value, diagnostics, lineNumber)
			}
			entry.Targets[target] = targetEntry
			manifest.Renderer.Primitives[primitive] = entry
			return
		}
		if primitive, ok := strings.CutPrefix(section, "renderer.primitives."); ok {
			entry := rendererPrimitive(manifest, primitive)
			switch key {
			case "description":
				entry.Description = parseManifestString(value, diagnostics, lineNumber)
			case "props":
				entry.Props = rendererFields(parseManifestStringArray(value, diagnostics, lineNumber), "unknown")
			case "events":
				entry.Events = rendererEvents(parseManifestStringArray(value, diagnostics, lineNumber))
			case "allow_override":
				enabled, ok := parseManifestBool(value)
				if !ok {
					*diagnostics = append(*diagnostics, pkgDiagnostic("NVA-PKG-014", fmt.Sprintf("allow_override must be true or false on line %d", lineNumber)))
					return
				}
				entry.AllowOverride = enabled
			}
			manifest.Renderer.Primitives[primitive] = entry
		}
	}
}

func rendererPrimitive(manifest *Manifest, kind string) RendererPrimitive {
	entry := manifest.Renderer.Primitives[kind]
	if entry.Kind == "" {
		entry.Kind = kind
	}
	if entry.Package == "" {
		entry.Package = manifest.Name
	}
	if entry.Targets == nil {
		entry.Targets = make(map[string]RendererTarget)
	}
	return entry
}

func rendererPrimitiveTargetSection(section string) (string, string, bool) {
	rest, ok := strings.CutPrefix(section, "renderer.primitives.")
	if !ok {
		return "", "", false
	}
	parts := strings.Split(rest, ".")
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func parsePackageTypes(value string, diagnostics *[]Diagnostic, lineNumber int) []PackageType {
	values := parseManifestStringArray(value, diagnostics, lineNumber)
	out := make([]PackageType, 0, len(values))
	for _, item := range values {
		if item != "" {
			out = append(out, PackageType(item))
		}
	}
	return out
}

func rendererFields(names []string, typ string) []RendererField {
	fields := make([]RendererField, 0, len(names))
	for _, name := range names {
		if name != "" {
			fields = append(fields, RendererField{Name: name, Type: typ, Optional: true})
		}
	}
	return fields
}

func rendererEvents(names []string) []RendererEvent {
	events := make([]RendererEvent, 0, len(names))
	for _, name := range names {
		if name != "" {
			events = append(events, RendererEvent{Name: name, Payload: "void"})
		}
	}
	return events
}

func parseManifestKey(value string) string {
	value = strings.TrimSpace(value)
	if unquoted, err := strconv.Unquote(value); err == nil {
		return unquoted
	}
	return value
}

func parseManifestString(value string, diagnostics *[]Diagnostic, lineNumber int) string {
	unquoted, err := strconv.Unquote(strings.TrimSpace(value))
	if err == nil {
		return unquoted
	}
	*diagnostics = append(*diagnostics, pkgDiagnostic("NVA-PKG-014", fmt.Sprintf("package manifest value must be quoted string on line %d", lineNumber)))
	return strings.Trim(strings.TrimSpace(value), `"`)
}

func parseManifestStringArray(value string, diagnostics *[]Diagnostic, lineNumber int) []string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
		*diagnostics = append(*diagnostics, pkgDiagnostic("NVA-PKG-014", fmt.Sprintf("package manifest value must be a string array on line %d", lineNumber)))
		return nil
	}
	body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
	if body == "" {
		return nil
	}
	parts := strings.Split(body, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		item := parseManifestString(strings.TrimSpace(part), diagnostics, lineNumber)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func parseManifestBool(value string) (bool, bool) {
	switch strings.TrimSpace(value) {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
}

func stripManifestComment(line string) string {
	if idx := strings.Index(line, "#"); idx >= 0 {
		return line[:idx]
	}
	return line
}

func normalizePackagePath(value string) string {
	return strings.Trim(strings.ReplaceAll(value, "\\", "/"), "/")
}
