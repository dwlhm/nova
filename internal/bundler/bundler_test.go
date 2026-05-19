package bundler

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBundleWebWritesStaticManifest(t *testing.T) {
	root := t.TempDir()
	writeBundleFile(t, root, "build/web/index.html", "<html></html>")
	writeBundleFile(t, root, "build/web/assets/nova-runtime.js", "runtime")
	writeBundleFile(t, root, "build/web/app.bundle.js", "bundle")

	result, err := Bundle(context.Background(), Input{Root: root, Target: "web"})
	if err != nil {
		t.Fatalf("bundle web: %v", err)
	}

	if result.OutputPath != filepath.Join(root, "build", "web", "index.html") {
		t.Fatalf("output = %s", result.OutputPath)
	}
	assertBundleFileContains(t, root, "build/web/bundle-manifest.json", `"format": "static-web"`)
	assertBundleFileContains(t, root, "build/web/bundle-manifest.json", `"app.bundle.js"`)
}

func TestBundleAndroidRunsGradleAndRecordsAPK(t *testing.T) {
	root := t.TempDir()
	writeBundleFile(t, root, "build/android/settings.gradle.kts", "include(\":app\")")
	writeBundleFile(t, root, "build/android/build.gradle.kts", "buildscript {}")
	writeBundleFile(t, root, "build/android/app/build.gradle.kts", "apply(plugin = \"com.android.application\")")
	runner := &fakeRunner{}

	result, err := Bundle(context.Background(), Input{
		Root:       root,
		Target:     "android",
		GradlePath: "fake-gradle",
		Offline:    true,
		Runner:     runner,
	})
	if err != nil {
		t.Fatalf("bundle android: %v", err)
	}

	wantArgs := []string{"--offline", "assembleDebug"}
	if !reflect.DeepEqual(runner.command.Args, wantArgs) {
		t.Fatalf("gradle args = %+v, want %+v", runner.command.Args, wantArgs)
	}
	if runner.command.Dir != filepath.Join(root, "build", "android") {
		t.Fatalf("gradle dir = %s", runner.command.Dir)
	}
	if result.OutputPath != filepath.Join(root, "build", "android", "app", "build", "outputs", "apk", "debug", "app-debug.apk") {
		t.Fatalf("output = %s", result.OutputPath)
	}
	assertBundleFileContains(t, root, "build/android/bundle-manifest.json", `"format": "apk"`)
	assertBundleFileContains(t, root, "build/android/bundle-manifest.json", `"app/build/outputs/apk/debug/app-debug.apk"`)
}

func TestChooseGradlePrefersHighestGradle8Distribution(t *testing.T) {
	got := chooseGradle([]string{
		"/tmp/gradle-9.0-milestone-1-bin/hash/gradle-9.0-milestone-1/bin/gradle",
		"/tmp/gradle-8.13-bin/hash/gradle-8.13/bin/gradle",
		"/tmp/gradle-8.14.3-bin/hash/gradle-8.14.3/bin/gradle",
	})
	if !strings.Contains(got, "gradle-8.14.3") {
		t.Fatalf("chosen gradle = %s", got)
	}
}

type fakeRunner struct {
	command Command
}

func (runner *fakeRunner) Run(_ context.Context, command Command) error {
	runner.command = command
	if err := os.MkdirAll(filepath.Join(command.Dir, "app", "build", "outputs", "apk", "debug"), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(command.Dir, "app", "build", "outputs", "apk", "debug", "app-debug.apk"), []byte("apk"), 0o644)
}

func writeBundleFile(t *testing.T, root string, path string, content string) {
	t.Helper()

	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(fullPath), err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", fullPath, err)
	}
}

func assertBundleFileContains(t *testing.T, root string, path string, want string) {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(content), want) {
		t.Fatalf("%s = %s, want %q", path, string(content), want)
	}
}
