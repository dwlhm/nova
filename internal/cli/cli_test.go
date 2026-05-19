package cli

import (
	"bytes"
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

[targets.android]
renderer = "@nova/android"
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

	var out bytes.Buffer
	var errOut bytes.Buffer
	if code := Run([]string{"build", "--target", "web"}, cwd, &out, &errOut); code != 0 {
		t.Fatalf("web build exit = %d\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}
	assertFileContains(t, cwd, "build/web/index.html", "Nova App")
	assertFileContains(t, cwd, "build/web/app.nova-ir.json", "\"target\": \"web\"")
	assertFileContains(t, cwd, "build/web/app.bundle.js", "@increment")
	assertFileContains(t, cwd, "build/web/bundle-manifest.json", "\"format\": \"static-web\"")

	t.Setenv("NOVA_GRADLE", fakeGradle(t, cwd))
	out.Reset()
	errOut.Reset()
	if code := Run([]string{"build", "--target", "android"}, cwd, &out, &errOut); code != 0 {
		t.Fatalf("android build exit = %d\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}
	assertFileContains(t, cwd, "build/android/build.gradle.kts", "com.android.tools.build:gradle:8.12.3")
	assertFileContains(t, cwd, "build/android/app/src/main/java/nova/generated/MainActivity.java", "setOnClickListener")
	assertFileContains(t, cwd, "build/android/generated/NovaApp.kt", "target = \"android\"")
	assertFileContains(t, cwd, "build/android/app/build/outputs/apk/debug/app-debug.apk", "apk")
	assertFileContains(t, cwd, "build/android/bundle-manifest.json", "\"format\": \"apk\"")

	writeFile(t, cwd, "build/android/app/src/main/java/nova/generated/Stale.kt", "stale")
	out.Reset()
	errOut.Reset()
	if code := Run([]string{"build", "--target", "android"}, cwd, &out, &errOut); code != 0 {
		t.Fatalf("second android build exit = %d\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}
	assertFileMissing(t, cwd, "build/android/app/src/main/java/nova/generated/Stale.kt")
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
	content := "#!/bin/sh\nmkdir -p app/build/outputs/apk/debug\nprintf apk > app/build/outputs/apk/debug/app-debug.apk\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake gradle: %v", err)
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

func assertFileMissing(t *testing.T, root string, path string) {
	t.Helper()

	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if _, err := os.Stat(fullPath); err == nil {
		t.Fatalf("%s should not exist", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}
