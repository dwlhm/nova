package artifact

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/build"
	"github.com/dwlhm/nova/internal/capability"
	"github.com/dwlhm/nova/internal/diagnostic"
	"github.com/dwlhm/nova/internal/parser"
	"github.com/dwlhm/nova/internal/project"
	"github.com/dwlhm/nova/internal/security"
	"github.com/dwlhm/nova/internal/target"
	"github.com/dwlhm/nova/internal/view"
)

const (
	languageVersion  = "0.1.0"
	abiVersion       = "0.1.0"
	schedulerVersion = "0.1.0"
	viewIRVersion    = "0.1.0"
	runtimeVersion   = "0.1.0"
)

type GenerateInput struct {
	Project        project.Manifest
	Plan           build.BuildPlan
	Sources        []build.SourceFile
	TargetManifest build.TargetManifest
	StyleAssets    []StyleAsset
}

type File struct {
	Path    string
	Content string
}

type StyleAsset struct {
	SourcePath string
	Content    string
}

type Diagnostic = diagnostic.Diagnostic

func Generate(input GenerateInput) ([]File, []Diagnostic) {
	contract, ok := runtimeContract(input.Plan.Target)
	if !ok {
		return nil, []Diagnostic{errorDiagnostic("NVA-TARGET-019", fmt.Sprintf("unsupported build target %s", input.Plan.Target))}
	}

	bundle, diagnostics := buildBundle(input)
	if len(diagnostics) > 0 {
		return nil, diagnostics
	}

	artifactMetadata := target.ArtifactMetadata{
		Target:             input.Plan.Target,
		EntryCapability:    input.Plan.Entry,
		LanguageVersion:    languageVersion,
		ABIVersion:         abiVersion,
		SchedulerVersion:   schedulerVersion,
		ViewIRVersion:      viewIRVersion,
		RuntimeVersion:     runtimeVersion,
		Permissions:        input.Plan.Permissions,
		ExternalOperations: externalOperationNames(input.Plan.ExternalOperations),
	}
	if targetDiagnostics := target.ValidateArtifact(contract, artifactMetadata); len(targetDiagnostics) > 0 {
		return nil, targetDiagnostics
	}

	switch input.Plan.Target {
	case "web":
		return webFiles(input, bundle, artifactMetadata), nil
	case "android":
		config, configDiagnostics := androidConfig(input.Project)
		if len(configDiagnostics) > 0 {
			return nil, configDiagnostics
		}
		return androidFiles(input, bundle, artifactMetadata, config), nil
	default:
		return nil, []Diagnostic{errorDiagnostic("NVA-TARGET-019", fmt.Sprintf("unsupported build target %s", input.Plan.Target))}
	}
}

type irBundle struct {
	Target              string                     `json:"target"`
	Entry               string                     `json:"entry"`
	Modules             []string                   `json:"modules"`
	Template            build.TemplateRef          `json:"template"`
	CapabilityManifests []capability.Manifest      `json:"capabilityManifests"`
	Model               appModel                   `json:"model"`
	ViewIR              view.IR                    `json:"viewIR"`
	ExternalOperations  []externalOperationSummary `json:"externalOperations"`
}

type externalOperationSummary struct {
	RequestingFile   string                `json:"requestingFile"`
	CapabilitySource string                `json:"capabilitySource"`
	CapabilityName   string                `json:"capabilityName"`
	Operation        string                `json:"operation"`
	Permissions      []security.Permission `json:"permissions"`
	Implementation   string                `json:"implementation"`
}

func buildBundle(input GenerateInput) (irBundle, []Diagnostic) {
	sourceMap := sourceFiles(input.Sources)
	selected, ok := selectedTemplate(input.Plan, sourceMap)
	if !ok {
		return irBundle{}, []Diagnostic{errorDiagnostic("NVA-RENDER-001", "selected template is not available in source graph")}
	}

	viewIR, viewDiagnostics := view.Project(selected, collectStateNames(input.Sources))
	diagnostics := make([]Diagnostic, 0, len(viewDiagnostics))
	for _, viewDiagnostic := range viewDiagnostics {
		diagnostics = append(diagnostics, Diagnostic{
			Code:     "NVA-RENDER-002",
			Severity: diagnostic.SeverityError,
			Message:  viewDiagnostic.Message,
		})
	}
	if len(diagnostics) > 0 {
		return irBundle{}, diagnostics
	}

	return irBundle{
		Target:              input.Plan.Target,
		Entry:               input.Plan.Entry,
		Modules:             modulePaths(input.Plan.Modules),
		Template:            input.Plan.Template,
		CapabilityManifests: capabilityManifests(input.Plan.Modules, sourceMap),
		Model:               buildAppModel(input.Plan.Modules, sourceMap),
		ViewIR:              viewIR,
		ExternalOperations:  externalSummaries(input.Plan.ExternalOperations),
	}, nil
}

