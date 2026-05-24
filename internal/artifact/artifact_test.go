package artifact

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/build"
	"github.com/dwlhm/nova/internal/lexer"
	"github.com/dwlhm/nova/internal/packages"
	"github.com/dwlhm/nova/internal/parser"
	"github.com/dwlhm/nova/internal/project"
)

func TestGenerateWebArtifactIncludesRuntimeViewIRAndMetadata(t *testing.T) {
	source := parseNova(t, `<import external storage from "@env/storage">
  operation load {
    input {
      key: string;
    }

    output unknown;
  }
/|
<contract state App>
  title: string <- "Nova";
/|
<template target <- web>
  <text value <- title /|
/|`)
	manifest := project.Manifest{
		Project:     project.Project{Name: "demo", Version: "0.1.0", Entry: "src/App.nova"},
		Permissions: project.PermissionMap{"storage.read": true},
	}
	targetManifest := build.WebTargetManifest()
	plan := build.Resolve(build.ResolutionInput{
		Project:        manifest,
		Target:         "web",
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("unexpected build diagnostics: %+v", plan.Diagnostics)
	}

	files, diagnostics := Generate(GenerateInput{
		Project:        manifest,
		Plan:           plan.Plan,
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
		StyleAssets: []StyleAsset{
			{SourcePath: "src/App.css", Content: ".app { color: red; }\n"},
		},
	})
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}

	assertArtifactFile(t, files, "build/web/index.html", "<script src=\"assets/nova-scheduler.js\"></script>\n  <script src=\"assets/nova-renderer.js\"></script>\n  <script src=\"assets/nova-runtime.js\"></script>")
	assertArtifactFile(t, files, "build/web/index.html", "assets/styles/src/App.css")
	assertArtifactFile(t, files, "build/web/assets/styles/src/App.css", ".app { color: red; }")
	assertArtifactFile(t, files, "build/web/assets/nova-scheduler.js", "window.NovaScheduler")
	assertArtifactFile(t, files, "build/web/assets/nova-renderer.js", "NovaRenderer")
	assertArtifactFile(t, files, "build/web/assets/nova-runtime.js", "window.NovaRuntime")
	assertArtifactFile(t, files, "build/web/app.bundle.js", "window.__NOVA_APP__")
	assertArtifactFile(t, files, "build/web/app.nova-ir.json", "\"viewIR\"")
	assertArtifactFile(t, files, "build/web/app.source-map.json", "src/App.nova")
	assertArtifactFile(t, files, "build/web/permissions.json", "storage.read")
	assertArtifactFile(t, files, "build/web/target-manifest.json", "\"id\": \"web\"")

	metadata := mustJSONFile[map[string]any](t, files, "build/web/metadata.json")
	if metadata["target"] != "web" || metadata["entryCapability"] != "src/App.nova" {
		t.Fatalf("metadata = %+v", metadata)
	}
}

func TestGenerateWebArtifactIncludesRendererExtensionAdapter(t *testing.T) {
	source := parseNova(t, `<template target <- web>
  <sparkline data <- [1, 2, 3] /|
/|`)
	manifest := project.Manifest{
		Project: project.Project{Name: "charts", Version: "0.1.0", Entry: "src/App.nova"},
		Renderer: project.RendererConfig{
			ExtensionPackages: []project.RendererPackageRef{{Name: "@acme/charts", Constraint: "*"}},
		},
	}
	targetManifest := build.WebTargetManifest()
	plan := build.Resolve(build.ResolutionInput{
		Project:        manifest,
		Target:         "web",
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
		PackageGraph: packages.ResolvedGraph{RendererExtensions: []packages.ResolvedRendererPackage{{
			Name:                 "@acme/charts",
			Version:              "1.0.0",
			TargetAdapter:        "platform/web/register.web.js",
			TargetAdapterContent: `export function register(NovaRenderer) { NovaRenderer.definePrimitive("sparkline", { mount() { return document.createElement("canvas"); } }); }`,
			Primitives: []packages.RendererPrimitive{{
				Package: "@acme/charts",
				Kind:    "sparkline",
				Props:   []packages.RendererField{{Name: "data", Type: "unknown"}},
				Targets: map[string]packages.RendererTarget{"web": {Strategy: "adapter"}},
			}},
		}}},
	})
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("unexpected build diagnostics: %+v", plan.Diagnostics)
	}

	files, diagnostics := Generate(GenerateInput{
		Project:        manifest,
		Plan:           plan.Plan,
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}

	assertArtifactFile(t, files, "build/web/index.html", "assets/renderer-extensions.js")
	assertArtifactFile(t, files, "build/web/assets/renderer-extensions.js", `NovaRenderer.definePrimitive("sparkline"`)
	assertArtifactFile(t, files, "build/web/app.nova-ir.json", `"renderer"`)
}

