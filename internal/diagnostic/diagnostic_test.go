package diagnostic

import (
	"strings"
	"testing"
)

func TestStableSortOrdersDiagnosticsBySourceAndCode(t *testing.T) {
	diagnostics := []Diagnostic{
		{Code: "NVA-TYPE-002", Severity: SeverityError, Message: "second", Span: span("src/B.nova", 1, 1)},
		{Code: "NVA-LEX-001", Severity: SeverityError, Message: "first", Span: span("src/A.nova", 8, 2)},
		{Code: "NVA-PARSE-001", Severity: SeverityWarning, Message: "also first", Span: span("src/A.nova", 8, 2)},
		{Code: "NVA-SEC-001", Severity: SeverityError, Message: "global"},
	}

	sorted := StableSort(diagnostics)

	want := []string{"NVA-SEC-001", "NVA-LEX-001", "NVA-PARSE-001", "NVA-TYPE-002"}
	for i, code := range want {
		if sorted[i].Code != code {
			t.Fatalf("diagnostic %d = %s, want %s in %+v", i, sorted[i].Code, code, sorted)
		}
	}
	if diagnostics[0].Code != "NVA-TYPE-002" {
		t.Fatalf("StableSort should not mutate input: %+v", diagnostics)
	}
}

func TestDiagnosticJSONLinesAndErrorDetection(t *testing.T) {
	diagnostics := []Diagnostic{
		{
			Code:     "NVA-PURITY-001",
			Severity: SeverityError,
			Message:  "func cannot call external operation storage.set",
			Span:     span("src/Counter.nova", 12, 11),
			Hint:     "external operations are only valid in lifecycle blocks",
		},
		{Code: "NVA-RENDER-010", Severity: SeverityInfo, Message: "rendered ViewIR"},
	}

	if !HasErrors(diagnostics) {
		t.Fatalf("expected error diagnostic")
	}

	lines, err := JSONLines(diagnostics)
	if err != nil {
		t.Fatalf("JSONLines returned error: %v", err)
	}
	if !strings.Contains(lines, `"code":"NVA-PURITY-001"`) || !strings.Contains(lines, `"severity":"error"`) {
		t.Fatalf("diagnostic stream should include stable code and severity, got %s", lines)
	}
	if !strings.HasSuffix(lines, "\n") {
		t.Fatalf("diagnostic stream should end with newline, got %q", lines)
	}
}

func span(file string, line int, column int) *SourceSpan {
	return &SourceSpan{File: file, Line: line, Column: column, Offset: line * 100, Length: column}
}
