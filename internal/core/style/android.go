package style

import "fmt"

// AndroidPseudoOrder is the deterministic state precedence for Android selectors.
var AndroidPseudoOrder = []string{"disabled", "active", "focus", "checked"}

// AndroidViewStateAttrs returns the generated Java View state attribute for a web-canonical pseudo.
func AndroidViewStateAttrs(pseudo string) (string, bool) {
	switch pseudo {
	case "disabled":
		return "-android.R.attr.state_enabled", true
	case "active":
		return "android.R.attr.state_pressed", true
	case "focus":
		return "android.R.attr.state_focused", true
	case "checked":
		return "android.R.attr.state_checked", true
	default:
		return "", false
	}
}

// FilterAndroidStates removes Android-unsupported pseudos and emits NVA-STYLE-011 warnings.
func FilterAndroidStates(states []StateRule) ([]StateRule, []Diagnostic) {
	filtered := make([]StateRule, 0, len(states))
	diagnostics := make([]Diagnostic, 0)
	for _, rule := range states {
		if rule.Pseudo == "hover" {
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "NVA-STYLE-011",
				Message: fmt.Sprintf("state %s hover is ignored on android", rule.Class),
			})
			continue
		}
		if _, ok := AndroidViewStateAttrs(rule.Pseudo); !ok {
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "NVA-STYLE-011",
				Message: fmt.Sprintf("state %s %s is ignored on android", rule.Class, rule.Pseudo),
			})
			continue
		}
		filtered = append(filtered, rule)
	}
	return filtered, diagnostics
}
