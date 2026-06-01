package ir

import (
	"testing"

	"github.com/dwlhm/nova/internal/core/lexer"
)

func TestExpressionToJSCompilesRecordLiterals(t *testing.T) {
	tokens := lexer.Tokenize(`{ path <- "/settings"; title <- currentTitle; }`)
	expression := expressionToJS(tokens[:len(tokens)-1], map[string]bool{"currentTitle": true}, nil)

	if expression != `({ path: "/settings", title: state.currentTitle })` {
		t.Fatalf("expression = %s", expression)
	}
}
