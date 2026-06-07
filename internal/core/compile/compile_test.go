package compile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompilePipelineSuccess(t *testing.T) {
	input := CompileInput{
		Profile: "web",
		Entry:   "src/App.nova",
		Sources: []SourceModule{{
			Path: "src/App.nova",
			Content: `<contract state Counter>
  count: number <- 0 {
    @increment -> count + 1;
  };
/|
<template target <- web>
  <button on_press -> @increment>
    <text value <- "Count " + count /|
  /|
/|
`,
		}},
	}

	program, diagnostics := Compile(input)
	if len(diagnostics) > 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}
	if len(program.Raw) != 1 {
		t.Fatalf("raw modules = %d, want 1", len(program.Raw))
	}
	if len(program.Checked) != 1 {
		t.Fatalf("checked modules = %d, want 1", len(program.Checked))
	}
	if program.Plan.Entry != "src/App.nova" {
		t.Fatalf("plan entry = %s, want src/App.nova", program.Plan.Entry)
	}
	if len(program.Plan.Modules) != 1 || program.Plan.Modules[0].Path != "src/App.nova" {
		t.Fatalf("unexpected plan modules: %+v", program.Plan.Modules)
	}
	if program.Plan.Template.SourceFile != "src/App.nova" {
		t.Fatalf("template source = %s, want src/App.nova", program.Plan.Template.SourceFile)
	}
	if len(program.NovaIR.Modules) == 0 {
		t.Fatalf("expected lowered NovaIR modules")
	}
}

func TestCompilePipelineReturnsParseDiagnostics(t *testing.T) {
	_, diagnostics := Compile(CompileInput{
		Profile: "web",
		Entry:   "src/App.nova",
		Sources: []SourceModule{{
			Path:    "src/App.nova",
			Content: `<template target <- web>`,
		}},
	})
	if len(diagnostics) == 0 {
		t.Fatalf("expected diagnostics")
	}
	if diagnostics[0].Stage != StageParser {
		t.Fatalf("diagnostic stage = %s, want %s", diagnostics[0].Stage, StageParser)
	}
}

func TestCompileFinanceConformanceWithoutExprDiagnostics(t *testing.T) {
	root := filepath.Join("..", "..", "..", "tests", "conformance", "web", "finance")
	paths := []string{
		"src/App.nova",
		"src/FinancePure.nova",
		"src/FinanceStore.nova",
		"src/FinancePersistence.nova",
	}
	sources := make([]SourceModule, 0, len(paths))
	for _, path := range paths {
		content, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		sources = append(sources, SourceModule{Path: path, Content: string(content)})
	}
	_, diagnostics := Compile(CompileInput{
		Profile: "web",
		Entry:   "src/App.nova",
		Sources: sources,
	})
	for _, diag := range diagnostics {
		t.Errorf("%s: %s", diag.Code, diag.Message)
	}
}