func webFiles(input GenerateInput, bundle irBundle, metadata target.ArtifactMetadata) []File {
	styleFiles := webStyleFiles(input.StyleAssets)
	files := []File{
		{Path: "build/web/index.html", Content: webIndex(input.Project.Project.Name, webStyleHrefs(styleFiles))},
		{Path: "build/web/assets/nova-runtime.css", Content: webCSS()},
		{Path: "build/web/assets/nova-runtime.js", Content: webRuntime()},
		{Path: "build/web/app.bundle.js", Content: webBundle(bundle)},
		{Path: "build/web/app.nova-ir.json", Content: mustJSON(bundle)},
		{Path: "build/web/app.source-map.json", Content: mustJSON(sourceMapSummary(input.Plan, bundle))},
		{Path: "build/web/permissions.json", Content: mustJSON(permissionSummary(input.Plan.Permissions))},
		{Path: "build/web/target-manifest.json", Content: mustJSON(targetManifestSummary(input.TargetManifest))},
		{Path: "build/web/metadata.json", Content: mustJSON(metadataSummary(metadata))},
	}
	return append(files, styleFiles...)
}

func androidFiles(input GenerateInput, bundle irBundle, metadata target.ArtifactMetadata, config androidTargetConfig) []File {
	return []File{
		{Path: "build/android/nova-ir/app.nova-ir.json", Content: mustJSON(bundle)},
		{Path: "build/android/nova-ir/app.source-map.json", Content: mustJSON(sourceMapSummary(input.Plan, bundle))},
		{Path: "build/android/nova-ir/permissions.json", Content: mustJSON(permissionSummary(input.Plan.Permissions))},
		{Path: "build/android/nova-ir/target-manifest.json", Content: mustJSON(targetManifestSummary(input.TargetManifest))},
		{Path: "build/android/nova-ir/metadata.json", Content: mustJSON(metadataSummary(metadata))},
		{Path: "build/android/settings.gradle.kts", Content: androidSettings(input.Project.Project.Name, config)},
		{Path: "build/android/gradle.properties", Content: androidGradleProperties()},
		{Path: "build/android/build.gradle.kts", Content: androidGradle(input.Project.Project.Name, config)},
		{Path: "build/android/app/build.gradle.kts", Content: androidAppGradle(config)},
		{Path: "build/android/app/src/main/AndroidManifest.xml", Content: androidManifest(config)},
		{Path: "build/android/app/src/main/res/values/styles.xml", Content: androidStyles(config)},
		{Path: "build/android/app/src/main/kotlin/nova/generated/MainActivity.kt", Content: androidMainActivity(input.Project.Project.Name, bundle, config)},
		{Path: "build/android/generated/NovaApp.kt", Content: androidApp(input.Project.Project.Name, bundle.Target, config)},
		{Path: "build/android/generated/NovaRoutes.kt", Content: androidRoutes(bundle, config)},
		{Path: "build/android/generated/NovaExternalBindings.kt", Content: androidExternalBindings(input.Plan.ExternalOperations, config)},
	}
}

func selectedTemplate(plan build.BuildPlan, sources map[string]parser.File) (parser.TemplateDecl, bool) {
	file, ok := sources[plan.Template.SourceFile]
	if !ok || plan.Template.Index < 0 || plan.Template.Index >= len(file.Templates) {
		return parser.TemplateDecl{}, false
	}
	return file.Templates[plan.Template.Index], true
}

func collectStateNames(sources []build.SourceFile) map[string]bool {
	names := make(map[string]bool)
	for _, source := range sources {
		for _, contract := range source.File.ContractStates {
			for _, state := range contract.States {
				names[state.Name] = true
			}
		}
		for _, decl := range source.File.Imports {
			if decl.Kind != parser.ImportState {
				continue
			}
			for _, item := range decl.Items {
				name := item.Name
				if item.Alias != "" {
					name = item.Alias
				}
				names[name] = true
			}
		}
	}
	return names
}

func capabilityManifests(modules []build.ModuleRef, sources map[string]parser.File) []capability.Manifest {
	manifests := make([]capability.Manifest, 0, len(modules))
	for _, module := range modules {
		file, ok := sources[module.Path]
		if !ok {
			continue
		}
		manifests = append(manifests, capability.BuildManifest(module.Path, file))
	}
	return manifests
}

func sourceFiles(sources []build.SourceFile) map[string]parser.File {
	out := make(map[string]parser.File, len(sources))
	for _, source := range sources {
		out[source.Path] = source.File
	}
	return out
}

func modulePaths(modules []build.ModuleRef) []string {
	out := make([]string, 0, len(modules))
	for _, module := range modules {
		out = append(out, module.Path)
	}
	return out
}

