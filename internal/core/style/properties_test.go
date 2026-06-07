package style

import "testing"

func TestParseDocumentValidatesFinanceStyleFile(t *testing.T) {
	content := `scope: app

token color.border = #d9e1ea

class finance-shell {
  padding: 10
  border: 1 color.border
  border-radius: 8
  background: #f6f8fb
}

class tab-button {
  padding: 10
}

state tab-button active {
  background: #eef4ff
  border-color: #8aa7df
}
`
	sheet, diagnostics := ParseDocument("Finance.nova-style", content)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}
	if sheet.Classes["finance-shell"].Properties["border"] != "1 #d9e1ea" {
		t.Fatalf("border = %q", sheet.Classes["finance-shell"].Properties["border"])
	}
}

func TestParseDocumentRejectsUnknownStateClass(t *testing.T) {
	content := `scope: app

state missing active {
  color: #000
}
`
	_, diagnostics := ParseDocument("Bad.nova-style", content)
	if len(diagnostics) == 0 {
		t.Fatal("expected diagnostics")
	}
}

func TestValidateBundleDetectsDuplicateClassInScope(t *testing.T) {
	bundle := Bundle{Sheets: []Sheet{
		{SourcePath: "a.nova-style", Scope: ScopeApp, Classes: map[string]ClassRule{"card": {}}},
		{SourcePath: "b.nova-style", Scope: ScopeApp, Classes: map[string]ClassRule{"card": {}}},
	}}
	diagnostics := ValidateBundle(bundle)
	if len(diagnostics) != 1 || diagnostics[0].Code != "NVA-STYLE-004" {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}
