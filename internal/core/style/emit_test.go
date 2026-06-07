package style

import (
	"strings"
	"testing"
)

func TestEmitCSSOrdersStatesDeterministically(t *testing.T) {
	sheet := Sheet{
		Classes: map[string]ClassRule{"btn": {Properties: map[string]string{"color": "#000"}}},
		States: []StateRule{
			{Class: "btn", Pseudo: "focus", Properties: map[string]string{"color": "#111"}},
			{Class: "btn", Pseudo: "active", Properties: map[string]string{"color": "#222"}},
		},
	}
	css := EmitCSS(sheet)
	activeIndex := strings.Index(css, ".btn:active")
	focusIndex := strings.Index(css, ".btn:focus")
	if activeIndex < 0 || focusIndex < 0 || activeIndex > focusIndex {
		t.Fatalf("css order wrong:\n%s", css)
	}
}

func TestEmitBundleCSSIncludesMargin(t *testing.T) {
	bundle := NormalizeBundle(Bundle{Sheets: []Sheet{{
		SourcePath: "App.nova-style",
		Classes:    map[string]ClassRule{"card": {Properties: map[string]string{"margin": "4"}}},
	}}}, "web")
	emitted := EmitBundleCSS(bundle)
	if !strings.Contains(emitted["App.nova-style"], "margin: 4;") {
		t.Fatalf("css = %q", emitted["App.nova-style"])
	}
}
