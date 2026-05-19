package validator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/lexer"
	"github.com/dwlhm/nova/internal/parser"
)

func TestValidateAllowsExternalOperationsInLifecycle(t *testing.T) {
	input := `<import external storage from "./storage.web.js">
  operation set {
    input {
      key: string;
      value: unknown;
    }

    output void;
  }
/|
<contract state Counter>
  count: number <- 0 {
    @increment -> count + 1;
  };
/|
<contract capability CounterEvents>
  emits {
    @stored: void;
  }
/|
<lifecycle after @increment>
  count |> storage.set key <- "counter";
  void -> @stored;
/|
`

	diagnostics := parseAndValidate(t, input)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
}

func TestValidateRejectsExternalOperationsOutsideLifecycle(t *testing.T) {
	input := `<import external storage from "./storage.web.js">
  operation set {
    input {
      key: string;
      value: unknown;
    }

    output void;
  }
/|
<contract state Counter>
  count: number <- 0 {
    @increment -> count |> storage.set key <- "counter";
  };
/|
<func save value: number returns void>
  value |> storage.set key <- "counter"
/|
<template>
  <button on_press -> @increment>
    count |> storage.set key <- "counter"
  /|
/|
`

	diagnostics := parseAndValidate(t, input)
	assertDiagnostic(t, diagnostics, "state transition cannot call external operation")
	assertDiagnostic(t, diagnostics, "func cannot call external operation")
	assertDiagnostic(t, diagnostics, "template cannot call external operation")
}

func TestValidateRejectsImpureFunctionBehavior(t *testing.T) {
	input := `<contract state Counter>
  count: number <- 0 {
    @increment -> count + 1;
  };
/|
<func impure value: number returns number>
  count |> add value;
  value -> @increment;
/|
`

	diagnostics := parseAndValidate(t, input)
	assertDiagnostic(t, diagnostics, "func cannot read state")
	assertDiagnostic(t, diagnostics, "func cannot emit scheduler event")
}

func TestValidateRejectsExternalOperationInStateInitialValue(t *testing.T) {
	input := `<import external storage from "./storage.web.js">
  operation load {
    input {
      key: string;
    }

    output unknown;
  }
/|
<contract state Counter>
  count: unknown <- storage.load key <- "counter";
/|
`

	diagnostics := parseAndValidate(t, input)
	assertDiagnostic(t, diagnostics, "state contract cannot call external operation")
}

func TestValidateAllowsTemplateStateReadAndEventEmit(t *testing.T) {
	input := `<contract state Counter>
  count: number <- 0 {
    @increment -> count + 1;
  };
/|
<template>
  <button on_press -> @increment>
    count
  /|
/|
`

	diagnostics := parseAndValidate(t, input)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
}

func TestValidateUsesImportAliasesAsLocalNames(t *testing.T) {
	input := `<import state count as cartCount from "./Cart.nova" /|
<import event @submit as @cart_submit from "./Cart.nova" /|
<template>
  <button on_press -> @cart_submit>
    cartCount
  /|
/|
<lifecycle after @cart_submit>
  void;
/|
<func total returns number>
  cartCount
/|
`

	diagnostics := parseAndValidate(t, input)
	assertDiagnostic(t, diagnostics, "func cannot read state cartCount")
	if hasValidatorDiagnostic(diagnostics, "undeclared scheduler event @cart_submit") {
		t.Fatalf("alias event should be declared: %v", diagnostics)
	}
}

func TestValidateRejectsLocalSymbolCollisions(t *testing.T) {
	input := `<import state count as total from "./Cart.nova" /|
<func total returns number>
  1
/|
<import external storage from "@env/storage">
  operation set {
    output void;
  }

  operation set {
    output void;
  }
/|
`

	diagnostics := parseAndValidate(t, input)
	assertDiagnostic(t, diagnostics, "local symbol total")
	assertDiagnostic(t, diagnostics, "local symbol storage.set")
}

func TestValidateAudioLabExample(t *testing.T) {
	path := filepath.Join("..", "..", "docs", "example", "audiolab.nova")
	input, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read example: %v", err)
	}

	diagnostics := parseAndValidate(t, string(input))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
}

func TestValidateMultiPageExample(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "multipage", "src", "App.nova")
	input, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read example: %v", err)
	}

	diagnostics := parseAndValidate(t, string(input))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
}

func TestValidateRejectsUndeclaredEvents(t *testing.T) {
	input := `<contract state Counter>
  count: number <- 0 {
    @increment -> count + 1;
  };
/|
<template>
  <button on_press -> @missing>
    count
  /|
/|
<lifecycle after @missing>
  void -> @also_missing;
/|
`

	diagnostics := parseAndValidate(t, input)
	assertDiagnostic(t, diagnostics, "template emits undeclared scheduler event @missing")
	assertDiagnostic(t, diagnostics, "lifecycle listens to undeclared scheduler event @missing")
	assertDiagnostic(t, diagnostics, "lifecycle emits undeclared scheduler event @also_missing")
}

func TestValidateRejectsEventPayloadArityMismatch(t *testing.T) {
	input := `<contract state Counter>
  count: number <- 0 {
    @set(value: number) -> value;
    @saved -> count;
  };
/|
<template>
  <button on_press -> @set>
    count
  /|
/|
<lifecycle after @saved>
  void -> @set;
/|
`

	diagnostics := parseAndValidate(t, input)
	assertDiagnostic(t, diagnostics, "template emits scheduler event @set with void, want 1 payload value")
	assertDiagnostic(t, diagnostics, "lifecycle emits scheduler event @set with void, want 1 payload value")
}

func TestValidateRejectsDirectStateWritesOutsideTransitions(t *testing.T) {
	input := `<contract state Counter>
  count: number <- 0 {
    @increment -> count <- 1;
  };
/|
<func reset value: number returns number>
  count <- value
/|
<lifecycle after @increment>
  count <- 10;
/|
`

	diagnostics := parseAndValidate(t, input)
	assertDiagnostic(t, diagnostics, "state transition cannot write state count")
	assertDiagnostic(t, diagnostics, "func cannot write state count")
	assertDiagnostic(t, diagnostics, "lifecycle cannot write state count")
}

func TestValidateRejectsSimpleTypeMismatches(t *testing.T) {
	input := `<contract state Counter>
  count: number <- "1" {
    @bad -> "not a number";
    @set(value: number) -> value;
  };
/|
<func label count: number returns string>
  count
/|
<template>
  <text value <- count /|
  <button on_press -> @set("bad") /|
/|`

	diagnostics := parseAndValidate(t, input)
	assertDiagnostic(t, diagnostics, "state count initial value has type")
	assertDiagnostic(t, diagnostics, "state transition for count has type")
	assertDiagnostic(t, diagnostics, "func label return has type")
	assertDiagnostic(t, diagnostics, "text value binding has type")
	assertDiagnostic(t, diagnostics, "template emits scheduler event @set payload 1")
}

func parseAndValidate(t *testing.T, input string) []Diagnostic {
	t.Helper()

	file, parserDiagnostics := parser.Parse(lexer.Tokenize(input))
	if len(parserDiagnostics) != 0 {
		t.Fatalf("unexpected parser diagnostics: %v", parserDiagnostics)
	}
	return Validate(file)
}

func assertDiagnostic(t *testing.T, diagnostics []Diagnostic, want string) {
	t.Helper()

	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, want) {
			return
		}
	}
	t.Fatalf("missing diagnostic %q in %v", want, diagnostics)
}

func hasValidatorDiagnostic(diagnostics []Diagnostic, want string) bool {
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, want) {
			return true
		}
	}
	return false
}
