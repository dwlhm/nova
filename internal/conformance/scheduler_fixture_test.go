package conformance

import (
	"encoding/json"
	"testing"

	"github.com/dwlhm/nova/internal/provider/build"
)

func TestRunSchedulerFixtureCounterIncrement(t *testing.T) {
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

	manifest, _ := readFixtureManifest(root)
	sources, _ := readFixtureSources(root, manifest.Project.Entry)
	resolution := build.Resolve(build.ResolutionInput{
		Project:        manifest,
		Target:         "web",
		Sources:        sources,
		TargetManifest: mustTargetManifest(t, "web"),
	})
	if len(resolution.Diagnostics) > 0 {
		t.Fatalf("resolution diagnostics = %+v", resolution.Diagnostics)
	}

	expected := ExpectedScheduler{
		Steps: []SchedulerStep{{Enqueue: &EnqueueStep{Source: "host", Event: "@increment"}}},
	}
	diagnostics, actual, ok := runSchedulerFixture(resolution.Plan, sources, manifest.Permissions, &expected)
	if !ok || len(diagnostics) > 0 {
		t.Fatalf("run scheduler fixture: ok=%v diagnostics=%+v", ok, diagnostics)
	}

	expected.Trace = actual
	encoded, err := json.MarshalIndent(expected, "", "  ")
	if err != nil {
		t.Fatalf("marshal trace: %v", err)
	}
	t.Logf("scheduler fixture trace:\n%s", string(encoded))

	diagnostics = CompareTrace(expected.Trace, actual)
	if len(diagnostics) > 0 {
		t.Fatalf("compare trace diagnostics = %+v", diagnostics)
	}
}

func TestRunFixtureDirComparesSchedulerTrace(t *testing.T) {
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
    },
    "scheduler": {
      "steps": [
        { "enqueue": { "source": "host", "event": "@increment" } }
      ],
      "trace": {
        "events": [
          { "sequence": 1, "source": "host", "name": "@increment" }
        ],
        "commits": [
          {
            "sequence": 1,
            "event": "@increment",
            "committed": true,
            "changes": [
              { "key": { "owner": "Counter", "name": "count" }, "before": 0, "after": 1 }
            ],
            "invalidations": [
              { "owner": "Counter", "name": "count" }
            ]
          }
        ]
      }
    }
  }
}
`)

	result := RunFixtureDir(root)
	if !result.Passed() {
		t.Fatalf("fixture diagnostics = %+v", result.Diagnostics)
	}
}

func TestRunSchedulerFixtureLifecycleBeforeAfterMount(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "nova.toml", `[project]
name = "lifecycle"
version = "0.1.0"
entry = "src/App.nova"

[targets.web]
renderer = "@nova/web"
`)
	writeFixtureFile(t, root, "src/App.nova", `<contract capability AppEvents>
  emits {
    @boot: void;
    @increment: void;
    @seen_before: void;
    @seen_after: void;
  }
/|

<contract state Counter>
  count: number <- 0 {
    @increment -> count + 1;
    @boot -> 5;
  };
/|

<lifecycle mount>
  void -> @boot;
/|

<lifecycle before @increment>
  void -> @seen_before;
/|

<lifecycle after @increment>
  void -> @seen_after;
/|

<template target <- web>
  <button on_press -> @increment>
    <text value <- count /|
/|
/|`)

	manifest, _ := readFixtureManifest(root)
	sources, _ := readFixtureSources(root, manifest.Project.Entry)
	resolution := build.Resolve(build.ResolutionInput{
		Project:        manifest,
		Target:         "web",
		Sources:        sources,
		TargetManifest: mustTargetManifest(t, "web"),
	})
	if len(resolution.Diagnostics) > 0 {
		t.Fatalf("resolution diagnostics = %+v", resolution.Diagnostics)
	}

	expected := ExpectedScheduler{
		Steps: []SchedulerStep{
			{Lifecycle: &LifecycleStep{Phase: "mount", Source: "runtime"}},
			{Enqueue: &EnqueueStep{Source: "host", Event: "@increment"}},
		},
	}
	diagnostics, actual, ok := runSchedulerFixture(resolution.Plan, sources, manifest.Permissions, &expected)
	if !ok || len(diagnostics) > 0 {
		t.Fatalf("run scheduler fixture: ok=%v diagnostics=%+v", ok, diagnostics)
	}

	if len(actual.LifecycleCalls) < 2 {
		t.Fatalf("lifecycle calls = %+v, want before and after", actual.LifecycleCalls)
	}
	var incrementCommit *CommitTrace
	for index := range actual.Commits {
		if actual.Commits[index].Event == "@increment" {
			incrementCommit = &actual.Commits[index]
		}
	}
	if incrementCommit == nil || len(incrementCommit.Changes) != 1 {
		t.Fatalf("increment commit = %+v", actual.Commits)
	}
	if incrementCommit.Changes[0].After != float64(6) {
		t.Fatalf("final count = %+v, want 6", incrementCommit.Changes[0].After)
	}
}

func mustTargetManifest(t *testing.T, targetID string) build.TargetManifest {
	t.Helper()
	manifest, ok := build.TargetManifestFor(targetID)
	if !ok {
		t.Fatalf("unsupported target %s", targetID)
	}
	return manifest
}