func TestGenerateWebArtifactAllowsUnknownKindWithWarningPolicy(t *testing.T) {
	source := parseNova(t, `<template target <- web>
  <custom_widget /|
/|`)
	manifest := project.Manifest{
		Project:  project.Project{Name: "custom", Version: "0.1.0", Entry: "src/App.nova"},
		Renderer: project.RendererConfig{UnknownKind: project.RendererUnknownKindWarn},
	}
	targetManifest := build.WebTargetManifest()
	plan := build.Resolve(build.ResolutionInput{
		Project:        manifest,
		Target:         "web",
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("unexpected build diagnostics: %+v", plan.Diagnostics)
	}

	files, diagnostics := Generate(GenerateInput{
		Project:        manifest,
		Plan:           plan.Plan,
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	if len(files) == 0 {
		t.Fatal("expected files despite warning")
	}
	assertArtifactDiagnostic(t, diagnostics, "unknown view kind custom_widget")
	if diagnostics[0].Code != "NVA-RENDER-002" {
		t.Fatalf("diagnostic = %+v, want NVA-RENDER-002", diagnostics)
	}
}

func TestGenerateWebArtifactScopesScopedStylesAndWritesStyleManifest(t *testing.T) {
	source := parseNova(t, `<template target <- web>
  <surface class <- "counter-shell">
    <text value <- "Counter" /|
  /|
/|`)
	manifest := project.Manifest{
		Project: project.Project{Name: "demo", Version: "0.1.0", Entry: "src/App.nova"},
	}
	targetManifest := build.WebTargetManifest()
	plan := build.Resolve(build.ResolutionInput{
		Project:        manifest,
		Target:         "web",
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("unexpected build diagnostics: %+v", plan.Diagnostics)
	}

	files, diagnostics := Generate(GenerateInput{
		Project:        manifest,
		Plan:           plan.Plan,
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
		StyleAssets: []StyleAsset{
			{SourcePath: "src/App.css", Content: ".counter-shell, button:hover {\n  color: red;\n}\n", Scope: StyleScopeApp},
		},
	})
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}

	assertArtifactFile(t, files, "build/web/index.html", `data-nova-style-scope="app"`)
	assertArtifactFile(t, files, "build/web/assets/styles/src/App.css", `#nova-root[data-nova-style-scope~="app"] .counter-shell`)
	assertArtifactFile(t, files, "build/web/assets/styles/src/App.css", `#nova-root[data-nova-style-scope~="app"] button:hover`)
	assertArtifactFile(t, files, "build/web/style-manifest.json", `"scope": "app"`)
	assertArtifactFile(t, files, "build/web/style-manifest.json", `"outputPath": "assets/styles/src/App.css"`)
}

func TestGenerateWebRuntimeUsesDependencyInvalidationsForGranularUpdates(t *testing.T) {
	source := parseNova(t, `<contract state Counter>
  count: number <- 0 {
    @increment -> count + 1;
  };
/|
<template target <- web>
  <surface>
    <text value <- "Count: " + count /|
    <button on_press -> @increment>
      <text value <- "+" /|
    /|
  /|
/|`)
	manifest := project.Manifest{
		Project: project.Project{Name: "demo", Version: "0.1.0", Entry: "src/App.nova"},
	}
	targetManifest := build.WebTargetManifest()
	plan := build.Resolve(build.ResolutionInput{
		Project:        manifest,
		Target:         "web",
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("unexpected build diagnostics: %+v", plan.Diagnostics)
	}

	files, diagnostics := Generate(GenerateInput{
		Project:        manifest,
		Plan:           plan.Plan,
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}

	assertArtifactFile(t, files, "build/web/assets/nova-scheduler.js", "function enqueue(scheduler, host, source, event, args, options)")
	assertArtifactFile(t, files, "build/web/assets/nova-scheduler.js", "scheduler.queue.push(envelope)")
	assertArtifactFile(t, files, "build/web/assets/nova-scheduler.js", "function drain(scheduler, host)")
	assertArtifactFile(t, files, "build/web/assets/nova-scheduler.js", "function planStateCommit(host, envelope, beforeState)")
	assertArtifactFile(t, files, "build/web/assets/nova-scheduler.js", "host.commitState(commit.state)")
	assertArtifactFile(t, files, "build/web/assets/nova-scheduler.js", "host.update(commit.invalidations)")
	assertArtifactFile(t, files, "build/web/assets/nova-runtime.js", "runtime.refs = new Map()")
	assertArtifactFile(t, files, "build/web/assets/nova-runtime.js", "window.NovaScheduler.create(schedulerHost(runtime))")
	assertArtifactFile(t, files, "build/web/assets/nova-runtime.js", "function updateBindings(runtime, invalidations)")
	assertArtifactFile(t, files, "build/web/assets/nova-runtime.js", "function shouldApplyBinding(states, invalidations)")
	assertArtifactFile(t, files, "build/web/assets/nova-runtime.js", "root.replaceChildren(...renderNodes")
	assertArtifactFileNotContains(t, files, "build/web/assets/nova-runtime.js", "render(runtime);\n    reconcileNavigation")
}

func TestGenerateAndroidArtifactIncludesGradleAndGeneratedBindings(t *testing.T) {
	source := parseNova(t, `<contract state Counter>
  count: number <- 0 {
    @increment -> count + 1;
    @decrement -> count - 1;
    @reset -> 0;
  };
/|
<template target <- android>
  <surface class <- "counter-shell">
    <text class <- "counter-value" value <- "Count: " + count /|
    <button on_press -> @increment>
      <text value <- "+" /|
    /|
  /|
/|`)
	manifest := project.Manifest{
		Project: project.Project{Name: "demo", Version: "0.1.0", Entry: "src/App.nova"},
		Targets: map[string]project.Target{"android": testAndroidTarget("dev.example.demo")},
	}
	targetManifest := build.AndroidTargetManifest()
	plan := build.Resolve(build.ResolutionInput{
		Project:        manifest,
		Target:         "android",
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("unexpected build diagnostics: %+v", plan.Diagnostics)
	}

	files, diagnostics := Generate(GenerateInput{
		Project:        manifest,
		Plan:           plan.Plan,
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
		StyleAssets: []StyleAsset{{
			SourcePath: "src/App.css",
			Content:    ".counter-shell { padding: 12px; background: #fbfcfe; border: 1px solid #dfe5ef; border-radius: 8px; }\n.counter-value { color: #151923; font-size: 34px; font-weight: 800; text-align: center; }",
			Scope:      StyleScopeGlobal,
		}},
	})
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}

	assertArtifactFile(t, files, "build/android/settings.gradle.kts", "include(\":app\")")
	assertArtifactFileNotContains(t, files, "build/android/gradle.properties", "android.useAndroidX=true")
	assertArtifactFile(t, files, "build/android/settings.gradle.kts", "id(\"com.android.application\") version \"8.12.3\"")
	assertArtifactFile(t, files, "build/android/app/build.gradle.kts", "plugins {\n    id(\"com.android.application\")")
	assertArtifactFile(t, files, "build/android/app/build.gradle.kts", "applicationId = \"dev.example.demo\"")
	assertArtifactFile(t, files, "build/android/app/build.gradle.kts", "compileSdk = 35")
	assertArtifactFile(t, files, "build/android/app/build.gradle.kts", "JavaVersion.VERSION_17")
	assertArtifactFile(t, files, "build/android/app/build.gradle.kts", "implementation(project(\":nova-scheduler\"))")
	assertArtifactFile(t, files, "build/android/settings.gradle.kts", "include(\":nova-scheduler\")")
	assertArtifactFile(t, files, "build/android/app/src/main/AndroidManifest.xml", "android:label=\"demo\"")
	assertArtifactFile(t, files, "build/android/app/src/main/AndroidManifest.xml", "nova.generated.MainActivity")
	assertArtifactFile(t, files, "build/android/app/src/main/res/values/styles.xml", "Theme.Nova")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "public final class MainActivity extends Activity implements NovaScheduler.Host")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "dispatch(\"@increment\"")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "private final NovaScheduler scheduler = new NovaScheduler(this)")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "public Map<String, Object> schedulerState()")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "node_0.setPadding(dp(12), dp(12), dp(12), dp(12));")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "GradientDrawable node_0Style = new GradientDrawable();")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "node_0_0.setTextSize(TypedValue.COMPLEX_UNIT_SP, 34);")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "node_0_0.setTypeface(Typeface.DEFAULT, Typeface.BOLD);")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/NovaRuntime.java", "public final class NovaRuntime")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/NovaRuntime.java", "public static Object evaluate(")
	assertArtifactFile(t, files, "build/android/nova-scheduler/src/main/java/nova/scheduler/NovaScheduler.java", "public final class NovaScheduler")
	assertArtifactFile(t, files, "build/android/nova-scheduler/src/main/java/nova/scheduler/NovaEventEnvelope.java", "final class NovaEventEnvelope")
	assertArtifactFile(t, files, "build/android/nova-scheduler/src/main/java/nova/scheduler/NovaTransition.java", "public final class NovaTransition")
	assertArtifactFile(t, files, "build/android/generated/NovaApp.java", "public final class NovaApp")
	assertArtifactFile(t, files, "build/android/generated/NovaExternalBindings.java", "NovaExternalBindings")
	assertArtifactFile(t, files, "build/android/nova-ir/app.nova-ir.json", "\"viewIR\"")
	assertArtifactFile(t, files, "build/android/nova-ir/permissions.json", "\"permissions\": []")
	assertArtifactFile(t, files, "build/android/nova-ir/target-manifest.json", "\"id\": \"android\"")
	assertArtifactFile(t, files, "build/android/nova-ir/target-manifest.json", "platform/android/storage.android.java")
}

