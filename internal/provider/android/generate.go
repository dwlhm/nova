package android

import (
	"fmt"
	"strings"

	"github.com/dwlhm/nova/internal/core/contract"
	viewandroid "github.com/dwlhm/nova/internal/provider/capability/view/android"
	"github.com/dwlhm/nova/internal/provider/shared"
	"github.com/dwlhm/nova/internal/provider/target"
)

func Generate(input shared.GenerateInput) ([]shared.File, []shared.Diagnostic) {
	if err := schedulerVersionCheck(); err != nil {
		return nil, []shared.Diagnostic{shared.ErrorDiagnostic("NVA-TARGET-019", err.Error())}
	}

	contractMeta, ok := shared.RuntimeContract(input.Plan.Target)
	if !ok {
		return nil, []shared.Diagnostic{shared.ErrorDiagnostic("NVA-TARGET-019", fmt.Sprintf("unsupported build target %s", input.Plan.Target))}
	}

	artifactMetadata := target.ArtifactMetadata{
		Target:             input.Plan.Target,
		EntryCapability:    input.Plan.Entry,
		LanguageVersion:    shared.LanguageVersion,
		ABIVersion:         shared.ABIVersion,
		SchedulerVersion:   shared.SchedulerVersion,
		ViewIRVersion:      shared.ViewIRVersion,
		RuntimeVersion:     shared.RuntimeVersion,
		Permissions:        input.Plan.Permissions,
		ExternalOperations: shared.ExternalOperationNames(input.Plan.ExternalOperations),
	}
	if targetDiagnostics := target.ValidateArtifact(contractMeta, artifactMetadata); len(targetDiagnostics) > 0 {
		return nil, targetDiagnostics
	}

	config, configDiagnostics := parseTargetConfig(input.Project)
	if len(configDiagnostics) > 0 {
		return nil, configDiagnostics
	}

	versions := shared.DefaultManifestVersions()
	return files(input, config, versions)
}

func files(input shared.GenerateInput, config targetConfig, versions shared.ManifestVersions) ([]shared.File, []shared.Diagnostic) {
	schedulerFiles, err := schedulerLibraryFiles()
	if err != nil {
		return nil, []shared.Diagnostic{shared.ErrorDiagnostic("NVA-TARGET-019", "android scheduler library: "+err.Error())}
	}

	app := input.Bundle.App
	manifest := shared.BuildManifest(input.Bundle, input.Plan, versions)
	out := []shared.File{
		{Path: "build/android/nova-ir/app.contract.json", Content: shared.MustJSON(app)},
		{Path: "build/android/nova-ir/build.manifest.json", Content: shared.MustJSON(manifest)},
		{Path: "build/android/settings.gradle.kts", Content: settingsGradle(input.Project.Project.Name, config)},
		{Path: "build/android/gradle.properties", Content: gradleProperties(config)},
		{Path: "build/android/build.gradle.kts", Content: rootGradle(input.Project.Project.Name, config)},
		{Path: "build/android/app/build.gradle.kts", Content: appGradle(config)},
		{Path: "build/android/app/src/main/AndroidManifest.xml", Content: androidManifestXML(config, input.Plan.Permissions, hasRouteState(input.Bundle.App))},
	}

	viewFiles, viewDiagnostics := viewandroid.Compose(input)
	if len(viewDiagnostics) > 0 {
		return nil, viewDiagnostics
	}

	sourceRoot := "build/android/app/src/main/java/" + strings.ReplaceAll(config.Namespace, ".", "/")
	out = append(out, schedulerFiles...)
	out = append(out, viewFiles...)
	out = append(out, rendererAdapterFiles(input.Plan.Renderer.Extensions)...)
	out = append(out, externalAdapterFiles(input.Plan.ExternalOperations, sourceRoot, input.ExternalAdapterContents)...)
	return out, nil
}

func hasRouteState(app contract.App) bool {
	for _, state := range app.Model.States {
		if state.Name == "route" {
			return true
		}
	}
	return false
}
