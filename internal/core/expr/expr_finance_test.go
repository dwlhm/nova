package expr

import (
	"os"
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/parser"
)

func TestEmitFinancePieLabelHelpers(t *testing.T) {
	content, err := os.ReadFile("../../../tests/conformance/web/finance/src/FinancePure.nova")
	if err != nil {
		t.Fatal(err)
	}
	file, diagnostics := parser.Parse(lexer.Tokenize(string(content)))
	if len(diagnostics) > 0 {
		t.Fatalf("parse: %+v", diagnostics)
	}
	reg := BuildRegistry([]parser.File{file})
	emitted, err := EmitJavaNovaExpr("nova.generated.finance", reg)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	for _, name := range []string{"pieIncomeLabelForLedger", "piePercent", "pieWidthClass"} {
		if !strings.Contains(emitted, name) {
			t.Fatalf("missing %s in emit", name)
		}
	}
}