func TestGenerateAndroidArtifactRequiresUserTargetConfig(t *testing.T) {
	source := parseNova(t, `<template target <- android>
  <text value <- "Hello" /|
/|`)
	manifest := project.Manifest{
		Project: project.Project{Name: "demo", Version: "0.1.0", Entry: "src/App.nova"},
	}
	targetManifest := build.AndroidTargetManifest()
	plan := build.Resolve(build.ResolutionInput{
		Project:        manifest,
		Target:         "android",
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("unexpected build diagnostics: %+v", plan.Diagnostics)
	}

	_, diagnostics := Generate(GenerateInput{
		Project:        manifest,
		Plan:           plan.Plan,
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	assertArtifactDiagnostic(t, diagnostics, "targets.android.application_id is required")
	assertArtifactDiagnostic(t, diagnostics, "targets.android.compile_sdk is required")
}

func TestGenerateAndroidArtifactRejectsUnsupportedRenderer(t *testing.T) {
	source := parseNova(t, `<template target <- android>
  <text value <- "Hello" /|
/|`)
	manifest := project.Manifest{
		Project: project.Project{Name: "demo", Version: "0.1.0", Entry: "src/App.nova"},
		Targets: map[string]project.Target{
			"android": {
				Renderer: "@nova/android-legacy",
				Options:  testAndroidTarget("dev.example.demo").Options,
			},
		},
	}
	targetManifest := build.AndroidTargetManifest()
	plan := build.Resolve(build.ResolutionInput{
		Project:        manifest,
		Target:         "android",
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("unexpected build diagnostics: %+v", plan.Diagnostics)
	}

	_, diagnostics := Generate(GenerateInput{
		Project:        manifest,
		Plan:           plan.Plan,
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	assertArtifactDiagnostic(t, diagnostics, "targets.android.renderer must be @nova/android")
}

func TestGenerateWebArtifactSupportsMultiPageRouteProjection(t *testing.T) {
	source := parseNova(t, `<contract type Route>
  path: string;
/|
<contract state Router>
  route: Route <- { path <- "/"; } {
    @route_changed(next: Route) -> next;
  };
/|
<template target <- web>
  <surface>
    <button on_press -> @route_changed({ path <- "/settings"; })>
      <text value <- "Settings" /|
    /|
    <page path <- "/">
      <text value <- "Home" /|
    /|
    <page path <- "/settings">
      <text value <- route.path /|
    /|
  /|
/|`)
	manifest := project.Manifest{
		Project: project.Project{Name: "demo", Version: "0.1.0", Entry: "src/App.nova"},
		Targets: map[string]project.Target{"android": testAndroidTarget("dev.example.multipage")},
	}
	targetManifest := build.WebTargetManifest()
	plan := build.Resolve(build.ResolutionInput{
		Project:        manifest,
		Target:         "web",
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("unexpected build diagnostics: %+v", plan.Diagnostics)
	}

	files, diagnostics := Generate(GenerateInput{
		Project:        manifest,
		Plan:           plan.Plan,
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}

	assertArtifactFile(t, files, "build/web/assets/nova-runtime.js", "renderPage")
	assertArtifactFile(t, files, "build/web/assets/nova-runtime.js", "recordExpression")
	assertArtifactFile(t, files, "build/web/assets/nova-runtime.js", "window.addEventListener(\"popstate\"")
	assertArtifactFile(t, files, "build/web/assets/nova-runtime.js", "window.history[method]")
	assertArtifactFile(t, files, "build/web/assets/nova-runtime.js", "ROUTE_CHANGED_EVENT")
	assertArtifactFile(t, files, "build/web/app.nova-ir.json", "\"Pages\"")
	assertArtifactFile(t, files, "build/web/app.bundle.js", `"initial": "({ path: \"/\" })"`)
}

func TestGenerateWebArtifactSupportsProductionRouteMatching(t *testing.T) {
	source := parseNova(t, `<contract type Route>
  path: string;
  params: unknown;
  query: unknown;
  fragment: string;
/|
<contract state Router>
  route: Route <- { path <- "/"; } {
    @route_changed(next: Route) -> next;
  };
/|
<template target <- web>
  <surface>
    <page path <- "/users/settings">
      <text value <- "Static settings" /|
    /|
    <page path <- "/users/:id">
      <text value <- "User detail" /|
    /|
    <page path <- "/docs/*">
      <text value <- "Docs" /|
    /|
    <page path <- "*">
      <text value <- "Not found" /|
    /|
  /|
/|`)
	manifest := project.Manifest{
		Project: project.Project{Name: "demo", Version: "0.1.0", Entry: "src/App.nova"},
		Targets: map[string]project.Target{"android": testAndroidTarget("dev.example.routing")},
	}
	targetManifest := build.WebTargetManifest()
	plan := build.Resolve(build.ResolutionInput{
		Project:        manifest,
		Target:         "web",
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("unexpected build diagnostics: %+v", plan.Diagnostics)
	}

	files, diagnostics := Generate(GenerateInput{
		Project:        manifest,
		Plan:           plan.Plan,
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}

	assertArtifactFile(t, files, "build/web/app.nova-ir.json", `"pattern": "/users/:id"`)
	assertArtifactFile(t, files, "build/web/app.nova-ir.json", `"id"`)
	assertArtifactFile(t, files, "build/web/app.nova-ir.json", `"fallback": true`)
	assertArtifactFile(t, files, "build/web/assets/nova-runtime.js", "function selectedPageNodes")
	assertArtifactFile(t, files, "build/web/assets/nova-runtime.js", "function routeMatch(pattern, path)")
	assertArtifactFile(t, files, "build/web/assets/nova-runtime.js", "function routeValueForShape(route, shape, runtime)")
	assertArtifactFile(t, files, "build/web/assets/nova-runtime.js", "next.params = match.params")
	assertArtifactFile(t, files, "build/web/assets/nova-runtime.js", "routeMatch(pattern, activeRoutePath(runtime))")
	assertArtifactFile(t, files, "build/web/assets/nova-runtime.js", "const parsed = new URL(text)")
	assertArtifactFile(t, files, "build/web/assets/nova-runtime.js", "new URLSearchParams(search || \"\")")
	assertArtifactFile(t, files, "build/web/assets/nova-runtime.js", "function safeDecodeURIComponent(value)")
}

func TestGenerateAndroidArtifactSupportsMultiPageRouteProjection(t *testing.T) {
	source := parseNova(t, `<contract type Route>
  path: string;
/|
<contract state Router>
  route: Route <- { path <- "/"; } {
    @route_changed(next: Route) -> next;
  };
/|
<template>
  <surface>
    <button on_press -> @route_changed({ path <- "/settings"; })>
      <text value <- "Settings" /|
    /|
    <page path <- "/">
      <text value <- "Home" /|
    /|
    <page path <- "/settings">
      <text value <- "Page " + route.path /|
    /|
  /|
/|`)
	manifest := project.Manifest{
		Project: project.Project{Name: "demo", Version: "0.1.0", Entry: "src/App.nova"},
		Targets: map[string]project.Target{"android": testAndroidTarget("dev.example.multipage")},
	}
	targetManifest := build.AndroidTargetManifest()
	plan := build.Resolve(build.ResolutionInput{
		Project:        manifest,
		Target:         "android",
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("unexpected build diagnostics: %+v", plan.Diagnostics)
	}

	files, diagnostics := Generate(GenerateInput{
		Project:        manifest,
		Plan:           plan.Plan,
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}

	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "state.put(\"route\", record(entry(\"path\", \"/\")))")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "dispatch(\"@route_changed\", Arrays.<Object>asList(record(entry(\"path\", \"/settings\"))))")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "public void onBackPressed()")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "private final List<Object> routeBackStack")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "dispatch(\"@navigate\", Collections.singletonList(record(entry(\"kind\", \"back\"))))")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "hasTransition(\"@navigate\")")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "handleSystemBack()")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "activeRoutePath()")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "applyBindings(invalidations)")
	assertArtifactFile(t, files, "build/android/generated/NovaRoutes.java", "SETTINGS = \"/settings\"")
}

func TestGenerateAndroidArtifactSupportsProductionRouteMatching(t *testing.T) {
	source := parseNova(t, `<contract type Route>
  path: string;
  params: unknown;
/|
<contract state Router>
  route: Route <- { path <- "/"; } {
    @route_changed(next: Route) -> next;
  };
/|
<template target <- android>
  <surface>
    <page path <- "/users/settings">
      <text value <- "Static settings" /|
    /|
    <page path <- "/users/:id">
      <text value <- "User detail" /|
    /|
    <page path <- "/docs/*">
      <text value <- "Docs" /|
    /|
    <page path <- "*">
      <text value <- "Not found" /|
    /|
  /|
/|`)
	manifest := project.Manifest{
		Project: project.Project{Name: "demo", Version: "0.1.0", Entry: "src/App.nova"},
		Targets: map[string]project.Target{"android": testAndroidTarget("dev.example.routing")},
	}
	targetManifest := build.AndroidTargetManifest()
	plan := build.Resolve(build.ResolutionInput{
		Project:        manifest,
		Target:         "android",
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("unexpected build diagnostics: %+v", plan.Diagnostics)
	}

	files, diagnostics := Generate(GenerateInput{
		Project:        manifest,
		Plan:           plan.Plan,
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}

	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "int selectedRouteScore = bestRouteScore(routePatterns(), activePath)")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "routeMatchScore(textValue(")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "private List<String> routePatterns()")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "state.put(\"route\", routeValueForShape(state.get(\"route\"), routePatterns()))")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/NovaRuntime.java", "public static int bestRouteScore(List<String> patterns, String value)")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/NovaRuntime.java", "private static RouteMatch routeMatch(String pattern, String value)")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/NovaRuntime.java", "next.put(\"params\", best.params)")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/NovaRuntime.java", "URI.create(text)")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/NovaRuntime.java", "safeDecodePathSegment")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/NovaRuntime.java", "encodeComponent")
	assertArtifactFile(t, files, "build/android/generated/NovaRoutes.java", "USERS_ID = \"/users/:id\"")
	assertArtifactFile(t, files, "build/android/generated/NovaRoutes.java", "FALLBACK = \"*\"")
}

