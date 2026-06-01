package build

import (
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/parser"
	"github.com/dwlhm/nova/internal/core/security"
	"github.com/dwlhm/nova/internal/packages"
	"github.com/dwlhm/nova/internal/project"
)

func TestResolveBuildPlanSelectsExactTargetTemplateAndImplementation(t *testing.T) {
	entry := parseNova(t, `<import Counter from "./Counter.nova" /|
<import external storage from "@env/storage">
  operation load {
    output unknown;
  }
/|
<template>
  <text value <- "portable" /|
/|
<template target <- web>
  <text value <- "web" /|
/|`)
	counter := parseNova(t, `<contract state Counter>
  count: number <- 0 {
    @increment -> count + 1;
  };
/|`)

	result := Resolve(ResolutionInput{
		Project: project.Manifest{
			Project:     project.Project{Name: "audiolab", Version: "0.1.0", Entry: "src/App.nova"},
			Permissions: project.PermissionMap{"storage.read": true},
		},
		Target: "web",
		Sources: []SourceFile{
			{Path: "src/Counter.nova", File: counter},
			{Path: "src/App.nova", File: entry},
		},
		TargetManifest: TargetManifest{
			ID:       "web",
			Families: []string{"browser"},
			ExternalCapabilities: []ExternalCapability{
				{
					Source: "@env/storage",
					Operations: []ExternalOperation{
						{
							Name:        "load",
							Permissions: []security.Permission{"storage.read"},
							Implementations: []Implementation{
								{Path: "platform/common/storage.common.js", Common: true},
								{Path: "platform/web/storage.web.js", Target: "web"},
							},
						},
					},
				},
			},
			PermissionMappings: security.PermissionSet("storage.read"),
		},
	})

	if len(result.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", result.Diagnostics)
	}
	if len(result.Plan.Modules) != 2 || result.Plan.Modules[0].Path != "src/App.nova" || result.Plan.Modules[1].Path != "src/Counter.nova" {
		t.Fatalf("modules = %+v, want deterministic entry-first graph", result.Plan.Modules)
	}
	if result.Plan.Template.Target != "web" || result.Plan.Template.Selection != SelectionExactTarget {
		t.Fatalf("selected template = %+v, want exact web", result.Plan.Template)
	}
	if len(result.Plan.ExternalOperations) != 1 {
		t.Fatalf("external operations = %+v, want one", result.Plan.ExternalOperations)
	}
	operation := result.Plan.ExternalOperations[0]
	if operation.Implementation.Path != "platform/web/storage.web.js" {
		t.Fatalf("implementation = %+v, want exact web implementation", operation.Implementation)
	}
	if len(result.Plan.Permissions) != 1 || result.Plan.Permissions[0] != "storage.read" {
		t.Fatalf("permissions = %+v, want storage.read", result.Plan.Permissions)
	}
	if result.Plan.Artifact.Target != "web" || result.Plan.Artifact.Entry != "src/App.nova" {
		t.Fatalf("artifact metadata = %+v", result.Plan.Artifact)
	}
}

func TestResolveFallsBackToTargetFamilyBeforeCommonImplementation(t *testing.T) {
	entry := parseNova(t, `<import external storage from "@env/storage">
  operation load {
    output unknown;
  }
/|
<template>
  <text value <- "portable" /|
/|`)

	result := Resolve(ResolutionInput{
		Project: project.Manifest{
			Project: project.Project{Name: "audiolab", Version: "0.1.0", Entry: "src/App.nova"},
		},
		Target:  "android",
		Sources: []SourceFile{{Path: "src/App.nova", File: entry}},
		TargetManifest: TargetManifest{
			ID:       "android",
			Families: []string{"mobile"},
			ExternalCapabilities: []ExternalCapability{
				{
					Source: "@env/storage",
					Operations: []ExternalOperation{
						{
							Name: "load",
							Implementations: []Implementation{
								{Path: "platform/common/storage.common.js", Common: true},
								{Path: "platform/mobile/storage.mobile.java", Family: "mobile"},
							},
						},
					},
				},
			},
		},
	})

	if len(result.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", result.Diagnostics)
	}
	if got := result.Plan.ExternalOperations[0].Implementation.Path; got != "platform/mobile/storage.mobile.java" {
		t.Fatalf("implementation = %s, want mobile family fallback", got)
	}
	if result.Plan.Template.Selection != SelectionPolymorphic {
		t.Fatalf("template selection = %+v, want polymorphic fallback", result.Plan.Template)
	}
}

