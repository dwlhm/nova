package artifact

import (
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/core/contract"
	"github.com/dwlhm/nova/internal/core/ir"
	"github.com/dwlhm/nova/internal/project"
	"github.com/dwlhm/nova/internal/provider/build"
	"github.com/dwlhm/nova/internal/provider/shared"
)

func TestBuildAppContractOmitsCompilerInternals(t *testing.T) {
	bundle := ir.Bundle{
		App: contract.App{
			V:      contract.Version,
			Target: "web",
			Model: contract.Model{States: []contract.State{{
				Owner: "App", Name: "count", Initial: "0",
				Transitions: []contract.Transition{{Event: "@increment", Expression: "count + 1"}},
			}}},
			View: contract.View{Nodes: []contract.Node{{
				Kind:  "text",
				Props: map[string]string{"value": "state.count"},
			}}},
		},
	}
	app := BuildAppContract(bundle)
	if app.V != contract.Version {
		t.Fatalf("v = %d", app.V)
	}
	if len(app.View.Nodes) != 1 || app.View.Nodes[0].Props["value"] == "" {
		t.Fatalf("expected compiled prop expr, got %+v", app.View.Nodes[0].Props)
	}
	raw := shared.MustJSON(app)
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
	sources := []build.SourceFile{{Path: "src/App.nova", File: source}}
	input, _ := testGenerateInput(t, manifest, plan.Plan, sources, targetManifest)
	files, diagnostics := Generate(input)
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
