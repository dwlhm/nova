package view

import (
	"testing"

	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/parser"
	"github.com/dwlhm/nova/internal/core/scheduler"
)

func TestProjectBuildsRendererNeutralIRAndDependencyMetadata(t *testing.T) {
	input := `<contract state Counter>
  count: number <- 0 {
    @increment -> count + 1;
  };
/|
<template target <- web>
  <button on_press -> @increment>
    <text value <- count /|
  /|
/|`

	file, diagnostics := parser.Parse(lexer.Tokenize(input))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected parser diagnostics: %v", diagnostics)
	}

	ir, viewDiagnostics := Project(file.Templates[0], map[string]bool{"count": true})
	if len(viewDiagnostics) != 0 {
		t.Fatalf("unexpected view diagnostics: %v", viewDiagnostics)
	}
	if ir.Target != "web" || len(ir.Nodes) != 1 {
		t.Fatalf("unexpected ir root: %+v", ir)
	}
	button := ir.Nodes[0]
	if button.Kind != "button" || len(button.Children) != 1 {
		t.Fatalf("unexpected button node: %+v", button)
	}
	if button.Events["on_press"].Event != scheduler.SchedulerEvent("@increment") {
		t.Fatalf("unexpected route: %+v", button.Events)
	}
	if len(ir.Metadata.EventRoutes) != 1 || ir.Metadata.EventRoutes[0].Event != "@increment" {
		t.Fatalf("event metadata = %+v, want @increment", ir.Metadata.EventRoutes)
	}
	if len(ir.Metadata.Bindings) != 1 || ir.Metadata.Bindings[0].Prop != "value" || ir.Metadata.Bindings[0].States[0] != "count" {
		t.Fatalf("binding metadata = %+v, want text value depends on count", ir.Metadata.Bindings)
	}
}

func TestProjectKeepsComponentEventRoutesAsData(t *testing.T) {
	input := `<template>
  <Button label <- "Save" @pressed -> @save /|
/|`
	file, diagnostics := parser.Parse(lexer.Tokenize(input))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected parser diagnostics: %v", diagnostics)
	}

	ir, viewDiagnostics := Project(file.Templates[0], nil)
	if len(viewDiagnostics) != 0 {
		t.Fatalf("unexpected view diagnostics: %v", viewDiagnostics)
	}
	if got := ir.Nodes[0].Events["@pressed"].Event; got != "@save" {
		t.Fatalf("component event route = %s, want @save", got)
	}
}

func TestProjectRecordsPageProjectionMetadata(t *testing.T) {
	input := `<contract type Route>
  path: string;
/|
<contract state Router>
  route: Route <- { path <- "/"; };
/|
<template target <- web>
  <surface>
    <page path <- "/">
      <text value <- "Home" /|
    /|
    <page path <- "/settings">
      <text value <- route.path /|
    /|
  /|
/|`
	file, diagnostics := parser.Parse(lexer.Tokenize(input))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected parser diagnostics: %v", diagnostics)
	}

	ir, viewDiagnostics := Project(file.Templates[0], map[string]bool{"route": true})
	if len(viewDiagnostics) != 0 {
		t.Fatalf("unexpected view diagnostics: %v", viewDiagnostics)
	}
	if len(ir.Metadata.Pages) != 2 {
		t.Fatalf("pages = %+v, want 2 page refs", ir.Metadata.Pages)
	}
	if got := ir.Metadata.Pages[0].Path.Text; got != "/" {
		t.Fatalf("first page path = %s, want /", got)
	}
	if got := ir.Metadata.Pages[1].Path.Text; got != "/settings" {
		t.Fatalf("second page path = %s, want /settings", got)
	}
	if len(ir.Metadata.Pages[0].NodePath) != 2 || ir.Metadata.Pages[0].NodePath[0] != 0 || ir.Metadata.Pages[0].NodePath[1] != 0 {
		t.Fatalf("first page node path = %+v, want [0 0]", ir.Metadata.Pages[0].NodePath)
	}
}
