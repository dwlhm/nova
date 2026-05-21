package lsp

import (
	"strings"
	"testing"
)

func TestAnalyzeDocumentPublishesSemanticDiagnosticsWithRanges(t *testing.T) {
	source := `<contract state Counter>
  count: number <- "1";
/|
`
	doc := analyzeDocument("file:///App.nova", source, 1)
	if len(doc.Diagnostics) == 0 {
		t.Fatal("expected semantic diagnostic")
	}
	diagnostic := doc.Diagnostics[0]
	if diagnostic.Code != semanticDiagnosticCode {
		t.Fatalf("diagnostic code = %s, want %s", diagnostic.Code, semanticDiagnosticCode)
	}
	if diagnostic.Range.Start.Line != 1 || diagnostic.Range.Start.Character != 19 {
		t.Fatalf("diagnostic range = %+v, want line 1 character 19", diagnostic.Range)
	}
	if !strings.Contains(diagnostic.Message, "state count initial value has type") {
		t.Fatalf("diagnostic message = %q", diagnostic.Message)
	}
}

func TestAnalyzeDocumentIndexesSymbolsForDefinitionAndReferences(t *testing.T) {
	source := `<contract state Counter>
  count: string <- "0" {
    @increment -> count;
  };
/|
<template>
  <button on_press -> @increment>
    count
  /|
/|
`
	doc := analyzeDocument("file:///App.nova", source, 1)
	if len(doc.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", doc.Diagnostics)
	}

	position := positionAt(source, strings.LastIndex(source, "count"))
	sym, ok := symbolAtPosition(doc, position)
	if !ok {
		t.Fatal("expected symbol at template state reference")
	}
	if sym.Name != "count" || sym.Detail != "state count: string" {
		t.Fatalf("symbol = %+v, want count state detail", sym)
	}

	refs := doc.references(sym.Key)
	if len(refs) != 3 {
		t.Fatalf("references = %d, want 3", len(refs))
	}
}

func TestCompletionItemsUseBuiltInCapabilityContext(t *testing.T) {
	source := `<contract state Counter>
  count: string <- "0" {
    @increment -> count;
  };
/|
<template>
  <button on_press -> @increment>
    <text value <- count /|
  /|
/|
`
	doc := analyzeDocument("file:///App.nova", source, 1)
	if len(doc.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", doc.Diagnostics)
	}

	attrItems := completionItems(doc, positionAt(source, strings.Index(source, "<button")+len("<button ")))
	if !hasCompletion(attrItems, "on_press -> ") || !hasCompletion(attrItems, "label <- ") {
		t.Fatalf("button completions missing default attrs/events: %+v", completionLabels(attrItems))
	}

	eventItems := completionItems(doc, positionAt(source, strings.Index(source, "on_press -> ")+len("on_press -> ")))
	if !hasCompletion(eventItems, "@increment") {
		t.Fatalf("event target completions missing scheduler event: %+v", completionLabels(eventItems))
	}

	exprItems := completionItems(doc, positionAt(source, strings.Index(source, "value <- ")+len("value <- ")))
	if !hasCompletion(exprItems, "count") {
		t.Fatalf("binding completions missing state symbol: %+v", completionLabels(exprItems))
	}
}

func TestBuiltInCapabilityHover(t *testing.T) {
	source := `<template>
  <button on_press -> @save /|
/|
`
	doc := analyzeDocument("file:///App.nova", source, 1)
	tok, ok := tokenAtPosition(doc, positionAt(source, strings.Index(source, "button")))
	if !ok {
		t.Fatal("expected token at button")
	}
	primitive, ok := builtins().primitive(tok.Literal)
	if !ok {
		t.Fatal("expected built-in primitive for button")
	}
	hover := primitiveHover(primitive)
	if !strings.Contains(hover, "on_press") || !strings.Contains(hover, "label?: string") {
		t.Fatalf("hover = %q, want button event and prop details", hover)
	}
}

func TestApplyContentChangeSupportsIncrementalRanges(t *testing.T) {
	text := "one\ntwo\n"
	changed := applyContentChange(text, textDocumentContentChange{
		Range: &Range{
			Start: Position{Line: 1, Character: 0},
			End:   Position{Line: 1, Character: 3},
		},
		Text: "three",
	})
	if changed != "one\nthree\n" {
		t.Fatalf("changed = %q, want replacement", changed)
	}
}

func hasCompletion(items []CompletionItem, label string) bool {
	for _, item := range items {
		if item.Label == label {
			return true
		}
	}
	return false
}

func completionLabels(items []CompletionItem) []string {
	labels := make([]string, 0, len(items))
	for _, item := range items {
		labels = append(labels, item.Label)
	}
	return labels
}
