package style

import "testing"

func TestParseDocumentResolvesTokensAndScope(t *testing.T) {
	content := `scope: app

token color.bg = #fbfcfe

class surface-card {
  padding: 12
  background: color.bg
}
`
	sheet, diagnostics := ParseDocument("src/App.nova-style", content)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}
	if sheet.Scope != ScopeApp {
		t.Fatalf("scope = %q", sheet.Scope)
	}
	if sheet.Classes["surface-card"].Properties["background"] != "#fbfcfe" {
		t.Fatalf("background = %q", sheet.Classes["surface-card"].Properties["background"])
	}
}

func TestParseDocumentRejectsDuplicateScope(t *testing.T) {
	_, diagnostics := ParseDocument("src/App.nova-style", "scope: app\nscope: global\n")
	if len(diagnostics) == 0 {
		t.Fatal("expected diagnostics")
	}
}