func TestGenerateAndroidArtifactWrapsRowsForDenseNavigation(t *testing.T) {
	source := parseNova(t, `<contract state Router>
  route: string <- "/" {
    @route_changed(next: string) -> next;
  };
/|
<template target <- android>
  <surface>
    <row>
      <button on_press -> @route_changed("/")>
        <text value <- "Home" /|
      /|
      <button on_press -> @route_changed("/users/settings")>
        <text value <- "User Settings" /|
      /|
      <button on_press -> @route_changed("/users/ada")>
        <text value <- "Ada" /|
      /|
      <button on_press -> @route_changed("/docs/reference/routing")>
        <text value <- "Docs" /|
      /|
      <button on_press -> @route_changed("/missing/route")>
        <text value <- "Missing" /|
      /|
    /|
  /|
/|`)
	manifest := project.Manifest{
		Project: project.Project{Name: "demo", Version: "0.1.0", Entry: "src/App.nova"},
		Targets: map[string]project.Target{"android": testAndroidTarget("dev.example.routing")},
	}
	targetManifest := build.AndroidTargetManifest()
	plan := build.Resolve(build.ResolutionInput{
		Project:        manifest,
		Target:         "android",
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("unexpected build diagnostics: %+v", plan.Diagnostics)
	}

	files, diagnostics := Generate(GenerateInput{
		Project:        manifest,
		Plan:           plan.Plan,
		Sources:        []build.SourceFile{{Path: "src/App.nova", File: source}},
		TargetManifest: targetManifest,
	})
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}

	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "GridLayout")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "setColumnCount(3)")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "new Button(this)")
	assertArtifactFile(t, files, "build/android/app/src/main/java/nova/generated/MainActivity.java", "TextView")
}

