package artifact

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/core/contract"
	"github.com/dwlhm/nova/internal/core/diagnostic"
	"github.com/dwlhm/nova/internal/core/ir"
	"github.com/dwlhm/nova/internal/core/security"
	"github.com/dwlhm/nova/internal/project"
	"github.com/dwlhm/nova/internal/provider/build"
	"github.com/dwlhm/nova/internal/provider/target"
)

const (
	languageVersion  = "0.1.0"
	abiVersion       = "0.1.0"
	schedulerVersion = "0.1.0"
	viewIRVersion    = "0.1.0"
	runtimeVersion   = "0.1.0"
)

type GenerateInput struct {
	Project                 project.Manifest
	Bundle                  ir.Bundle
	Plan                    build.BuildPlan
	TargetManifest          build.TargetManifest
	StyleAssets             []StyleAsset
	ExternalAdapterContents map[string]string
}

type File struct {
	Path    string
	Content string
}

type StyleAsset struct {
	SourcePath string
	Content    string
	Scope      StyleScope
}

type Diagnostic = diagnostic.Diagnostic

type StyleScope string

const (
	StyleScopeGlobal StyleScope = "global"
	StyleScopeApp    StyleScope = "app"
)

func Generate(input GenerateInput) ([]File, []Diagnostic) {
	if err := schedulerLibraryVersionCheck(); err != nil {
		return nil, []Diagnostic{errorDiagnostic("NVA-TARGET-019", err.Error())}
	}

	contractMeta, ok := runtimeContract(input.Plan.Target)
	if !ok {
		return nil, []Diagnostic{errorDiagnostic("NVA-TARGET-019", fmt.Sprintf("unsupported build target %s", input.Plan.Target))}
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
	if targetDiagnostics := target.ValidateArtifact(contractMeta, artifactMetadata); len(targetDiagnostics) > 0 {
		return nil, targetDiagnostics
	}

	versions := buildManifestVersions{
		LanguageVersion:  languageVersion,
		SchedulerVersion: schedulerVersion,
		RuntimeVersion:   runtimeVersion,
	}

	switch input.Plan.Target {
	case "web":
		return webFiles(input, artifactMetadata, versions), nil
	case "android":
		config, configDiagnostics := androidConfig(input.Project)
		if len(configDiagnostics) > 0 {
			return nil, configDiagnostics
		}
		files, libDiagnostics := androidFiles(input, artifactMetadata, config, versions)
		return files, libDiagnostics
	default:
		return nil, []Diagnostic{errorDiagnostic("NVA-TARGET-019", fmt.Sprintf("unsupported build target %s", input.Plan.Target))}
	}
}

func webFiles(input GenerateInput, metadata target.ArtifactMetadata, versions buildManifestVersions) []File {
	styles := webStyleBundle(input.StyleAssets)
	extensions := webRendererExtensions(input.Plan.Renderer.Extensions)
	external := webExternalAdapters(input.Plan.ExternalOperations, input.ExternalAdapterContents)
	app := input.Bundle.App
	manifest := buildManifest(input.Bundle, input.Plan, versions)
	files := []File{
		{Path: "build/web/index.html", Content: webIndex(input.Project.Project.Name, webStyleHrefs(styles.Files), styles.RootScope, extensions.Enabled, external.Enabled)},
		{Path: "build/web/assets/nova-runtime.css", Content: webCSS()},
		{Path: "build/web/assets/nova-scheduler.js", Content: webSchedulerModule()},
		{Path: "build/web/assets/nova-renderer.js", Content: webRendererModule()},
		{Path: "build/web/assets/nova-runtime.js", Content: webRuntime()},
		{Path: "build/web/app.bundle.js", Content: webBundle(app)},
		{Path: "build/web/app.contract.json", Content: mustJSON(app)},
		{Path: "build/web/build.manifest.json", Content: mustJSON(manifest)},
		{Path: "build/web/style-manifest.json", Content: mustJSON(styles.Manifest)},
	}
	if extensions.Enabled {
		files = append(files, File{Path: "build/web/assets/renderer-extensions.js", Content: extensions.Content})
	}
	if external.Enabled {
		files = append(files, File{Path: "build/web/assets/external-adapters.js", Content: external.Content})
	}
	return append(files, styles.Files...)
}

func androidFiles(input GenerateInput, metadata target.ArtifactMetadata, config androidTargetConfig, versions buildManifestVersions) ([]File, []Diagnostic) {
	schedulerFiles, err := androidSchedulerLibraryFiles()
	if err != nil {
		return nil, []Diagnostic{errorDiagnostic("NVA-TARGET-019", "android scheduler library: "+err.Error())}
	}

	app := input.Bundle.App
	manifest := buildManifest(input.Bundle, input.Plan, versions)
	files := []File{
		{Path: "build/android/nova-ir/app.contract.json", Content: mustJSON(app)},
		{Path: "build/android/nova-ir/build.manifest.json", Content: mustJSON(manifest)},
		{Path: "build/android/settings.gradle.kts", Content: androidSettings(input.Project.Project.Name, config)},
		{Path: "build/android/gradle.properties", Content: androidGradleProperties(config)},
		{Path: "build/android/build.gradle.kts", Content: androidGradle(input.Project.Project.Name, config)},
		{Path: "build/android/app/build.gradle.kts", Content: androidAppGradle(config)},
		{Path: "build/android/app/src/main/AndroidManifest.xml", Content: androidManifest(config, input.Plan.Permissions)},
		{Path: "build/android/app/src/main/res/values/styles.xml", Content: androidStyles(config)},
	}
	sourceRoot := "build/android/app/src/main/java/" + strings.ReplaceAll(config.Namespace, ".", "/")
	files = append(files, schedulerFiles...)
	files = append(files,
		File{Path: sourceRoot + "/MainActivity.java", Content: androidMainActivity(app, config, input.StyleAssets)},
		File{Path: sourceRoot + "/NovaRuntime.java", Content: androidRuntime(config)},
		File{Path: sourceRoot + "/NovaPrimitiveRegistry.java", Content: androidPrimitiveRegistry(config)},
		File{Path: sourceRoot + "/NovaRendererExtensions.java", Content: androidRendererExtensions(input.Plan.Renderer.Extensions, config)},
		File{Path: "build/android/generated/NovaApp.java", Content: androidAppFromContract(input.Project.Project.Name, app, config)},
		File{Path: "build/android/generated/NovaRoutes.java", Content: androidRoutesFromContract(app.View, config)},
		File{Path: "build/android/generated/NovaExternalBindings.java", Content: androidExternalBindings(input.Plan.ExternalOperations, config)},
	)
	files = append(files, androidRendererAdapterFiles(input.Plan.Renderer.Extensions)...)
	files = append(files, androidExternalAdapterFiles(input.Plan.ExternalOperations, sourceRoot, input.ExternalAdapterContents)...)
	return files, nil
}

func externalOperationNames(operations []build.ResolvedExternalOperation) []string {
	names := make([]string, 0, len(operations))
	for _, operation := range operations {
		names = append(names, operation.CapabilitySource+"."+operation.Operation)
	}
	sort.Strings(names)
	return names
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

func webIndex(name string, styleHrefs []string, rootStyleScope string, rendererExtensions bool, externalAdapters bool) string {
	if strings.TrimSpace(name) == "" {
		name = "Nova App"
	}
	links := "  <link rel=\"stylesheet\" href=\"assets/nova-runtime.css\">\n"
	for _, href := range styleHrefs {
		links += "  <link rel=\"stylesheet\" href=\"" + escapeHTML(href) + "\">\n"
	}
	scopeAttr := ""
	if strings.TrimSpace(rootStyleScope) != "" {
		scopeAttr = " data-nova-style-scope=\"" + escapeHTML(rootStyleScope) + "\""
	}
	extensionScript := ""
	if rendererExtensions {
		extensionScript = "  <script src=\"assets/renderer-extensions.js\"></script>\n"
	}
	externalScript := ""
	if externalAdapters {
		externalScript = "  <script src=\"assets/external-adapters.js\"></script>\n"
	}
	return "<!doctype html>\n<html lang=\"en\">\n<head>\n  <meta charset=\"utf-8\">\n  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n  <title>" + escapeHTML(name) + "</title>\n" + links + "</head>\n<body>\n  <main id=\"nova-root\"" + scopeAttr + " aria-label=\"" + escapeHTML(name) + "\"></main>\n  <script src=\"assets/nova-scheduler.js\"></script>\n  <script src=\"assets/nova-renderer.js\"></script>\n" + extensionScript + externalScript + "  <script src=\"assets/nova-runtime.js\"></script>\n  <script src=\"app.bundle.js\"></script>\n</body>\n</html>\n"
}

type webStyles struct {
	Files     []File
	Manifest  map[string]any
	RootScope string
}

type webStyleManifestAsset struct {
	SourcePath string `json:"sourcePath"`
	OutputPath string `json:"outputPath"`
	Scope      string `json:"scope"`
	Order      int    `json:"order"`
}

func webStyleBundle(styles []StyleAsset) webStyles {
	files := make([]File, 0, len(styles))
	manifestAssets := make([]webStyleManifestAsset, 0, len(styles))
	seen := make(map[string]bool, len(styles))
	hasAppScope := false
	for _, style := range styles {
		outputPath := webStyleOutputPath(style.SourcePath)
		if outputPath == "" || seen[outputPath] {
			continue
		}
		seen[outputPath] = true
		scope := normalizeStyleScope(style.Scope)
		if scope == StyleScopeApp {
			hasAppScope = true
		}
		files = append(files, File{Path: outputPath, Content: webStyleContent(style.Content, scope)})
		manifestAssets = append(manifestAssets, webStyleManifestAsset{
			SourcePath: style.SourcePath,
			OutputPath: strings.TrimPrefix(outputPath, "build/web/"),
			Scope:      string(scope),
			Order:      len(manifestAssets),
		})
	}
	rootScope := ""
	if hasAppScope {
		rootScope = string(StyleScopeApp)
	}
	return webStyles{
		Files: files,
		Manifest: map[string]any{
			"assets": manifestAssets,
		},
		RootScope: rootScope,
	}
}

func normalizeStyleScope(scope StyleScope) StyleScope {
	if scope == StyleScopeApp {
		return StyleScopeApp
	}
	return StyleScopeGlobal
}

func webStyleContent(content string, scope StyleScope) string {
	if scope != StyleScopeApp {
		return content
	}
	return scopeCSS(content, `#nova-root[data-nova-style-scope~="app"]`)
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

func quoteCodeString(value string) string {
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

// BuildAppContract returns the runtime contract from an already lowered bundle.
func BuildAppContract(bundle ir.Bundle) contract.App {
	return bundle.App
}

func cloneIntPath(values []int) []int {
	out := make([]int, len(values))
	copy(out, values)
	return out
}
