package web

import (
	"fmt"

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

	versions := shared.DefaultManifestVersions()
	return files(input, versions), nil
}

func files(input shared.GenerateInput, versions shared.ManifestVersions) []shared.File {
	styles := styleBundle(input.StyleAssets)
	extensions := rendererExtensions(input.Plan.Renderer.Extensions)
	external := externalAdapters(input.Plan.ExternalOperations, input.ExternalAdapterContents)
	app := input.Bundle.App
	manifest := shared.BuildManifest(input.Bundle, input.Plan, versions)
	out := []shared.File{
		{Path: "build/web/index.html", Content: indexHTML(input.Project.Project.Name, styleHrefs(styles.Files), styles.RootScope, extensions.Enabled, external.Enabled)},
		{Path: "build/web/assets/nova-runtime.css", Content: webCSS()},
		{Path: "build/web/assets/nova-scheduler.js", Content: schedulerModule()},
		{Path: "build/web/assets/nova-renderer.js", Content: rendererModule()},
		{Path: "build/web/assets/nova-runtime.js", Content: webRuntime()},
		{Path: "build/web/app.bundle.js", Content: webBundle(app)},
		{Path: "build/web/app.contract.json", Content: shared.MustJSON(app)},
		{Path: "build/web/build.manifest.json", Content: shared.MustJSON(manifest)},
		{Path: "build/web/style-manifest.json", Content: shared.MustJSON(styles.Manifest)},
	}
	if extensions.Enabled {
		out = append(out, shared.File{Path: "build/web/assets/renderer-extensions.js", Content: extensions.Content})
	}
	if external.Enabled {
		out = append(out, shared.File{Path: "build/web/assets/external-adapters.js", Content: external.Content})
	}
	return append(out, styles.Files...)
}
