package style

import (
	"sort"
	"strings"
)

// MergedClassRules combines all sheets into one class → properties map (later sheets override).
func MergedClassRules(bundle Bundle) map[string]ClassRule {
	out := make(map[string]ClassRule)
	for _, sheet := range bundle.Sheets {
		for className, rule := range sheet.Classes {
			current := out[className]
			if current.Properties == nil {
				current.Properties = make(map[string]string)
			}
			for name, value := range rule.Properties {
				current.Properties[name] = value
			}
			out[className] = current
		}
	}
	return out
}

// MergedStateRules combines states per class and pseudo (later sheets override).
func MergedStateRules(bundle Bundle) map[string]map[string]StateRule {
	out := make(map[string]map[string]StateRule)
	for _, sheet := range bundle.Sheets {
		for _, state := range sheet.States {
			classStates := out[state.Class]
			if classStates == nil {
				classStates = make(map[string]StateRule)
				out[state.Class] = classStates
			}
			current := classStates[state.Pseudo]
			if current.Properties == nil {
				current.Properties = make(map[string]string)
			}
			current.Class = state.Class
			current.Pseudo = state.Pseudo
			for name, value := range state.Properties {
				current.Properties[name] = value
			}
			classStates[state.Pseudo] = current
		}
	}
	return out
}

// ResolvedStyleForClasses merges portable classes from a static class list.
func ResolvedStyleForClasses(bundle Bundle, classList string) ResolvedStyle {
	props := make(map[string]string)
	for _, className := range strings.Fields(classList) {
		rule, ok := MergedClassRules(bundle)[className]
		if !ok {
			continue
		}
		for name, value := range rule.Properties {
			props[name] = value
		}
	}
	return NewResolvedStyle(props)
}

// ResolvedStatesForClasses merges interaction states for a static class list.
func ResolvedStatesForClasses(bundle Bundle, classList string) map[string]ResolvedStyle {
	out := make(map[string]ResolvedStyle)
	merged := MergedStateRules(bundle)
	for _, className := range strings.Fields(classList) {
		classStates, ok := merged[className]
		if !ok {
			continue
		}
		for pseudo, rule := range classStates {
			current := out[pseudo]
			if current.Properties == nil {
				current.Properties = make(map[string]string)
			}
			for name, value := range rule.Properties {
				current.Properties[name] = value
			}
			out[pseudo] = current
		}
	}
	return out
}

// NormalizeBundle sorts states and applies target-specific state filtering for providers.
func NormalizeBundle(bundle Bundle, targetID string) Bundle {
	out := bundle
	out.Sheets = make([]Sheet, 0, len(bundle.Sheets))
	for _, sheet := range bundle.Sheets {
		copySheet := sheet
		copySheet.States = sortStateRules(sheet.States)
		if targetID == "android" {
			filtered, _ := FilterAndroidStates(copySheet.States)
			copySheet.States = filtered
		}
		out.Sheets = append(out.Sheets, copySheet)
	}
	return out
}

func sortStateRules(states []StateRule) []StateRule {
	if len(states) <= 1 {
		return states
	}
	out := append([]StateRule(nil), states...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Class != out[j].Class {
			return out[i].Class < out[j].Class
		}
		return out[i].Pseudo < out[j].Pseudo
	})
	return out
}

// MergeResolvedStyles overlays override properties onto base.
func MergeResolvedStyles(base, override ResolvedStyle) ResolvedStyle {
	out := ResolvedStyle{Properties: make(map[string]string)}
	for name, value := range base.Properties {
		out.Properties[name] = value
	}
	for name, value := range override.Properties {
		out.Properties[name] = value
	}
	return out
}
