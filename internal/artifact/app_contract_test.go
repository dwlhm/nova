package artifact

import (
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/build"
	"github.com/dwlhm/nova/internal/contract"
	"github.com/dwlhm/nova/internal/lexer"
	"github.com/dwlhm/nova/internal/project"
	"github.com/dwlhm/nova/internal/security"
	"github.com/dwlhm/nova/internal/view"
)

func TestBuildAppContractOmitsCompilerInternals(t *testing.T) {
	bundle := irBundle{
		Target: "web",
		Entry:  "src/App.nova",
		Model: appModel{States: []stateModel{{
			Owner: "App", Name: "count", Initial: "0",
			Transitions: []transitionModel{{Event: "@increment", Expression: "count + 1"}},
		}}},
		ViewIR: view.IR{
			Nodes: []view.Node{{
				Kind:  "text",
				Props: map[string]view.Binding{"value": {Text: "count", Tokens: []lexer.Token{{Type: lexer.IDENT, Literal: "count"}}}},
			}},
		},
	}
	app := buildAppContract(bundle, []security.Permission{"storage.read"})
	if app.V != contract.Version {
		t.Fatalf("v = %d", app.V)
	}
	if len(app.View.Nodes) != 1 || app.View.Nodes[0].Props["value"] == "" {
		t.Fatalf("expected compiled prop expr, got %+v", app.View.Nodes[0].Props)
	}
	raw := mustJSON(app)
	for _, forbidden := range []string{"capabilityManifests", "viewIR", "Tokens", "capabilityManifest"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("contract must not contain %q", forbidden)
		}
	}
}

func TestGenerateWebUsesAppContractNotNovaIR(t *testing.T) {
	source := parseNova(t, `<contract state App>
  n: number <- 0 { @inc -> n + 1; };
/|
<template target <- web>
  <text value <- n /|
/|`)
	manifest := project.Manifest{Project: project.Project{Name: "x", Version: "0.1.0", Entry: "src/App.nova"}}
	targetManifest := build.WebTargetManifest()
	plan := build.Resolve(build.ResolutionInput{
		Project: manifest, Target: "web",
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	files, diagnostics := Generate(GenerateInput{
		Project: manifest, Plan: plan.Plan,
		Sources: []build.SourceFile{{Path: "src/App.nova", File: source}}, TargetManifest: targetManifest,
	})
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", diagnostics)
	}
	assertArtifactFile(t, files, "build/web/app.contract.json", `"v": 1`)
	assertArtifactFile(t, files, "build/web/app.bundle.js", `"v": 1`)
	assertArtifactFile(t, files, "build/web/build.manifest.json", `"contractVersion": 1`)
	for _, path := range []string{"build/web/app.nova-ir.json", "build/web/metadata.json", "build/web/permissions.json"} {
		for _, file := range files {
			if file.Path == path {
				t.Fatalf("unexpected legacy artifact %s", path)
			}
		}
	}
}

// parseNova and assertArtifactFile are defined in artifact_test.go