func TestExpressionToJSCompilesRecordLiterals(t *testing.T) {
	tokens := lexer.Tokenize(`{ path <- "/settings"; title <- currentTitle; }`)
	expression := expressionToJS(tokens[:len(tokens)-1], map[string]bool{"currentTitle": true}, nil)

	if expression != `({ path: "/settings", title: state.currentTitle })` {
		t.Fatalf("expression = %s", expression)
	}
}

func parseNova(t *testing.T, input string) parser.File {
	t.Helper()

	file, diagnostics := parser.Parse(lexer.Tokenize(input))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected parser diagnostics: %+v", diagnostics)
	}
	return file
}

func testAndroidTarget(applicationID string) project.Target {
	return project.Target{
		Renderer: "@nova/android",
		Options: map[string]string{
			"application_id": applicationID,
			"namespace":      "nova.generated",
			"compile_sdk":    "35",
			"min_sdk":        "23",
			"target_sdk":     "35",
			"version_code":   "1",
			"version_name":   "0.1.0",
			"gradle_plugin":  "8.12.3",
			"theme":          "Theme.Nova",
			"theme_parent":   "android:style/Theme.DeviceDefault.Light.NoActionBar",
			"java_version":   "17",
			"label":          "demo",
		},
	}
}

func assertArtifactFile(t *testing.T, files []File, path string, want string) {
	t.Helper()

	for _, file := range files {
		if file.Path != path {
			continue
		}
		if !strings.Contains(file.Content, want) {
			t.Fatalf("%s = %s, want content containing %q", path, file.Content, want)
		}
		return
	}
	t.Fatalf("missing artifact file %s in %+v", path, files)
}