func externalSummaries(operations []build.ResolvedExternalOperation) []externalOperationSummary {
	summaries := make([]externalOperationSummary, 0, len(operations))
	for _, operation := range operations {
		summaries = append(summaries, externalOperationSummary{
			RequestingFile:   operation.RequestingFile,
			CapabilitySource: operation.CapabilitySource,
			CapabilityName:   operation.CapabilityName,
			Operation:        operation.Operation,
			Permissions:      clonePermissions(operation.Permissions),
			Implementation:   operation.Implementation.Path,
		})
	}
	return summaries
}

func externalOperationNames(operations []build.ResolvedExternalOperation) []string {
	names := make([]string, 0, len(operations))
	for _, operation := range operations {
		names = append(names, operation.CapabilitySource+"."+operation.Operation)
	}
	sort.Strings(names)
	return names
}

func sourceMapSummary(plan build.BuildPlan, bundle irBundle) map[string]any {
	return map[string]any{
		"entry":         plan.Entry,
		"modules":       bundle.Modules,
		"templateFile":  plan.Template.SourceFile,
		"templateIndex": plan.Template.Index,
		"target":        plan.Target,
	}
}

func permissionSummary(permissions []security.Permission) map[string]any {
	if permissions == nil {
		permissions = []security.Permission{}
	}
	return map[string]any{"permissions": permissions}
}

func targetManifestSummary(manifest build.TargetManifest) map[string]any {
	return map[string]any{
		"id":                   manifest.ID,
		"families":             manifest.Families,
		"externalCapabilities": manifest.ExternalCapabilities,
		"permissionMappings":   manifest.PermissionMappings,
	}
}

func metadataSummary(metadata target.ArtifactMetadata) map[string]any {
	permissions := metadata.Permissions
	if permissions == nil {
		permissions = []security.Permission{}
	}
	operations := metadata.ExternalOperations
	if operations == nil {
		operations = []string{}
	}
	return map[string]any{
		"target":             metadata.Target,
		"entryCapability":    metadata.EntryCapability,
		"languageVersion":    metadata.LanguageVersion,
		"abiVersion":         metadata.ABIVersion,
		"schedulerVersion":   metadata.SchedulerVersion,
		"viewIrVersion":      metadata.ViewIRVersion,
		"runtimeVersion":     metadata.RuntimeVersion,
		"permissions":        permissions,
		"externalOperations": operations,
	}
}

func runtimeContract(targetID string) (target.RuntimeContract, bool) {
	switch targetID {
	case "web":
		return target.WebContract(), true
	case "android":
		return target.AndroidContract(), true
	default:
		return target.RuntimeContract{}, false
	}
}

func webIndex(name string, styleHrefs []string) string {
	if strings.TrimSpace(name) == "" {
		name = "Nova App"
	}
	links := "  <link rel=\"stylesheet\" href=\"assets/nova-runtime.css\">\n"
	for _, href := range styleHrefs {
		links += "  <link rel=\"stylesheet\" href=\"" + escapeHTML(href) + "\">\n"
	}
	return "<!doctype html>\n<html lang=\"en\">\n<head>\n  <meta charset=\"utf-8\">\n  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n  <title>" + escapeHTML(name) + "</title>\n" + links + "</head>\n<body>\n  <main id=\"nova-root\" aria-label=\"" + escapeHTML(name) + "\"></main>\n  <script src=\"assets/nova-runtime.js\"></script>\n  <script src=\"app.bundle.js\"></script>\n</body>\n</html>\n"
}

func webStyleFiles(styles []StyleAsset) []File {
	files := make([]File, 0, len(styles))
	seen := make(map[string]bool, len(styles))
	for _, style := range styles {
		outputPath := webStyleOutputPath(style.SourcePath)
		if outputPath == "" || seen[outputPath] {
			continue
		}
		seen[outputPath] = true
		files = append(files, File{Path: outputPath, Content: style.Content})
	}
	return files
}

func webStyleHrefs(files []File) []string {
	hrefs := make([]string, 0, len(files))
	for _, file := range files {
		if href, ok := strings.CutPrefix(file.Path, "build/web/"); ok {
			hrefs = append(hrefs, href)
		}
	}
	return hrefs
}

func webStyleOutputPath(sourcePath string) string {
	clean := strings.Trim(strings.ReplaceAll(sourcePath, "\\", "/"), "/")
	if clean == "" || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return ""
	}
	return "build/web/assets/styles/" + clean
}

func mustJSON(value any) string {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(encoded) + "\n"
}

func clonePermissions(permissions []security.Permission) []security.Permission {
	out := make([]security.Permission, len(permissions))
	copy(out, permissions)
	return out
}

func quoteKotlin(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func escapeHTML(value string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;")
	return replacer.Replace(value)
}

func escapeGradleComment(value string) string {
	return strings.ReplaceAll(value, "\n", " ")
}

func errorDiagnostic(code string, message string) Diagnostic {
	return Diagnostic{Code: code, Severity: diagnostic.SeverityError, Message: message}
}
