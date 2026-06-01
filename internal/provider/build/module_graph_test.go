package build

import (
	"testing"

	"github.com/dwlhm/nova/internal/project"
)

func TestResolveRejectsCyclicModuleGraph(t *testing.T) {
	a := parseNova(t, `<import B from "./B.nova" /|
<template>
  <text value <- "A" /|
/|`)
	b := parseNova(t, `<import A from "./A.nova" /|
<template>
  <text value <- "B" /|
/|`)
	result := Resolve(ResolutionInput{
		Project: project.Manifest{Project: project.Project{Entry: "src/A.nova"}},
		Target:  "web",
		Sources: []SourceFile{
			{Path: "src/A.nova", File: a},
			{Path: "src/B.nova", File: b},
		},
		TargetManifest: WebTargetManifest(),
	})
	if len(result.Diagnostics) == 0 {
		t.Fatal("expected cycle diagnostic")
	}
	if result.Diagnostics[0].Code != "NVA-MODULE-001" {
		t.Fatalf("diagnostic = %+v", result.Diagnostics[0])
	}
}
