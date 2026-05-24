package project

import "fmt"

func parseRendererUnknownKind(value string, diagnostics *[]Diagnostic, lineNumber int) RendererUnknownKindPolicy {
	policy := RendererUnknownKindPolicy(parseString(value, diagnostics, lineNumber))
	switch policy {
	case RendererUnknownKindError, RendererUnknownKindWarn, RendererUnknownKindPassthroughWeb:
		return policy
	default:
		*diagnostics = append(*diagnostics, Diagnostic{
			Code:    "NVA-PROJECT-005",
			Message: fmt.Sprintf("renderer.unknown_kind must be error, warn, or passthrough_web on line %d", lineNumber),
		})
		return RendererUnknownKindError
	}
}

func parseRendererPackages(value string, diagnostics *[]Diagnostic, lineNumber int) []RendererPackageRef {
	items, ok := parseStringArray(value, diagnostics, lineNumber)
	if !ok {
		*diagnostics = append(*diagnostics, Diagnostic{
			Code:    "NVA-PROJECT-006",
			Message: fmt.Sprintf("renderer.extensions.packages must be a string array on line %d", lineNumber),
		})
		return nil
	}
	packages := make([]RendererPackageRef, 0, len(items))
	for _, item := range items {
		if item != "" {
			packages = append(packages, RendererPackageRef{Name: item, Constraint: "*"})
		}
	}
	return packages
}

func assignRendererDictionaryValue(entry *RendererPrimitive, key string, value string, diagnostics *[]Diagnostic, lineNumber int) {
	if entry.Targets == nil {
		entry.Targets = make(map[string]RendererTarget)
	}
	switch key {
	case "kind":
		entry.Kind = parseString(value, diagnostics, lineNumber)
	case "description":
		entry.Description = parseString(value, diagnostics, lineNumber)
	case "props":
		entry.Props = rendererFields(parseStringArrayOrDiagnostic(value, diagnostics, lineNumber), "unknown")
	case "events":
		entry.Events = rendererEvents(parseStringArrayOrDiagnostic(value, diagnostics, lineNumber))
	case "allow_override":
		enabled, ok := parseBool(value)
		if !ok {
			*diagnostics = append(*diagnostics, Diagnostic{
				Code:    "NVA-PROJECT-007",
				Message: fmt.Sprintf("renderer.dictionary allow_override must be true or false on line %d", lineNumber),
			})
			return
		}
		entry.AllowOverride = enabled
	}
}

func assignRendererDictionaryTarget(entry *RendererPrimitive, target string, key string, value string, diagnostics *[]Diagnostic, lineNumber int) {
	if entry.Targets == nil {
		entry.Targets = make(map[string]RendererTarget)
	}
	targetEntry := entry.Targets[target]
	switch key {
	case "strategy":
		targetEntry.Strategy = parseString(value, diagnostics, lineNumber)
	case "adapter":
		targetEntry.Adapter = normalizePath(parseString(value, diagnostics, lineNumber))
	case "tag":
		targetEntry.Tag = parseString(value, diagnostics, lineNumber)
	case "delegate", "delegate_kind":
		targetEntry.Delegate = parseString(value, diagnostics, lineNumber)
	}
	entry.Targets[target] = targetEntry
}

func parseStringArrayOrDiagnostic(value string, diagnostics *[]Diagnostic, lineNumber int) []string {
	items, ok := parseStringArray(value, diagnostics, lineNumber)
	if ok {
		return items
	}
	*diagnostics = append(*diagnostics, Diagnostic{
		Code:    "NVA-PROJECT-006",
		Message: fmt.Sprintf("renderer dictionary value must be a string array on line %d", lineNumber),
	})
	return nil
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