func TestResolveBuildPlanMergesRendererPackageDictionary(t *testing.T) {
	entry := parseNova(t, `<template>
  <sparkline data <- points /|
/|`)

	result := Resolve(ResolutionInput{
		Project: project.Manifest{
			Project: project.Project{Name: "charts", Version: "0.1.0", Entry: "src/App.nova"},
			Renderer: project.RendererConfig{
				UnknownKind:       project.RendererUnknownKindError,
				ExtensionPackages: []project.RendererPackageRef{{Name: "@acme/charts", Constraint: "*"}},
			},
		},
		Target:         "web",
		Sources:        []SourceFile{{Path: "src/App.nova", File: entry}},
		TargetManifest: TargetManifest{ID: "web"},
		PackageGraph: packages.ResolvedGraph{RendererExtensions: []packages.ResolvedRendererPackage{{
			Name:          "@acme/charts",
			Version:       "1.0.0",
			TargetAdapter: "platform/web/register.web.js",
			Primitives: []packages.RendererPrimitive{{
				Package: "@acme/charts",
				Kind:    "sparkline",
				Props:   []packages.RendererField{{Name: "data", Type: "unknown"}},
				Targets: map[string]packages.RendererTarget{"web": {Strategy: "adapter"}},
			}},
		}}},
	})

	if len(result.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", result.Diagnostics)
	}
	if !rendererPlanHasPrimitive(result.Plan.Renderer, "sparkline") {
		t.Fatalf("renderer plan = %+v, want sparkline", result.Plan.Renderer)
	}
	if len(result.Plan.Renderer.Extensions) != 1 || result.Plan.Renderer.Extensions[0].AdapterPath != "platform/web/register.web.js" {
		t.Fatalf("renderer extensions = %+v", result.Plan.Renderer.Extensions)
	}
}

func TestResolveBuildPlanReportsRendererConflictsAndMissingAdapters(t *testing.T) {
	entry := parseNova(t, `<template>
  <sparkline data <- points /|
/|`)
	result := Resolve(ResolutionInput{
		Project:        project.Manifest{Project: project.Project{Name: "charts", Version: "0.1.0", Entry: "src/App.nova"}},
		Target:         "web",
		Sources:        []SourceFile{{Path: "src/App.nova", File: entry}},
		TargetManifest: TargetManifest{ID: "web"},
		PackageGraph: packages.ResolvedGraph{RendererExtensions: []packages.ResolvedRendererPackage{
			{
				Name:    "@acme/one",
				Version: "1.0.0",
				Primitives: []packages.RendererPrimitive{{
					Package: "@acme/one",
					Kind:    "sparkline",
					Targets: map[string]packages.RendererTarget{"web": {Strategy: "adapter"}},
				}},
			},
			{
				Name:          "@acme/two",
				Version:       "1.0.0",
				TargetAdapter: "platform/web/register.web.js",
				Primitives: []packages.RendererPrimitive{{
					Package: "@acme/two",
					Kind:    "sparkline",
					Targets: map[string]packages.RendererTarget{"web": {Strategy: "adapter"}},
				}},
			},
		}},
	})

	assertBuildDiagnostic(t, result.Diagnostics, "NVA-RENDER-003")
	assertBuildDiagnostic(t, result.Diagnostics, "NVA-RENDER-004")
}

