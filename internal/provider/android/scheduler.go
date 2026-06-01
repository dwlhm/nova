package android

import (
	"fmt"

	"github.com/dwlhm/nova/internal/provider/shared"
	schedulerjava "github.com/dwlhm/nova/runtime/nova-scheduler-java"
)

func schedulerLibraryFiles() ([]shared.File, error) {
	sources, err := schedulerjava.SourceFiles()
	if err != nil {
		return nil, err
	}
	root := "build/android/nova-scheduler/src/main/java/" + schedulerjava.PackagePath
	files := make([]shared.File, 0, len(sources)+1)
	for _, source := range sources {
		files = append(files, shared.File{
			Path:    root + "/" + source.RelativePath,
			Content: source.Content,
		})
	}
	files = append(files, shared.File{Path: "build/android/nova-scheduler/build.gradle.kts", Content: schedulerGradle()})
	return files, nil
}

func schedulerGradle() string {
	return `plugins {
    java
}

group = "dev.nova"
version = "` + schedulerjava.Version + `"

java {
    toolchain {
        languageVersion.set(JavaLanguageVersion.of(17))
    }
}
`
}

func javaSchedulerActivityHost() string {
	return `    private void dispatch(String eventName, List<Object> args) {
        scheduler.dispatch("renderer", eventName, args);
    }

    @Override
    public Map<String, Object> schedulerState() {
        return state;
    }

    @Override
    public List<NovaTransition> schedulerTransitions() {
        return transitions();
    }

    @Override
    public List<String> schedulerRoutePatterns() {
        return routePatterns();
    }

    @Override
    public List<String> schedulerStateNames() {
        return stateNames();
    }

    @Override
    public boolean schedulerHasRouteState() {
        return hasRouteState();
    }

    @Override
    public Object schedulerCloneRoute(Object value) {
        return cloneRoute(value);
    }

    @Override
    public Object schedulerRouteValueForShape(Object value, List<String> patterns) {
        return routeValueForShape(value, patterns);
    }

    @Override
    public Object schedulerEvaluate(String expression, Map<String, Object> state, Map<String, Object> payload) {
        return evaluate(expression, state, payload);
    }

    @Override
    public void schedulerReconcileRouteBackStack(Object beforeRoute, Object afterRoute) {
        reconcileRouteBackStack(beforeRoute, afterRoute);
    }

`
}

func javaSchedulerApplyStateCommit() string {
	return `    @Override
    public void schedulerApplyStateCommit(Set<String> invalidations) {
        applyStateCommit(invalidations);
    }

`
}

func javaSchedulerLifecycleStubs() string {
	return `    @Override
    public void schedulerBeforeEvent(String eventName, List<Object> args) {}

    @Override
    public void schedulerAfterEvent(String eventName, List<Object> args) {}

    protected void schedulerMountLifecycles() {}

    protected void schedulerDisposeLifecycles() {}

`
}

func schedulerVersionCheck() error {
	if schedulerjava.Version != shared.SchedulerVersion {
		return fmt.Errorf("java scheduler library version %s does not match provider schedulerVersion %s", schedulerjava.Version, shared.SchedulerVersion)
	}
	return nil
}
