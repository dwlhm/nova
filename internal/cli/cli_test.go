package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBuildWritesWebAndAndroidArtifacts(t *testing.T) {
	cwd := t.TempDir()
	writeFile(t, cwd, "nova.toml", `[project]
name = "demo"
version = "0.1.0"
entry = "src/App.nova"

[targets.web]
renderer = "@nova/web"
styles = ["src/App.css"]

[targets.android]
renderer = "@nova/android"
styles = ["src/App.css"]
application_id = "dev.example.demo"
namespace = "nova.generated"
compile_sdk = 35
min_sdk = 23
target_sdk = 35
version_code = 1
version_name = "0.1.0"
gradle_plugin = "8.12.3"
theme = "Theme.Nova"
theme_parent = "android:style/Theme.DeviceDefault.Light.NoActionBar"
java_version = "17"
label = "demo"
	`)
	writeFile(t, cwd, "src/App.nova", `<contract state Counter>
  count: number <- 0 {
    @increment -> count + 1;
    @decrement -> count - 1;
    @reset -> 0;
  };
/|
<template target <- web>
  <surface class <- "counter-shell">
    <text value <- "Count: " + count /|
    <button on_press -> @increment>
      <text value <- "+" /|
    /|
  /|
/|
<template target <- android>
  <surface class <- "counter-shell">
    <text value <- "Count: " + count /|
    <button on_press -> @increment>
      <text value <- "+" /|
    /|
	/|
	/|`)
	writeFile(t, cwd, "src/App.css", `.counter-shell {
  display: grid;
  padding: 12px;
}
`)

	var out bytes.Buffer
	var errOut bytes.Buffer
	if code := Run([]string{"build", "--target", "web"}, cwd, &out, &errOut); code != 0 {
		t.Fatalf("web build exit = %d\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}
	assertFileContains(t, cwd, "build/web/index.html", "demo")
	assertFileContains(t, cwd, "build/web/index.html", "assets/styles/src/App.css")
	assertFileContains(t, cwd, "build/web/assets/styles/src/App.css", ".counter-shell")
	assertFileContains(t, cwd, "build/web/app.contract.json", "\"target\": \"web\"")
	assertFileContains(t, cwd, "build/web/app.bundle.js", "@increment")
	assertFileContains(t, cwd, "build/web/bundle-manifest.json", "\"format\": \"static-web\"")

	t.Setenv("NOVA_GRADLE", fakeGradle(t, cwd))
	out.Reset()
	errOut.Reset()
	if code := Run([]string{"build", "--target", "android"}, cwd, &out, &errOut); code != 0 {
		t.Fatalf("android build exit = %d\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}
	assertFileContains(t, cwd, "build/android/settings.gradle.kts", "id(\"com.android.application\") version \"8.12.3\"")
	assertFileContains(t, cwd, "build/android/settings.gradle.kts", "include(\":nova-scheduler\")")
	assertFileNotContains(t, cwd, "build/android/gradle.properties", "android.useAndroidX=true")
	assertFileContains(t, cwd, "build/android/app/build.gradle.kts", "implementation(project(\":nova-scheduler\"))")
	assertFileContains(t, cwd, "build/android/nova-scheduler/src/main/java/nova/scheduler/NovaScheduler.java", "public final class NovaScheduler")
	assertFileContains(t, cwd, "build/android/app/src/main/java/nova/generated/MainActivity.java", "dispatch(\"@increment\"")
	assertFileContains(t, cwd, "build/android/app/src/main/java/nova/generated/MainActivity.java", "node_0.setPadding(dp(12), dp(12), dp(12), dp(12));")
	assertFileContains(t, cwd, "build/android/generated/NovaApp.java", "target = \"android\"")
	assertFileContains(t, cwd, "build/android/app/build/outputs/apk/debug/app-debug.apk", "apk")
	assertFileContains(t, cwd, "build/android/bundle-manifest.json", "\"format\": \"apk\"")
	assertFileContains(t, cwd, "gradle.log", "--offline assembleDebug")

	writeFile(t, cwd, "build/android/app/src/main/java/nova/generated/Stale.java", "stale")
	out.Reset()
	errOut.Reset()
	if code := Run([]string{"build", "--target", "android"}, cwd, &out, &errOut); code != 0 {
		t.Fatalf("second android build exit = %d\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}
	assertFileMissing(t, cwd, "build/android/app/src/main/java/nova/generated/Stale.java")
}

func TestRunDevOnceBuildsWebArtifact(t *testing.T) {
	cwd := t.TempDir()
	writeFile(t, cwd, "nova.toml", `[project]
name = "demo"
version = "0.1.0"
entry = "src/App.nova"

[targets.web]
renderer = "@nova/web"
`)
	writeFile(t, cwd, "src/App.nova", `<template target <- web>
  <text value <- "web" /|
/|`)

	var out bytes.Buffer
	var errOut bytes.Buffer
	if code := Run([]string{"dev", "--target", "web", "--once"}, cwd, &out, &errOut); code != 0 {
		t.Fatalf("web dev exit = %d\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}
	assertFileContains(t, cwd, "build/web/bundle-manifest.json", "\"format\": \"static-web\"")
	if !strings.Contains(out.String(), "dev strategy: browser_full_reload") {
		t.Fatalf("stdout = %s, want dev strategy", out.String())
	}
}

func TestRunDevOnceInstallsAndLaunchesAndroid(t *testing.T) {
	cwd := t.TempDir()
	writeFile(t, cwd, "nova.toml", `[project]
name = "demo"
version = "0.1.0"
entry = "src/App.nova"

[targets.android]
renderer = "@nova/android"
application_id = "dev.example.demo"
namespace = "nova.generated"
compile_sdk = 35
min_sdk = 23
target_sdk = 35
version_code = 1
version_name = "0.1.0"
gradle_plugin = "8.12.3"
theme = "Theme.Nova"
theme_parent = "android:style/Theme.DeviceDefault.Light.NoActionBar"
java_version = "17"
label = "demo"
`)
	writeFile(t, cwd, "src/App.nova", `<template target <- android>
  <text value <- "android" /|
/|`)

	adb := fakeADB(t, cwd)
	t.Setenv("NOVA_GRADLE", fakeGradle(t, cwd))

	var out bytes.Buffer
	var errOut bytes.Buffer
	if code := Run([]string{"dev", "--target", "android", "--once", "--adb", adb}, cwd, &out, &errOut); code != 0 {
		t.Fatalf("android dev exit = %d\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}
	assertFileContains(t, cwd, "build/android/app/build/outputs/apk/debug/app-debug.apk", "apk")
	assertFileContains(t, cwd, "gradle.log", "assembleDebug")
	assertFileNotContains(t, cwd, "gradle.log", "--offline")
	assertFileContains(t, cwd, "adb.log", "install --user 0 -r")
	assertFileContains(t, cwd, "adb.log", "shell am start --user 0 -a android.intent.action.MAIN -c android.intent.category.LAUNCHER -n dev.example.demo/nova.generated.MainActivity")
	if !strings.Contains(out.String(), "dev strategy: android_install_sync") {
		t.Fatalf("stdout = %s, want android install sync strategy", out.String())
	}
}

func TestRunDevOnceCanTargetCustomAndroidUser(t *testing.T) {
	cwd := t.TempDir()
	writeFile(t, cwd, "nova.toml", `[project]
name = "demo"
version = "0.1.0"
entry = "src/App.nova"

[targets.android]
renderer = "@nova/android"
application_id = "dev.example.demo"
namespace = "nova.generated"
compile_sdk = 35
min_sdk = 23
target_sdk = 35
version_code = 1
version_name = "0.1.0"
gradle_plugin = "8.12.3"
theme = "Theme.Nova"
theme_parent = "android:style/Theme.DeviceDefault.Light.NoActionBar"
java_version = "17"
label = "demo"
`)
	writeFile(t, cwd, "src/App.nova", `<template target <- android>
  <text value <- "android" /|
/|`)

	adb := fakeADB(t, cwd)
	t.Setenv("NOVA_GRADLE", fakeGradle(t, cwd))

	var out bytes.Buffer
	var errOut bytes.Buffer
	if code := Run([]string{"dev", "--target", "android", "--once", "--adb", adb, "--android-user", "10"}, cwd, &out, &errOut); code != 0 {
		t.Fatalf("android dev exit = %d\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}
	assertFileContains(t, cwd, "adb.log", "install --user 10 -r")
	assertFileContains(t, cwd, "adb.log", "shell am start --user 10 -a android.intent.action.MAIN -c android.intent.category.LAUNCHER -n dev.example.demo/nova.generated.MainActivity")
}

func TestCommandOutputErrorDetectsAndroidActivityManagerFailure(t *testing.T) {
	message, ok := commandOutputError("Starting: Intent {}\nError: Activity not started, unable to resolve Intent {}\n")
	if !ok {
		t.Fatalf("expected Android Activity Manager error to be detected")
	}
	if !strings.Contains(message, "unable to resolve Intent") {
		t.Fatalf("message = %q", message)
	}
}

func TestRunBuildReportsStyleAssetDiagnostics(t *testing.T) {
	cases := []struct {
		name   string
		styles string
		want   string
	}{
		{name: "unsafe path", styles: `styles = ["../theme.css"]`, want: "NVA-STYLE-001"},
		{name: "missing file", styles: `styles = ["src/Missing.css"]`, want: "NVA-STYLE-002"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cwd := t.TempDir()
			writeFile(t, cwd, "nova.toml", fmt.Sprintf(`[project]
name = "demo"
version = "0.1.0"
entry = "src/App.nova"

[targets.web]
renderer = "@nova/web"
%s
`, tt.styles))
			writeFile(t, cwd, "src/App.nova", `<template target <- web>
  <text value <- "web" /|
/|`)

			var out bytes.Buffer
			var errOut bytes.Buffer
			if code := Run([]string{"build", "--target", "web", "--bundle=false"}, cwd, &out, &errOut); code == 0 {
				t.Fatalf("web build should fail, stdout=%s stderr=%s", out.String(), errOut.String())
			}
			if !strings.Contains(errOut.String(), tt.want) {
				t.Fatalf("stderr = %s, want %s", errOut.String(), tt.want)
			}
		})
	}
}

func TestRunBuildReportsDiagnosticsForMissingTargetTemplate(t *testing.T) {
	cwd := t.TempDir()
	writeFile(t, cwd, "nova.toml", `[project]
name = "demo"
version = "0.1.0"
entry = "src/App.nova"
`)
	writeFile(t, cwd, "src/App.nova", `<template target <- web>
  <text value <- "web" /|
/|`)

	var out bytes.Buffer
	var errOut bytes.Buffer
	if code := Run([]string{"build", "--target", "android"}, cwd, &out, &errOut); code == 0 {
		t.Fatalf("android build should fail without template, stdout=%s stderr=%s", out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "NVA-TEMPLATE-001") {
		t.Fatalf("stderr should contain build diagnostic, got %s", errOut.String())
	}
}

func TestRunInitCreatesReadableStarterProject(t *testing.T) {
	cwd := t.TempDir()

	var out bytes.Buffer
	var errOut bytes.Buffer
	if code := Run([]string{"init", "--name", "hello"}, cwd, &out, &errOut); code != 0 {
		t.Fatalf("init exit = %d\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}

	assertFileContains(t, cwd, "nova.toml", `name = "hello"`)
	assertFileContains(t, cwd, "nova.toml", `scoped_styles = ["src/App.css"]`)
	assertFileContains(t, cwd, "src/App.nova", `<contract state Counter>`)
	assertFileContains(t, cwd, "src/App.css", `.counter-shell`)
	if !strings.Contains(out.String(), "initialized Nova project hello") {
		t.Fatalf("stdout = %s, want init summary", out.String())
	}
}

func TestRunInitRefusesToOverwriteProjectWithoutForce(t *testing.T) {
	cwd := t.TempDir()
	writeFile(t, cwd, "nova.toml", `[project]
name = "existing"
`)

	var out bytes.Buffer
	var errOut bytes.Buffer
	if code := Run([]string{"init", "--name", "next"}, cwd, &out, &errOut); code == 0 {
		t.Fatalf("init should fail without --force, stdout=%s stderr=%s", out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "NVA-INIT-001") {
		t.Fatalf("stderr = %s, want overwrite diagnostic", errOut.String())
	}
}

func TestRunCheckValidatesProjectWithoutWritingArtifacts(t *testing.T) {
	cwd := t.TempDir()
	writeFile(t, cwd, "nova.toml", `[project]
name = "demo"
version = "0.1.0"
entry = "src/App.nova"

[targets.web]
renderer = "@nova/web"
`)
	writeFile(t, cwd, "src/App.nova", `<template target <- web>
  <text value <- "web" /|
/|`)

	var out bytes.Buffer
	var errOut bytes.Buffer
	if code := Run([]string{"check", "--target", "web"}, cwd, &out, &errOut); code != 0 {
		t.Fatalf("check exit = %d\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "checked web project demo") {
		t.Fatalf("stdout = %s, want check summary", out.String())
	}
	assertFileMissing(t, cwd, "build/web/index.html")
}

func TestRunInspectPrintsBuildPlanJSON(t *testing.T) {
	cwd := t.TempDir()
	writeFile(t, cwd, "nova.toml", `[project]
name = "demo"
version = "0.1.0"
entry = "src/App.nova"

[targets.web]
renderer = "@nova/web"
`)
	writeFile(t, cwd, "src/App.nova", `<template target <- web>
  <text value <- "web" /|
/|`)

	var out bytes.Buffer
	var errOut bytes.Buffer
	if code := Run([]string{"inspect", "--target", "web"}, cwd, &out, &errOut); code != 0 {
		t.Fatalf("inspect exit = %d\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), `"target": "web"`) || !strings.Contains(out.String(), `"entry": "src/App.nova"`) {
		t.Fatalf("inspect output = %s, want target and entry JSON", out.String())
	}
}

func TestRunFmtCheckReportsUnformattedSources(t *testing.T) {
	cwd := t.TempDir()
	writeFile(t, cwd, "src/App.nova", `<template target <- web>
<surface>
<text value <- "web" /|
/|
/|`)

	var out bytes.Buffer
	var errOut bytes.Buffer
	if code := Run([]string{"fmt", "--check", "src/App.nova"}, cwd, &out, &errOut); code == 0 {
		t.Fatalf("fmt --check should fail for unformatted source, stdout=%s stderr=%s", out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "NVA-FMT-001") {
		t.Fatalf("stderr = %s, want fmt diagnostic", errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if code := Run([]string{"fmt", "src/App.nova"}, cwd, &out, &errOut); code != 0 {
		t.Fatalf("fmt exit = %d\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}
	assertFileContains(t, cwd, "src/App.nova", "  <surface>")
	assertFileContains(t, cwd, "src/App.nova", "    <text value <- \"web\" /|")
}

func TestRunTestExecutesConformanceFixtures(t *testing.T) {
	cwd := t.TempDir()
	writeFile(t, cwd, "tests/conformance/web/counter/nova.toml", `[project]
name = "counter"
version = "0.1.0"
entry = "src/App.nova"

[targets.web]
renderer = "@nova/web"
`)
	writeFile(t, cwd, "tests/conformance/web/counter/src/App.nova", `<contract state Counter>
  count: number <- 0 {
    @increment -> count + 1;
  };
/|
<template target <- web>
  <button on_press -> @increment>
    <text value <- "Count " + count /|
  /|
/|`)
	writeFile(t, cwd, "tests/conformance/web/counter/nova.conformance.json", `{
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
      "pages": 0
    }
  }
}
`)

	var out bytes.Buffer
	var errOut bytes.Buffer
	if code := Run([]string{"test"}, cwd, &out, &errOut); code != 0 {
		t.Fatalf("test exit = %d\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "1 conformance fixture passed") {
		t.Fatalf("stdout = %s, want conformance summary", out.String())
	}
}

func writeFile(t *testing.T, root string, path string, content string) {
	t.Helper()

	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(fullPath), err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", fullPath, err)
	}
}

func fakeGradle(t *testing.T, root string) string {
	t.Helper()

	path := filepath.Join(root, "fake-gradle")
	logPath := filepath.Join(root, "gradle.log")
	content := "#!/bin/sh\nprintf '%s ' \"$@\" >> " + logPath + "\nprintf '\\n' >> " + logPath + "\nmkdir -p app/build/outputs/apk/debug\nprintf apk > app/build/outputs/apk/debug/app-debug.apk\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake gradle: %v", err)
	}
	return path
}

func fakeADB(t *testing.T, root string) string {
	t.Helper()

	path := filepath.Join(root, "fake-adb")
	logPath := filepath.Join(root, "adb.log")
	content := "#!/bin/sh\nprintf '%s ' \"$@\" >> " + logPath + "\nprintf '\\n' >> " + logPath + "\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake adb: %v", err)
	}
	return path
}

func assertFileContains(t *testing.T, root string, path string, want string) {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(content), want) {
		t.Fatalf("%s = %s, want %q", path, string(content), want)
	}
}

func assertFileNotContains(t *testing.T, root string, path string, unwanted string) {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if strings.Contains(string(content), unwanted) {
		t.Fatalf("%s = %s, did not want %q", path, string(content), unwanted)
	}
}

func assertFileMissing(t *testing.T, root string, path string) {
	t.Helper()

	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if _, err := os.Stat(fullPath); err == nil {
		t.Fatalf("%s should not exist", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}
