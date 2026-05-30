package capability

import (
	"testing"

	"github.com/dwlhm/nova/internal/lexer"
	"github.com/dwlhm/nova/internal/parser"
)

func TestBuildManifestCapturesCapabilitySurface(t *testing.T) {
	input := `<import Counter from "./Counter.nova" /|
<import state count as cartCount from "./Cart.nova" /|
<import event @submit as @cart_submit from "./Cart.nova" /|
<import external storage from "@env/storage">
  operation set {
    input {
      key: string;
      value: unknown;
    }

    output void;
  }
/|
<contract type UserId /|
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
/|
`

	file, diagnostics := parser.Parse(lexer.Tokenize(input))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected parser diagnostics: %v", diagnostics)
	}

	manifest := BuildManifest("./Counter.nova", file)
	if manifest.Ref != "./Counter.nova" {
		t.Fatalf("manifest ref = %s, want ./Counter.nova", manifest.Ref)
	}
	if len(manifest.Imports) != 4 {
		t.Fatalf("imports = %d, want 4: %+v", len(manifest.Imports), manifest.Imports)
	}
	if manifest.Imports[1].EffectiveName != "cartCount" {
		t.Fatalf("state import effective name = %s, want cartCount", manifest.Imports[1].EffectiveName)
	}
	if manifest.Imports[2].EffectiveName != "@cart_submit" {
		t.Fatalf("event import effective name = %s, want @cart_submit", manifest.Imports[2].EffectiveName)
	}
	if len(manifest.Types) != 1 || manifest.Types[0].Ref.Name != "UserId" || !manifest.Types[0].Opaque {
		t.Fatalf("unexpected types: %+v", manifest.Types)
	}
	if len(manifest.States) != 1 || manifest.States[0].Ref.Name != "count" || manifest.States[0].Type != "number" {
		t.Fatalf("unexpected states: %+v", manifest.States)
	}
	if len(manifest.Events) != 2 {
		t.Fatalf("events = %d, want 2: %+v", len(manifest.Events), manifest.Events)
	}
	if len(manifest.ExternalOperations) != 1 || manifest.ExternalOperations[0].Ref != Symbol("@env/storage", "set") {
		t.Fatalf("unexpected external operations: %+v", manifest.ExternalOperations)
	}
	if len(manifest.TargetConstraints) != 1 || manifest.TargetConstraints[0].Target != "web" {
		t.Fatalf("unexpected target constraints: %+v", manifest.TargetConstraints)
	}
}

func TestValidateAcyclicModuleGraphRejectsCycles(t *testing.T) {
	diagnostics := ValidateAcyclicModuleGraph([]Manifest{
		{
			Ref: "./A.nova",
			Imports: []ImportRef{{
				Kind:   ImportCapability,
				Module: "./B.nova",
				Name:   "B",
			}},
		},
		{
			Ref: "./B.nova",
			Imports: []ImportRef{{
				Kind:   ImportCapability,
				Module: "./A.nova",
				Name:   "A",
			}},
		},
	})

	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics = %d, want 1: %+v", len(diagnostics), diagnostics)
	}
	want := []CapabilityRef{"./A.nova", "./B.nova", "./A.nova"}
	assertCycle(t, diagnostics[0].Cycle, want)
}

func TestValidateAcyclicModuleGraphAllowsExternalLeaves(t *testing.T) {
	diagnostics := ValidateAcyclicModuleGraph([]Manifest{
		{
			Ref: "./A.nova",
			Imports: []ImportRef{{
				Kind:   ImportExternal,
				Module: "@env/storage",
				Name:   "storage",
			}},
		},
	})

	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}
}

func assertCycle(t *testing.T, got []CapabilityRef, want []CapabilityRef) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("cycle = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("cycle = %+v, want %+v", got, want)
		}
	}
}
