package artifact

import (
	"fmt"

	rendererjs "github.com/dwlhm/nova/runtime/nova-renderer-js"
	schedulerjava "github.com/dwlhm/nova/runtime/nova-scheduler-java"
	schedulerjs "github.com/dwlhm/nova/runtime/nova-scheduler-js"
)

func webSchedulerModule() string {
	return schedulerjs.Source
}

func webRendererModule() string {
	return rendererjs.Source
}

func androidSchedulerLibraryFiles() ([]File, error) {
	sources, err := schedulerjava.SourceFiles()
	if err != nil {
		return nil, err
	}
	root := "build/android/nova-scheduler/src/main/java/" + schedulerjava.PackagePath
	files := make([]File, 0, len(sources)+1)
	for _, source := range sources {
		files = append(files, File{
			Path:    root + "/" + source.RelativePath,
			Content: source.Content,
		})
	}
	files = append(files, File{Path: "build/android/nova-scheduler/build.gradle.kts", Content: androidSchedulerGradle()})
	return files, nil
}

func androidSchedulerGradle() string {
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

func androidJavaSchedulerActivityHost() string {
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

func androidJavaSchedulerApplyStateCommit() string {
	return `    @Override
    public void schedulerApplyStateCommit(Set<String> invalidations) {
        applyStateCommit(invalidations);
    }

`
}

func androidJavaSchedulerLifecycleStubs() string {
	return `    @Override
    public void schedulerBeforeEvent(String eventName, List<Object> args) {}

    @Override
    public void schedulerAfterEvent(String eventName, List<Object> args) {}

    protected void schedulerMountLifecycles() {}

    protected void schedulerDisposeLifecycles() {}

`
}

func schedulerLibraryVersionCheck() error {
	if schedulerjs.Version != schedulerVersion {
		return fmt.Errorf("web scheduler library version %s does not match artifact schedulerVersion %s", schedulerjs.Version, schedulerVersion)
	}
	if schedulerjava.Version != schedulerVersion {
		return fmt.Errorf("java scheduler library version %s does not match artifact schedulerVersion %s", schedulerjava.Version, schedulerVersion)
	}
	if rendererjs.Version != runtimeVersion {
		return fmt.Errorf("web renderer library version %s does not match artifact runtimeVersion %s", rendererjs.Version, runtimeVersion)
	}
	return nil
}
