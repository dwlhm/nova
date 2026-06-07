package androidcodegen

import (
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/core/style"
)

func TestAndroidStyleSheetUsesCoreResolvedPadding(t *testing.T) {
	bundle := style.Bundle{Sheets: []style.Sheet{{
		Classes: map[string]style.ClassRule{
			"card": {Properties: map[string]string{"padding": "8 12"}},
		},
	}}}
	sheet := newAndroidStyleSheet(bundle)
	resolved := sheet.StyleForClassList("card")
	padding, ok := resolved.Padding()
	if !ok || padding.Left != 12 || padding.Top != 8 {
		t.Fatalf("padding = %+v ok=%v", padding, ok)
	}
}

func TestAndroidJavaStyleApplicationCallsNovaStyle(t *testing.T) {
	resolved := style.NewResolvedStyle(map[string]string{"padding": "10"})
	code := androidJavaStyleApplication("view", "surface", resolved, nil, "        ")
	if !strings.Contains(code, "NovaStyle.applyWithStates") {
		t.Fatalf("code = %q", code)
	}
	if !strings.Contains(code, "this::dp") {
		t.Fatalf("code = %q", code)
	}
}

func TestNovaStyleRulesFromBundleEmitsClassRules(t *testing.T) {
	bundle := style.Bundle{Sheets: []style.Sheet{{
		Classes: map[string]style.ClassRule{
			"card": {Properties: map[string]string{"margin": "8"}},
		},
	}}}
	rules := NovaStyleRulesFromBundle(bundle)
	if !strings.Contains(rules, "CLASS_RULES") {
		t.Fatalf("rules = %q", rules)
	}
	if !strings.Contains(rules, `"card"`) {
		t.Fatalf("rules = %q", rules)
	}
}