func assertArtifactFileNotContains(t *testing.T, files []File, path string, unwanted string) {
	t.Helper()

	for _, file := range files {
		if file.Path != path {
			continue
		}
		if strings.Contains(file.Content, unwanted) {
			t.Fatalf("%s = %s, want content not containing %q", path, file.Content, unwanted)
		}
		return
	}
	t.Fatalf("missing artifact file %s in %+v", path, files)
}

func assertArtifactFileMissing(t *testing.T, files []File, path string) {
	t.Helper()

	for _, file := range files {
		if file.Path == path {
			t.Fatalf("artifact file %s should be missing", path)
		}
	}
}

func assertArtifactDiagnostic(t *testing.T, diagnostics []Diagnostic, want string) {
	t.Helper()

	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, want) {
			return
		}
	}
	t.Fatalf("missing diagnostic %q in %+v", want, diagnostics)
}

func mustJSONFile[T any](t *testing.T, files []File, path string) T {
	t.Helper()

	for _, file := range files {
		if file.Path != path {
			continue
		}
		var out T
		if err := json.Unmarshal([]byte(file.Content), &out); err != nil {
			t.Fatalf("decode %s: %v\n%s", path, err, file.Content)
		}
		return out
	}
	t.Fatalf("missing artifact file %s", path)
	var zero T
	return zero
}
