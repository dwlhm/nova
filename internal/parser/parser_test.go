package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/lexer"
)

func TestParseADRTopLevelConstructs(t *testing.T) {
	input := `<import Counter from "./Counter.nova" /|
<import state count from "./Counter.nova" /|
<import event @increment from "./Counter.nova" /|
<import external storage from "./storage.web.js">
  operation set {
    input {
      key: string;
      value: unknown;
    }

    output void;
  }
/|
<contract type UserId /|
<contract type Status>
  "idle" | "loading" | "success" | "error";
/|
<contract type User>
  id: UserId;
  name: string;
  email?: string;
/|
<contract state Counter>
  count: number <- 0 {
    @increment -> count |> add 1;
    @set(value: number) -> value;
  };
/|
<contract capability Button>
  props {
    label: string;
    disabled?: boolean;
  }

  emits {
    @pressed: void;
  }
/|
<func add value: number amount: number returns number>
  value + amount
/|
<template target <- web>
  <button on_press -> @increment>
    +
  /|
/|
<lifecycle after @increment>
  count |> storage.set key <- "counter";
  void -> @stored;
/|
`

	file, diagnostics := Parse(lexer.Tokenize(input))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}

	if len(file.Imports) != 3 {
		t.Fatalf("imports = %d, want 3", len(file.Imports))
	}
	if file.Imports[0].Kind != ImportCapability || file.Imports[0].Items[0].Name != "Counter" {
		t.Fatalf("unexpected capability import: %+v", file.Imports[0])
	}
	if file.Imports[1].Kind != ImportState || file.Imports[1].Items[0].Name != "count" {
		t.Fatalf("unexpected state import: %+v", file.Imports[1])
	}
	if file.Imports[2].Kind != ImportEvent || file.Imports[2].Items[0].Name != "@increment" {
		t.Fatalf("unexpected event import: %+v", file.Imports[2])
	}

	if len(file.ExternalImports) != 1 {
		t.Fatalf("external imports = %d, want 1", len(file.ExternalImports))
	}
	external := file.ExternalImports[0]
	if external.Name != "storage" || external.From != "./storage.web.js" {
		t.Fatalf("unexpected external import: %+v", external)
	}
	if len(external.Operations) != 1 || external.Operations[0].Name != "set" {
		t.Fatalf("unexpected operations: %+v", external.Operations)
	}
	if len(external.Operations[0].Inputs) != 2 || external.Operations[0].Output.Text != "void" {
		t.Fatalf("unexpected operation signature: %+v", external.Operations[0])
	}

	if len(file.ContractTypes) != 3 {
		t.Fatalf("contract types = %d, want 3", len(file.ContractTypes))
	}
	if !file.ContractTypes[0].Opaque || file.ContractTypes[0].Name != "UserId" {
		t.Fatalf("unexpected opaque type: %+v", file.ContractTypes[0])
	}
	if file.ContractTypes[1].Alias.Text != "idle|loading|success|error" {
		t.Fatalf("unexpected alias type: %+v", file.ContractTypes[1].Alias)
	}
	if len(file.ContractTypes[2].Fields) != 3 || !file.ContractTypes[2].Fields[2].Optional {
		t.Fatalf("unexpected record type: %+v", file.ContractTypes[2])
	}

	if len(file.ContractStates) != 1 {
		t.Fatalf("contract states = %d, want 1", len(file.ContractStates))
	}
	stateContract := file.ContractStates[0]
	if stateContract.Name != "Counter" || len(stateContract.States) != 1 {
		t.Fatalf("unexpected state contract: %+v", stateContract)
	}
	count := stateContract.States[0]
	if count.Name != "count" || count.Type.Text != "number" || len(count.Transitions) != 2 {
		t.Fatalf("unexpected state decl: %+v", count)
	}
	if count.Transitions[1].Event.Name != "@set" || count.Transitions[1].Event.Params[0].Name != "value" {
		t.Fatalf("unexpected transition payload: %+v", count.Transitions[1])
	}

	if len(file.ContractCapabilities) != 1 {
		t.Fatalf("capability contracts = %d, want 1", len(file.ContractCapabilities))
	}
	capability := file.ContractCapabilities[0]
	if capability.Name != "Button" || len(capability.Props) != 2 || len(capability.Emits) != 1 {
		t.Fatalf("unexpected capability: %+v", capability)
	}

	if len(file.Funcs) != 1 || file.Funcs[0].Name != "add" || len(file.Funcs[0].Params) != 2 {
		t.Fatalf("unexpected funcs: %+v", file.Funcs)
	}
	if file.Funcs[0].Return.Text != "number" {
		t.Fatalf("unexpected return type: %+v", file.Funcs[0].Return)
	}

	if len(file.Templates) != 1 || file.Templates[0].Target != "web" || len(file.Templates[0].Tokens) == 0 {
		t.Fatalf("unexpected template: %+v", file.Templates)
	}

	if len(file.Lifecycles) != 1 {
		t.Fatalf("lifecycles = %d, want 1", len(file.Lifecycles))
	}
	lifecycle := file.Lifecycles[0]
	if lifecycle.Phase != "after" || lifecycle.Event != "@increment" || len(lifecycle.Statements) != 2 {
		t.Fatalf("unexpected lifecycle: %+v", lifecycle)
	}
}

