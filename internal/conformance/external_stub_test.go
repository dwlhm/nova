package conformance

import (
	"testing"

	"github.com/dwlhm/nova/internal/core/scheduler"
	"github.com/dwlhm/nova/internal/core/security"
	"github.com/dwlhm/nova/internal/provider/build"
)

func TestRunSchedulerFixtureExternalStorageCompletion(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "nova.toml", `[project]
name = "external_storage"
version = "0.1.0"
entry = "src/App.nova"

[targets.web]
renderer = "@nova/web"

[permissions]
storage.write = true
`)
	writeFixtureFile(t, root, "src/App.nova", `<import external storage from "@env/storage">
  operation set {
    input {
      key: string;
      value: unknown;
    }

    output void;
  }
/|

<contract capability StorageEvents>
  emits {
    @save: void;
    @saved: void;
    @save_failed: void;
  }
/|

<contract state AppState>
  payload: string <- "demo" {
    @save -> payload;
  };
/|

<lifecycle after @save>
  payload |> storage.set key <- "demo" value <- payload onSuccess <- @saved onFailure <- @save_failed;
/|

<template target <- web>
  <button on_press -> @save>
    <text value <- "save" /|
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
		CompleteExternals: true,
		Steps:             []SchedulerStep{{Enqueue: &EnqueueStep{Source: "host", Event: "@save"}}},
	}
	diagnostics, actual, ok := runSchedulerFixture(resolution.Plan, sources, manifest.Permissions, &expected)
	if !ok || len(diagnostics) > 0 {
		t.Fatalf("run scheduler fixture: ok=%v diagnostics=%+v", ok, diagnostics)
	}
	if len(actual.ExternalCalls) != 1 || actual.ExternalCalls[0].OnSuccess != "@saved" {
		t.Fatalf("external calls = %+v", actual.ExternalCalls)
	}
	if len(actual.Events) != 2 || actual.Events[1].Name != "@saved" {
		t.Fatalf("events = %+v, want @saved completion", actual.Events)
	}
}

func TestRunSchedulerFixtureExternalStorageFailureAndDenied(t *testing.T) {
	appNova := `<import external storage from "@env/storage">
  operation set {
    input {
      key: string;
      value: unknown;
    }

    output void;
  }
/|

<contract capability StorageEvents>
  emits {
    @save: void;
    @saved: void;
    @save_failed: void;
  }
/|

<contract state AppState>
  payload: string <- "demo" {
    @save -> payload;
  };
/|

<lifecycle after @save>
  payload |> storage.set key <- "demo" value <- payload onSuccess <- @saved onFailure <- @save_failed;
/|

<template target <- web>
  <button on_press -> @save>
    <text value <- "save" /|
  /|
/|`

	cases := []struct {
		name               string
		stub               string
		runtimePermissions []security.Permission
		wantEvent          scheduler.SchedulerEvent
	}{
		{name: "failure", stub: "failure", wantEvent: "@save_failed"},
		{name: "denied", stub: "", runtimePermissions: []security.Permission{}, wantEvent: "@save_failed"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFixtureFile(t, root, "nova.toml", `[project]
name = "external_storage"
version = "0.1.0"
entry = "src/App.nova"

[targets.web]
renderer = "@nova/web"

[permissions]
storage.write = true
`)
			writeFixtureFile(t, root, "src/App.nova", appNova)

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
				CompleteExternals:  true,
				ExternalStub:       tc.stub,
				RuntimePermissions: tc.runtimePermissions,
				Steps:              []SchedulerStep{{Enqueue: &EnqueueStep{Source: "host", Event: "@save"}}},
			}
			diagnostics, actual, ok := runSchedulerFixture(resolution.Plan, sources, manifest.Permissions, &expected)
			if !ok || len(diagnostics) > 0 {
				t.Fatalf("run scheduler fixture: ok=%v diagnostics=%+v", ok, diagnostics)
			}
			if len(actual.Events) != 2 || actual.Events[1].Name != tc.wantEvent {
				t.Fatalf("events = %+v, want completion %s", actual.Events, tc.wantEvent)
			}
		})
	}
}
