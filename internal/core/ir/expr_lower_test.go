package ir

import (
	"os"
	"testing"

	"github.com/dwlhm/nova/internal/core/expr"
	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/parser"
)

func TestExpressionToJSCompilesRecordLiterals(t *testing.T) {
	tokens := lexer.Tokenize(`{ path <- "/settings"; title <- currentTitle; }`)
	expression, diagnostics := lowerExpressionJS(tokens[:len(tokens)-1], expr.BuildRegistry(nil), map[string]bool{"currentTitle": true}, nil)
	if len(diagnostics) > 0 {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}

	if expression != `({ path: "/settings", title: state.currentTitle })` {
		t.Fatalf("expression = %s", expression)
	}
}

func TestFinanceRouteChangedTransitionLowers(t *testing.T) {
	pure := parseNovaFile(t, "../../../tests/conformance/web/finance/src/FinancePure.nova")
	store := parseNovaFile(t, "../../../tests/conformance/web/finance/src/FinanceStore.nova")
	sources := map[string]parser.File{
		"src/FinancePure.nova":  pure,
		"src/FinanceStore.nova": store,
	}
	registry := buildExprRegistry(sources)
	stateNames := collectModelStateNames([]ModuleRef{{Path: "src/FinanceStore.nova"}}, sources)
	paramSet := stringSet([]string{"next"})
	for _, contract := range store.ContractStates {
		for _, state := range contract.States {
			if state.Name != "inputTabClass" {
				continue
			}
			for _, transition := range state.Transitions {
				if transition.Event.Name != "@route_changed" {
					continue
				}
				lowered, diags := lowerExpressionJS(transition.Expr, registry, stateNames, paramSet)
				if len(diags) > 0 {
					t.Fatalf("diagnostics = %+v", diags)
				}
				t.Log(lowered)
			}
		}
	}
}

func parseNovaFile(t *testing.T, path string) parser.File {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	file, diagnostics := parser.Parse(lexer.Tokenize(string(content)))
	if len(diagnostics) > 0 {
		t.Fatalf("parse diagnostics: %+v", diagnostics)
	}
	return file
}
