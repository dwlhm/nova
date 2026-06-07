package style

import "testing"

func TestParseBoxShorthandMatchesCSSOrder(t *testing.T) {
	box, ok := ParseBoxShorthand("4 0 0 0")
	if !ok {
		t.Fatal("expected ok")
	}
	if box.Top != 4 || box.Right != 0 || box.Bottom != 0 || box.Left != 0 {
		t.Fatalf("box = %+v", box)
	}
	two, ok := ParseBoxShorthand("10 12")
	if !ok || two.Top != 10 || two.Right != 12 || two.Bottom != 10 || two.Left != 12 {
		t.Fatalf("two-value box = %+v ok=%v", two, ok)
	}
}

func TestResolvePropertyValueResolvesBorderTokenFields(t *testing.T) {
	tokens := map[string]string{"color.border": "#d9e1ea"}
	resolved := resolvePropertyValue("1 color.border", tokens)
	if resolved != "1 #d9e1ea" {
		t.Fatalf("resolved = %q", resolved)
	}
}

func TestValidatePropertyValuesRejectsInvalidPadding(t *testing.T) {
	diagnostics := ValidatePropertyValues("class card", map[string]string{
		"padding": "10 12 13 14 15",
	})
	if len(diagnostics) != 1 || diagnostics[0].Code != "NVA-STYLE-010" {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

func TestValidatePropertyValuesAcceptsMarginShorthand(t *testing.T) {
	diagnostics := ValidatePropertyValues("class card", map[string]string{"margin": "8 12"})
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}
