package style

import (
	"fmt"
	"sort"
)

// WebStylesheet is a web-only raw CSS file referenced by <import stylesheet>.
type WebStylesheet struct {
	SourcePath string
	Scope      Scope
	Content    string
}

// Bundle is the target-neutral style IR passed from the host edge into providers.
type Bundle struct {
	Sheets         []Sheet
	WebStylesheets []WebStylesheet
}

// ReadFile loads style file bytes for a project-relative path.
type ReadFile func(path string) ([]byte, error)

// BuildBundle resolves compile-time import refs into IR. The targetID controls
// portable validation (android) and whether web-only stylesheets are included.
func BuildBundle(targetID string, imports []ImportRef, read ReadFile) (Bundle, []Diagnostic, bool) {
	bundle := Bundle{
		Sheets:         make([]Sheet, 0),
		WebStylesheets: make([]WebStylesheet, 0),
	}
	diagnostics := make([]Diagnostic, 0)
	failed := false

	for _, ref := range imports {
		if ref.Kind == ImportStylesheet && targetID == "android" {
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "NVA-STYLE-013",
				Message: fmt.Sprintf("omitting web-only stylesheet %s on android", ref.Path),
			})
			continue
		}
		content, err := read(ref.Path)
		if err != nil {
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "NVA-STYLE-002",
				Message: fmt.Sprintf("read stylesheet %s: %s", ref.Path, err.Error()),
			})
			failed = true
			continue
		}
		switch ref.Kind {
		case ImportStyle:
			sheet, sheetDiagnostics := ParseDocument(ref.Path, string(content))
			refFailed := len(sheetDiagnostics) > 0
			diagnostics = append(diagnostics, sheetDiagnostics...)
			if targetID == "android" {
				filtered, stateDiags := FilterAndroidStates(sheet.States)
				diagnostics = append(diagnostics, stateDiags...)
				sheet.States = filtered
				for _, item := range ValidatePortableSubset(sheet) {
					diagnostics = append(diagnostics, item)
					refFailed = true
				}
			}
			if refFailed {
				failed = true
				continue
			}
			bundle.Sheets = append(bundle.Sheets, sheet)
		case ImportStylesheet:
			bundle.WebStylesheets = append(bundle.WebStylesheets, WebStylesheet{
				SourcePath: ref.Path,
				Scope:      ScopeGlobal,
				Content:    string(content),
			})
		}
	}

	sort.Slice(bundle.Sheets, func(i, j int) bool {
		return bundle.Sheets[i].SourcePath < bundle.Sheets[j].SourcePath
	})
	sort.Slice(bundle.WebStylesheets, func(i, j int) bool {
		return bundle.WebStylesheets[i].SourcePath < bundle.WebStylesheets[j].SourcePath
	})
	return bundle, diagnostics, !failed
}

// NormalizeScope returns global unless the scope is explicitly app-scoped.
func NormalizeScope(scope Scope) Scope {
	if scope == ScopeApp {
		return ScopeApp
	}
	return ScopeGlobal
}
