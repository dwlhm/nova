package expr

import (
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/parser"
)

func parseTestSource(t *testing.T, source string) parser.File {
	t.Helper()
	file, diagnostics := parser.Parse(lexer.Tokenize(source))
	if len(diagnostics) > 0 {
		t.Fatalf("parse diagnostics: %+v", diagnostics)
	}
	return file
}

func TestEvaluateFuncCallAndTernary(t *testing.T) {
	file := parseTestSource(t, `
<func filterLabel kind: string filter: string returns string>
  filter == kind ? "● " + kind : kind
/|

<contract state S>
  historyFilter: string <- "all" {
    @ready -> filterLabel "Semua" historyFilter;
  };
/|
`)

	reg := BuildRegistry([]parser.File{file})
	stateNames := map[string]bool{"historyFilter": true}
	ctx := Context{State: map[string]any{"historyFilter": "Semua"}}

	value, err := Evaluate(
		lexer.Tokenize(`filterLabel "Semua" historyFilter`),
		reg,
		stateNames,
		nil,
		ctx,
	)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if value != "● Semua" {
		t.Fatalf("value = %q, want ● Semua", value)
	}
}

func TestEvaluateChartMax(t *testing.T) {
	file := parseTestSource(t, `
<func chartMax income: number expense: number transfer: number returns number>
  income > expense ? income > transfer ? income : transfer : expense > transfer ? expense : transfer
/|
`)
	reg := BuildRegistry([]parser.File{file})
	value, err := Evaluate(
		lexer.Tokenize(`chartMax 10 30 20`),
		reg,
		nil,
		nil,
		Context{},
	)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if value != float64(30) {
		t.Fatalf("value = %v, want 30", value)
	}
}

func TestEmitJavaNovaExprRecordBodyUsesMethodParams(t *testing.T) {
	file := parseTestSource(t, `
<func applyExpense ledger: record category: string amount: number account: string note: string returns record>
  {
    version <- ledger.version;
    transactionCount <- ledger.transactionCount + 1;
    netTotal <- ledger.netTotal - amount;
    line1 <- expenseSummary category amount account note;
  }
/|

<func expenseSummary category: string amount: number account: string note: string returns string>
  category + " " + amount
/|
`)
	reg := BuildRegistry([]parser.File{file})
	emitted, err := EmitJavaNovaExpr("nova.generated.finance", reg)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	if !strings.Contains(emitted, "public static Object invoke(String name, Object[] args)") {
		t.Fatalf("emitted NovaExpr must include static invoke dispatch:\n%s", emitted)
	}
	for _, part := range []string{
		"NovaExpr.field(ledger, \"version\")",
		"NovaRuntime.record(",
		"NovaRuntime.entry(",
		"NovaExpr.expenseSummary(category, amount, account, note)",
	} {
		if !strings.Contains(emitted, part) {
			t.Fatalf("emitted = %q, missing %q", emitted, part)
		}
	}
	for _, forbidden := range []string{"payload.", "record(entry", "(ledger)."} {
		if strings.Contains(emitted, forbidden) {
			t.Fatalf("emitted = %q, must not contain %q", emitted, forbidden)
		}
	}
}

func TestLowerJavaFuncBodyUsesMethodParamsAndFieldHelper(t *testing.T) {
	file := parseTestSource(t, `
<func bump ledger: record returns record>
  {
    version <- ledger.version + 1;
    note <- ledger.note;
  }
/|
`)
	reg := BuildRegistry([]parser.File{file})
	fn, _ := reg.Lookup("bump")
	lowered, err := LowerJava(fn.Body, reg, nil, stringSet(fn.Params))
	if err != nil {
		t.Fatalf("lower: %v", err)
	}
	for _, part := range []string{
		"NovaExpr.field(ledger, \"version\")",
		"NovaRuntime.add(",
		"NovaRuntime.record(",
		"NovaRuntime.entry(",
	} {
		if !strings.Contains(lowered, part) {
			t.Fatalf("lowered = %q, missing %q", lowered, part)
		}
	}
	if strings.Contains(lowered, "payload.") {
		t.Fatalf("lowered = %q, should not reference payload", lowered)
	}
}

func TestEmitJavaLedgerFromStoredDoesNotFallbackToNull(t *testing.T) {
	file := parseTestSource(t, `
<func storedVersion value: unknown returns number>
  value == null ? 1 : value.version
/|

<func storedTransactionCount value: unknown returns number>
  value == null ? 0 : value.transactionCount
/|

<func emptyLedger returns record>
  {
    version <- 1;
    transactionCount <- 0;
  }
/|

<func ledgerFromStored value: unknown returns record>
  value == null ? emptyLedger() : {
    version <- storedVersion value;
    transactionCount <- storedTransactionCount value;
  }
/|
`)
	reg := BuildRegistry([]parser.File{file})
	fn, ok := reg.Lookup("ledgerFromStored")
	if !ok {
		t.Fatal("missing ledgerFromStored")
	}
	lowered, err := LowerJava(fn.Body, reg, nil, stringSet(fn.Params))
	if err != nil {
		t.Fatalf("lower: %v", err)
	}
	if lowered == "null" {
		t.Fatalf("lowered = null, want ledgerFromStored body")
	}
	emitted, err := EmitJavaNovaExpr("nova.generated.finance", reg)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	if !strings.Contains(emitted, "public static Object invoke(String name, Object[] args)") {
		t.Fatalf("emitted NovaExpr must include static invoke dispatch:\n%s", emitted)
	}
	if strings.Contains(emitted, "public static Object ledgerFromStored(Object value) {\n        return null;\n    }") {
		t.Fatalf("emitted ledgerFromStored must not fallback to null:\n%s", emitted)
	}
}

func TestLowerJSFuncCall(t *testing.T) {
	file := parseTestSource(t, `
<func piePercent value: number total: number returns string>
  total == 0 ? "0%" : value * 100 / total + "%"
/|
`)
	reg := BuildRegistry([]parser.File{file})
	lowered, err := LowerJS(
		lexer.Tokenize(`piePercent incomeTotal incomeTotal + expenseTotal`),
		reg,
		map[string]bool{"incomeTotal": true, "expenseTotal": true},
		nil,
	)
	if err != nil {
		t.Fatalf("lower: %v", err)
	}
	for _, part := range []string{"NovaExpr.piePercent", "state.incomeTotal", "state.expenseTotal"} {
		if !strings.Contains(lowered, part) {
			t.Fatalf("lowered = %q, missing %q", lowered, part)
		}
	}
}