func TestAndroidTargetManifestUsesJavaEnvironmentAdapters(t *testing.T) {
	manifest := AndroidTargetManifest()

	for _, capability := range manifest.ExternalCapabilities {
		for _, operation := range capability.Operations {
			for _, implementation := range operation.Implementations {
				if implementation.Target != "android" {
					t.Fatalf("%s.%s target = %q, want android", capability.Source, operation.Name, implementation.Target)
				}
				if strings.Contains(implementation.Path, ".kt") {
					t.Fatalf("%s.%s implementation = %s, want Java production adapter", capability.Source, operation.Name, implementation.Path)
				}
				if !strings.HasSuffix(implementation.Path, ".android.java") {
					t.Fatalf("%s.%s implementation = %s, want .android.java", capability.Source, operation.Name, implementation.Path)
				}
			}
		}
	}
}

func rendererPlanHasPrimitive(plan RendererPlan, kind string) bool {
	for _, primitive := range plan.Primitives {
		if primitive.Kind == kind {
			return true
		}
	}
	return false
}

func TestResolveReportsMissingTargetCapabilityWithCandidateContext(t *testing.T) {
	entry := parseNova(t, `<import external storage from "@env/storage">
  operation load {
    output unknown;
  }
/|
<template target <- web>
  <text value <- "web only" /|
/|`)

	result := Resolve(ResolutionInput{
		Project: project.Manifest{
			Project: project.Project{Name: "audiolab", Version: "0.1.0", Entry: "src/App.nova"},
		},
		Target:  "android",
		Sources: []SourceFile{{Path: "src/App.nova", File: entry}},
		TargetManifest: TargetManifest{
			ID: "android",
			ExternalCapabilities: []ExternalCapability{
				{
					Source: "@env/storage",
					Operations: []ExternalOperation{
						{
							Name: "load",
							Implementations: []Implementation{
								{Path: "platform/web/storage.web.js", Target: "web"},
							},
						},
					},
				},
			},
		},
	})

	assertBuildDiagnostic(t, result.Diagnostics, "NVA-TARGET-004")
	assertBuildDiagnostic(t, result.Diagnostics, "@env/storage.load has no android implementation")
	assertBuildDiagnostic(t, result.Diagnostics, "requested by src/App.nova")
	assertBuildDiagnostic(t, result.Diagnostics, "considered: platform/web/storage.web.js")
	assertBuildDiagnostic(t, result.Diagnostics, "no template for target android")
}

func TestResolveReportsExternalOperationSignatureMismatch(t *testing.T) {
	entry := parseNova(t, `<import external storage from "@env/storage">
  operation set {
    input {
      key: string;
      value: unknown;
    }

    output void;
  }
/|
<template>
  <text value <- "portable" /|
/|`)

	result := Resolve(ResolutionInput{
		Project: project.Manifest{
			Project: project.Project{Name: "audiolab", Version: "0.1.0", Entry: "src/App.nova"},
		},
		Target:  "web",
		Sources: []SourceFile{{Path: "src/App.nova", File: entry}},
		TargetManifest: TargetManifest{
			ID: "web",
			ExternalCapabilities: []ExternalCapability{
				{
					Source: "@env/storage",
					Operations: []ExternalOperation{
						{
							Name: "set",
							Inputs: []Field{
								{Name: "key", Type: "string"},
							},
							Output: "number",
							Implementations: []Implementation{
								{Path: "platform/web/storage.web.js", Target: "web"},
							},
						},
					},
				},
			},
		},
	})

	assertBuildDiagnostic(t, result.Diagnostics, "external operation @env/storage.set missing input value")
	assertBuildDiagnostic(t, result.Diagnostics, "external operation @env/storage.set output type number does not match declared void")
}

func parseNova(t *testing.T, input string) parser.File {
	t.Helper()

	file, diagnostics := parser.Parse(lexer.Tokenize(input))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected parser diagnostics: %+v", diagnostics)
	}
	return file
}

func assertBuildDiagnostic(t *testing.T, diagnostics []Diagnostic, want string) {
	t.Helper()

	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, want) || diagnostic.Code == want {
			return
		}
	}
	t.Fatalf("missing diagnostic %q in %+v", want, diagnostics)
}