func TestParseRejectsEventsWithoutSchedulerPrefix(t *testing.T) {
	input := `<contract state Counter>
  count: number <- 0 {
    increment -> count;
  };
/|
<lifecycle after increment>
  void -> @done;
/|
`

	_, diagnostics := Parse(lexer.Tokenize(input))
	if !hasDiagnostic(diagnostics, "expected scheduler event") {
		t.Fatalf("expected scheduler event diagnostic, got: %v", diagnostics)
	}
}

func TestParseEventImportAliasRequiresSchedulerPrefix(t *testing.T) {
	input := `<import event @submit as @cart_submit from "./Cart.nova" /|`

	file, diagnostics := Parse(lexer.Tokenize(input))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	if len(file.Imports) != 1 || len(file.Imports[0].Items) != 1 {
		t.Fatalf("unexpected imports: %+v", file.Imports)
	}
	item := file.Imports[0].Items[0]
	if item.Name != "@submit" || item.Alias != "@cart_submit" {
		t.Fatalf("unexpected import item: %+v", item)
	}

	_, diagnostics = Parse(lexer.Tokenize(`<import event @submit as cartSubmit from "./Cart.nova" /|`))
	if !hasDiagnostic(diagnostics, "expected event import alias") {
		t.Fatalf("expected event import alias diagnostic, got: %v", diagnostics)
	}
}

func TestParseStateRecordInitialValue(t *testing.T) {
	input := `<contract type Engine>
  isFlat: boolean;
  gainOffset: number;
/|
<contract state Audio>
  engine: Engine <- {
    isFlat <- true;
    gainOffset <- 0.0;
  };

  count: number <- 0 {
    @increment -> count + 1;
  };
/|
`

	file, diagnostics := Parse(lexer.Tokenize(input))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	if len(file.ContractStates) != 1 || len(file.ContractStates[0].States) != 2 {
		t.Fatalf("unexpected state contract: %+v", file.ContractStates)
	}
	engine := file.ContractStates[0].States[0]
	if engine.Name != "engine" || len(engine.Transitions) != 0 || len(engine.Initial) == 0 {
		t.Fatalf("unexpected record initial state: %+v", engine)
	}
	count := file.ContractStates[0].States[1]
	if count.Name != "count" || len(count.Transitions) != 1 {
		t.Fatalf("unexpected transition state: %+v", count)
	}
}

func TestParseAudioLabExample(t *testing.T) {
	path := filepath.Join("..", "..", "docs", "example", "audiolab.nova")
	input, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read example: %v", err)
	}

	file, diagnostics := Parse(lexer.Tokenize(string(input)))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}

	if len(file.ExternalImports) != 3 {
		t.Fatalf("external imports = %d, want 3", len(file.ExternalImports))
	}
	if len(file.ContractTypes) != 2 {
		t.Fatalf("contract types = %d, want 2", len(file.ContractTypes))
	}
	if len(file.ContractStates) != 1 || file.ContractStates[0].Name != "AudioLab" {
		t.Fatalf("unexpected state contracts: %+v", file.ContractStates)
	}
	if len(file.ContractStates[0].States) != 4 {
		t.Fatalf("state decls = %d, want 4", len(file.ContractStates[0].States))
	}
	if len(file.Funcs) != 2 {
		t.Fatalf("funcs = %d, want 2", len(file.Funcs))
	}
	if len(file.Templates) != 1 || file.Templates[0].Target != "web" {
		t.Fatalf("unexpected templates: %+v", file.Templates)
	}
	if len(file.Lifecycles) != 3 {
		t.Fatalf("lifecycles = %d, want 3", len(file.Lifecycles))
	}
}

func TestParseMultiPageExample(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "multipage", "src", "App.nova")
	input, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read example: %v", err)
	}

	file, diagnostics := Parse(lexer.Tokenize(string(input)))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}

	if len(file.ContractTypes) != 1 || file.ContractTypes[0].Name != "Route" {
		t.Fatalf("unexpected route type: %+v", file.ContractTypes)
	}
	if len(file.ContractStates) != 1 || file.ContractStates[0].Name != "Router" {
		t.Fatalf("unexpected router state: %+v", file.ContractStates)
	}
	if len(file.Templates) != 1 || file.Templates[0].Target != "" {
		t.Fatalf("unexpected templates: %+v", file.Templates)
	}
}

func hasDiagnostic(diagnostics []Diagnostic, want string) bool {
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, want) {
			return true
		}
	}
	return false
}
