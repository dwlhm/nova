package conformance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dwlhm/nova/internal/scheduler"
)

func TestTraceSchedulerResultsCapturesEventsCommitsAndExternalCalls(t *testing.T) {
	key := scheduler.Key("Counter", "count")
	runtime := scheduler.NewRuntime(
		[]scheduler.StateCell{
			scheduler.NewStateCell("Counter", "count", "number", 0, scheduler.On("@increment", func(snapshot scheduler.Snapshot, event scheduler.EventEnvelope) (scheduler.DataValue, error) {
				return snapshot.MustValue(key).(int) + 1, nil
			})),
		},
		[]scheduler.LifecycleHandler{
			scheduler.After("Counter", "@increment", func(ctx scheduler.LifecycleContext) (scheduler.LifecycleOutput, error) {
				return scheduler.LifecycleOutput{
					External: []scheduler.ExternalOperationRequest{
						scheduler.ExternalOperation("Counter", "storage", "set", map[string]scheduler.DataValue{"value": ctx.Snapshot.MustValue(key)}, "void", "@stored", "@store_failed"),
					},
				}, nil
			}),
		},
	)
	runtime, _, _ = scheduler.Enqueue(runtime, "host", "@increment", nil)
	_, results := scheduler.Drain(runtime)

	trace := TraceSchedulerResults(results)

	if len(trace.Events) != 1 || trace.Events[0].Name != "@increment" {
		t.Fatalf("events trace = %+v", trace.Events)
	}
	if len(trace.Commits) != 1 || len(trace.Commits[0].Changes) != 1 {
		t.Fatalf("commits trace = %+v", trace.Commits)
	}
	if len(trace.ExternalCalls) != 1 || trace.ExternalCalls[0].Operation != "set" {
		t.Fatalf("external trace = %+v", trace.ExternalCalls)
	}
}

func TestCompareTraceReportsStableDiagnostics(t *testing.T) {
	expected := Trace{Events: []EventTrace{{Sequence: 1, Source: "host", Name: "@increment"}}}
	actual := Trace{Events: []EventTrace{{Sequence: 1, Source: "host", Name: "@decrement"}}}

	diagnostics := CompareTrace(expected, actual)

	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics = %+v, want one mismatch", diagnostics)
	}
	if diagnostics[0].Code != "NVA-CONFORMANCE-001" {
		t.Fatalf("diagnostic code = %s, want NVA-CONFORMANCE-001", diagnostics[0].Code)
	}
}

func TestRunFixtureDirComparesExpectedArtifactAndViewMetadata(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "nova.toml", `[project]
name = "counter"
version = "0.1.0"
entry = "src/App.nova"

[targets.web]
renderer = "@nova/web"
`)
	writeFixtureFile(t, root, "src/App.nova", `<contract state Counter>
  count: number <- 0 {
    @increment -> count + 1;
  };
/|
<template target <- web>
  <button on_press -> @increment>
    <text value <- "Count " + count /|
  /|
/|`)
	writeFixtureFile(t, root, "nova.conformance.json", `{
  "target": "web",
  "expected": {
    "diagnosticCodes": [],
    "artifact": {
      "target": "web",
      "entry": "src/App.nova",
      "modules": ["src/App.nova"],
      "permissions": []
    },
    "view": {
      "bindings": 1,
      "eventRoutes": 1,
      "pages": 0,
      "routePatterns": []
    }
  }
}
`)

	result := RunFixtureDir(root)
	if !result.Passed() {
		t.Fatalf("fixture diagnostics = %+v", result.Diagnostics)
	}
	if result.Name == "" || result.Target != "web" {
		t.Fatalf("fixture result = %+v, want named web result", result)
	}
}

func TestRunFixtureDirReportsMismatches(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "nova.toml", `[project]
name = "counter"
version = "0.1.0"
entry = "src/App.nova"

[targets.web]
renderer = "@nova/web"
`)
	writeFixtureFile(t, root, "src/App.nova", `<template target <- web>
  <text value <- "web" /|
/|`)
	writeFixtureFile(t, root, "nova.conformance.json", `{
  "target": "web",
  "expected": {
    "diagnosticCodes": [],
    "view": {
      "bindings": 9,
      "eventRoutes": 0,
      "pages": 0,
      "routePatterns": []
    }
  }
}
`)

	result := RunFixtureDir(root)
	if result.Passed() {
		t.Fatalf("fixture should fail when expected view metadata mismatches")
	}
	assertConformanceDiagnostic(t, result.Diagnostics, "NVA-CONFORMANCE-012")
}

func TestRunFixtureDirComparesExpectedRoutePatterns(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "nova.toml", `[project]
name = "routing"
version = "0.1.0"
entry = "src/App.nova"

[targets.web]
renderer = "@nova/web"
`)
	writeFixtureFile(t, root, "src/App.nova", `<contract state Router>
  route: unknown <- { path <- "/"; };
/|
<template target <- web>
  <surface>
    <page path <- "/users/:id">
      <text value <- "User" /|
    /|
    <page path <- "/docs/*">
      <text value <- "Docs" /|
    /|
    <page path <- "*">
      <text value <- "Missing" /|
    /|
  /|
/|`)
	writeFixtureFile(t, root, "nova.conformance.json", `{
  "target": "web",
  "expected": {
    "diagnosticCodes": [],
    "view": {
      "bindings": 0,
      "eventRoutes": 0,
      "pages": 3,
      "routePatterns": ["/users/:id", "/docs/*", "*"]
    }
  }
}
`)

	result := RunFixtureDir(root)
	if !result.Passed() {
		t.Fatalf("fixture diagnostics = %+v", result.Diagnostics)
	}
}

func writeFixtureFile(t *testing.T, root string, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, filepath.FromSlash(path))), 0o755); err != nil {
		t.Fatalf("mkdir fixture path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(path)), []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
}

func assertConformanceDiagnostic(t *testing.T, diagnostics []Diagnostic, code string) {
	t.Helper()

	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("missing conformance diagnostic %s in %+v", code, diagnostics)
}
