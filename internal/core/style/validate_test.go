package style

import (
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/view"
)

func TestValidateViewClassesWarnsOnDynamicClass(t *testing.T) {
	bundle := Bundle{Sheets: []Sheet{{
		Classes: map[string]ClassRule{"surface-card": {}},
	}}}
	viewIR := view.IR{Nodes: []view.Node{{
		Kind: "surface",
		Props: map[string]view.Binding{
			"class": {Tokens: []lexer.Token{{Type: lexer.IDENT, Literal: "themeClass"}}},
		},
	}}}
	diagnostics := ValidateViewClasses(viewIR, bundle)
	if len(diagnostics) != 1 || diagnostics[0].Code != "NVA-STYLE-007" {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

func TestValidateViewClassesWarnsOnUnknownClass(t *testing.T) {
	bundle := Bundle{Sheets: []Sheet{{
		Classes: map[string]ClassRule{"surface-card": {}},
	}}}
	viewIR := view.IR{Nodes: []view.Node{{
		Kind: "button",
		Props: map[string]view.Binding{
			"class": {Tokens: []lexer.Token{{Type: lexer.STRING, Literal: "missing-btn"}}},
		},
	}}}
	diagnostics := ValidateViewClasses(viewIR, bundle)
	if len(diagnostics) != 1 || diagnostics[0].Code != "NVA-STYLE-008" {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

func TestValidateViewClassesAcceptsMultiClassLiteral(t *testing.T) {
	bundle := Bundle{Sheets: []Sheet{{
		Classes: map[string]ClassRule{
			"type-button":    {},
			"expense-action": {},
		},
	}}}
	viewIR := view.IR{Nodes: []view.Node{{
		Kind: "button",
		Props: map[string]view.Binding{
			"class": {Tokens: []lexer.Token{{Type: lexer.STRING, Literal: "type-button expense-action"}}},
		},
	}}}
	if diagnostics := ValidateViewClasses(viewIR, bundle); len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

func TestStaticClassBindingParsesQuotedLiteral(t *testing.T) {
	classes, ok := staticClassBinding(view.Binding{
		Tokens: []lexer.Token{{Type: lexer.STRING, Literal: "primary-btn"}},
	})
	if !ok || strings.Join(classes, " ") != "primary-btn" {
		t.Fatalf("classes = %v ok = %v", classes, ok)
	}
}
