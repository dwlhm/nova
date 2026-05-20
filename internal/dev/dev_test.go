package dev

import (
	"reflect"
	"testing"
)

func TestPlanWebCycleBuildsServesAndReloadsAfterChanges(t *testing.T) {
	initial := PlanCycle(CycleInput{Target: TargetWeb, Initial: true})
	if !initial.Build || !initial.Bundle || !initial.Serve {
		t.Fatalf("initial web plan = %+v, want build, bundle, and serve", initial)
	}
	if initial.ReloadBrowser {
		t.Fatalf("initial web plan should not reload before a browser is connected: %+v", initial)
	}
	if initial.Strategy != StrategyBrowserFullReload {
		t.Fatalf("initial web strategy = %s, want browser full reload", initial.Strategy)
	}

	changed := PlanCycle(CycleInput{
		Target:  TargetWeb,
		Changes: ChangeSet{Paths: []string{"src/App.nova"}},
	})
	if !changed.Build || !changed.Bundle || !changed.ReloadBrowser {
		t.Fatalf("changed web plan = %+v, want build, bundle, and browser reload", changed)
	}
}

func TestPlanAndroidCycleInstallsAndLaunchesInsteadOfRuntimeHMR(t *testing.T) {
	plan := PlanCycle(CycleInput{
		Target:  TargetAndroid,
		Initial: true,
		Launch:  true,
	})

	if !plan.Build || !plan.Bundle || !plan.InstallAPK || !plan.LaunchAndroid {
		t.Fatalf("android plan = %+v, want build, bundle, install, and launch", plan)
	}
	if plan.Strategy != StrategyAndroidInstallSync {
		t.Fatalf("android strategy = %s, want install sync", plan.Strategy)
	}
	if len(plan.Reasons) == 0 {
		t.Fatalf("android plan should explain why it avoids runtime HMR")
	}
}

func TestPlanCycleIgnoresSteadyStateAfterInitialRun(t *testing.T) {
	plan := PlanCycle(CycleInput{Target: TargetAndroid})

	if plan.Build || plan.Bundle || plan.InstallAPK || plan.Strategy != StrategyNone {
		t.Fatalf("steady state plan = %+v, want no action", plan)
	}
}

func TestDiffSnapshotsReportsAddedChangedAndRemovedPaths(t *testing.T) {
	before := Snapshot{
		"src/App.nova": {Path: "src/App.nova", Size: 10, ModTime: 100},
		"src/Old.nova": {Path: "src/Old.nova", Size: 12, ModTime: 100},
	}
	after := Snapshot{
		"nova.toml":    {Path: "nova.toml", Size: 1, ModTime: 100},
		"src/App.nova": {Path: "src/App.nova", Size: 11, ModTime: 100},
	}

	got := DiffSnapshots(before, after)
	want := ChangeSet{Paths: []string{"nova.toml", "src/App.nova", "src/Old.nova"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("changes = %+v, want %+v", got, want)
	}
}

func TestAndroidCommandSpecs(t *testing.T) {
	install := AndroidInstallCommand("adb", "build/android/app-debug.apk", "0")
	if !reflect.DeepEqual(install.Args, []string{"install", "--user", "0", "-r", "build/android/app-debug.apk"}) {
		t.Fatalf("install command = %+v", install)
	}

	launch := AndroidLaunchCommand("adb", "dev.example.demo", "nova.generated.MainActivity", "0")
	want := []string{"shell", "am", "start", "--user", "0", "-a", "android.intent.action.MAIN", "-c", "android.intent.category.LAUNCHER", "-n", "dev.example.demo/nova.generated.MainActivity"}
	if !reflect.DeepEqual(launch.Args, want) {
		t.Fatalf("launch command = %+v, want args %+v", launch, want)
	}
}

func TestAndroidCommandSpecsAllowUserlessFallback(t *testing.T) {
	install := AndroidInstallCommand("adb", "app.apk", "")
	if !reflect.DeepEqual(install.Args, []string{"install", "-r", "app.apk"}) {
		t.Fatalf("install command = %+v", install)
	}

	launch := AndroidLaunchCommand("adb", "dev.example.demo", "nova.generated.MainActivity", "")
	want := []string{"shell", "am", "start", "-a", "android.intent.action.MAIN", "-c", "android.intent.category.LAUNCHER", "-n", "dev.example.demo/nova.generated.MainActivity"}
	if !reflect.DeepEqual(launch.Args, want) {
		t.Fatalf("launch command = %+v, want args %+v", launch, want)
	}
}
