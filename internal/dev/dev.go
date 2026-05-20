package dev

import "sort"

type Target string

const (
	TargetWeb     Target = "web"
	TargetAndroid Target = "android"
)

type Strategy string

const (
	StrategyNone               Strategy = "none"
	StrategyBrowserFullReload  Strategy = "browser_full_reload"
	StrategyAndroidInstallSync Strategy = "android_install_sync"
)

type FileState struct {
	Path    string
	Size    int64
	ModTime int64
}

type Snapshot map[string]FileState

type ChangeSet struct {
	Paths []string
}

func (changes ChangeSet) Empty() bool {
	return len(changes.Paths) == 0
}

type CycleInput struct {
	Target  Target
	Initial bool
	Changes ChangeSet
	Launch  bool
}

type CyclePlan struct {
	Build         bool
	Bundle        bool
	Serve         bool
	ReloadBrowser bool
	InstallAPK    bool
	LaunchAndroid bool
	Strategy      Strategy
	Reasons       []string
}

func PlanCycle(input CycleInput) CyclePlan {
	if !input.Initial && input.Changes.Empty() {
		return CyclePlan{Strategy: StrategyNone}
	}

	switch input.Target {
	case TargetWeb:
		return planWebCycle(input)
	case TargetAndroid:
		return planAndroidCycle(input)
	default:
		return CyclePlan{Strategy: StrategyNone}
	}
}

func planWebCycle(input CycleInput) CyclePlan {
	plan := CyclePlan{
		Build:    true,
		Bundle:   true,
		Serve:    input.Initial,
		Strategy: StrategyBrowserFullReload,
		Reasons:  []string{"web dev mode rebuilds the static artifact and refreshes connected browsers"},
	}
	if !input.Initial {
		plan.ReloadBrowser = true
	}
	return plan
}

func planAndroidCycle(input CycleInput) CyclePlan {
	return CyclePlan{
		Build:         true,
		Bundle:        true,
		InstallAPK:    true,
		LaunchAndroid: input.Launch,
		Strategy:      StrategyAndroidInstallSync,
		Reasons:       []string{"android dev mode uses install/update sync instead of expensive runtime HMR"},
	}
}

func DiffSnapshots(before Snapshot, after Snapshot) ChangeSet {
	changed := make([]string, 0)
	for path, next := range after {
		prev, ok := before[path]
		if !ok || prev.Size != next.Size || prev.ModTime != next.ModTime {
			changed = append(changed, path)
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok {
			changed = append(changed, path)
		}
	}
	sort.Strings(changed)
	return ChangeSet{Paths: changed}
}

type CommandSpec struct {
	Path string
	Args []string
}

func AndroidInstallCommand(adbPath string, apkPath string, userID string) CommandSpec {
	args := []string{"install"}
	if userID != "" {
		args = append(args, "--user", userID)
	}
	args = append(args, "-r", apkPath)
	return CommandSpec{
		Path: adbPath,
		Args: args,
	}
}

func AndroidLaunchCommand(adbPath string, applicationID string, activityClass string, userID string) CommandSpec {
	args := []string{"shell", "am", "start"}
	if userID != "" {
		args = append(args, "--user", userID)
	}
	args = append(args,
		"-a",
		"android.intent.action.MAIN",
		"-c",
		"android.intent.category.LAUNCHER",
		"-n",
		applicationID+"/"+activityClass,
	)
	return CommandSpec{Path: adbPath, Args: args}
}
