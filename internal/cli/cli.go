package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/bundler"
	"github.com/dwlhm/nova/internal/conformance"
	"github.com/dwlhm/nova/internal/core/compile"
	"github.com/dwlhm/nova/internal/core/diagnostic"
	novaformat "github.com/dwlhm/nova/internal/core/format"
	"github.com/dwlhm/nova/internal/lsp"
	"github.com/dwlhm/nova/internal/packageio"
	"github.com/dwlhm/nova/internal/packages"
	"github.com/dwlhm/nova/internal/project"
	"github.com/dwlhm/nova/internal/provider/artifact"
	"github.com/dwlhm/nova/internal/provider/build"
)

func Run(args []string, cwd string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: nova init|check|build|dev|test|inspect|fmt|lsp")
		return 2
	}
	switch args[0] {
	case "init":
		return runInit(args[1:], cwd, stdout, stderr)
	case "check":
		return runCheck(args[1:], cwd, stdout, stderr)
	case "build":
		return runBuild(args[1:], cwd, stdout, stderr)
	case "dev":
		return runDev(args[1:], cwd, stdout, stderr)
	case "test":
		return runTest(args[1:], cwd, stdout, stderr)
	case "inspect":
		return runInspect(args[1:], cwd, stdout, stderr)
	case "fmt":
		return runFmt(args[1:], cwd, stdout, stderr)
	case "lsp":
		return runLSP(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %s\n", args[0])
		return 2
	}
}

type buildOptions struct {
	TargetID     string
	OutRoot      string
	BundleTarget bool
	Offline      bool
	GradlePath   string
	GradleTask   string
	Production   bool
}

type buildResult struct {
	Project      project.Manifest
	TargetID     string
	OutputRoot   string
	ArtifactPath string
	Bundle       *bundler.Result
}

type projectPipeline struct {
	Project        project.Manifest
	TargetID       string
	TargetManifest build.TargetManifest
	Sources        []build.SourceFile
	Resolution     build.ResolutionResult
	PackageGraph   packages.ResolvedGraph
	LockDigest     string
	StyleAssets    []artifact.StyleAsset
	Files          []artifact.File
}

func runInit(args []string, cwd string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	flags.SetOutput(stderr)
	name := flags.String("name", filepath.Base(cwd), "project name")
	force := flags.Bool("force", false, "overwrite existing starter files")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	files := initProjectFiles(strings.TrimSpace(*name))
	if !*force {
		for _, file := range files {
			if _, err := os.Stat(filepath.Join(cwd, filepath.FromSlash(file.Path))); err == nil {
				fmt.Fprintf(stderr, "NVA-INIT-001: refusing to overwrite existing %s; pass --force to replace starter files\n", file.Path)
				return 1
			} else if !os.IsNotExist(err) {
				fmt.Fprintf(stderr, "NVA-INIT-001: inspect %s: %s\n", file.Path, err.Error())
				return 1
			}
		}
	}
	if err := writeArtifactFiles(cwd, files); err != nil {
		fmt.Fprintf(stderr, "NVA-INIT-001: %s\n", err.Error())
		return 1
	}
	fmt.Fprintf(stdout, "initialized Nova project %s\n", projectNameOrDefault(strings.TrimSpace(*name)))
	return 0
}

func runCheck(args []string, cwd string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	targetID := flags.String("target", "web", "target to check: web or android")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	pipeline, ok := runProjectPipeline(cwd, *targetID, true, true, stderr)
	if !ok {
		return 1
	}
	fmt.Fprintf(stdout, "checked %s project %s\n", pipeline.TargetID, projectNameOrDefault(pipeline.Project.Project.Name))
	return 0
}

func runBuild(args []string, cwd string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("build", flag.ContinueOnError)
	flags.SetOutput(stderr)
	targetID := flags.String("target", "web", "target to build: web or android")
	outRoot := flags.String("out", ".", "output root")
	bundleTarget := flags.Bool("bundle", true, "run target bundling after artifact generation")
	offline := flags.Bool("offline", true, "run target bundling without dependency downloads when supported")
	gradlePath := flags.String("gradle", "", "gradle executable path for android bundling")
	gradleTask := flags.String("gradle-task", "assembleDebug", "gradle task for android bundling")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	if _, ok := buildProject(context.Background(), cwd, buildOptions{
		TargetID:     *targetID,
		OutRoot:      *outRoot,
		BundleTarget: *bundleTarget,
		Offline:      *offline,
		GradlePath:   *gradlePath,
		GradleTask:   *gradleTask,
		Production:   true,
	}, stdout, stderr); !ok {
		return 1
	}
	return 0
}

func buildProject(ctx context.Context, cwd string, options buildOptions, stdout io.Writer, stderr io.Writer) (buildResult, bool) {
	outRoot := options.OutRoot
	if outRoot == "" {
		outRoot = "."
	}

	pipeline, ok := runProjectPipeline(cwd, options.TargetID, true, options.Production, stderr)
	if !ok {
		return buildResult{}, false
	}

	outputRoot := filepath.Join(cwd, filepath.FromSlash(outRoot))
	if err := cleanTargetOutput(outputRoot, pipeline.TargetID); err != nil {
		fmt.Fprintf(stderr, "NVA-TOOL-001: %s\n", err.Error())
		return buildResult{}, false
	}
	if err := writeArtifactFiles(outputRoot, pipeline.Files); err != nil {
		fmt.Fprintf(stderr, "NVA-TOOL-001: %s\n", err.Error())
		return buildResult{}, false
	}

	result := buildResult{
		Project:      pipeline.Project,
		TargetID:     pipeline.TargetID,
		OutputRoot:   outputRoot,
		ArtifactPath: filepath.Join(outputRoot, "build", pipeline.TargetID),
	}

	fmt.Fprintf(stdout, "built %s artifact in %s\n", pipeline.TargetID, filepath.ToSlash(filepath.Join(outRoot, "build", pipeline.TargetID)))
	if options.BundleTarget {
		bundleResult, err := bundler.Bundle(ctx, bundler.Input{
			Root:       outputRoot,
			Target:     pipeline.TargetID,
			GradlePath: options.GradlePath,
			GradleTask: options.GradleTask,
			Offline:    options.Offline,
			Stdout:     stdout,
			Stderr:     stderr,
		})
		if err != nil {
			fmt.Fprintf(stderr, "NVA-BUNDLE-001: %s\n", err.Error())
			return buildResult{}, false
		}
		result.Bundle = &bundleResult
		fmt.Fprintf(stdout, "bundled %s output at %s\n", pipeline.TargetID, bundler.RelativePath(outputRoot, bundleResult.OutputPath))
		fmt.Fprintf(stdout, "bundle manifest written to %s\n", bundler.RelativePath(outputRoot, bundleResult.ManifestPath))
	}
	return result, true
}

func runProjectPipeline(cwd string, targetID string, generateArtifacts bool, production bool, stderr io.Writer) (projectPipeline, bool) {
	if targetID == "" {
		targetID = "web"
	}
	targetManifest, ok := build.TargetManifestFor(targetID)
	if !ok {
		fmt.Fprintf(stderr, "NVA-TARGET-019: unsupported build target %s\n", targetID)
		return projectPipeline{}, false
	}

	manifest, diagnostics, ok := loadManifest(cwd)
	for _, diagnostic := range diagnostics {
		fmt.Fprintln(stderr, diagnostic)
	}
	if !ok {
		return projectPipeline{}, false
	}

	packageManifests, _ := packageio.LoadProjectManifests(cwd, targetID, packageio.RendererDependencies(manifest.Renderer.ExtensionPackages))
	packageExports := packageio.BuildExportIndex(cwd, packageManifests)
	sourceModules, sourceDiagnostics, ok := loadSourceModules(cwd, manifest.Project.Entry, packageExports)
	for _, diagnostic := range sourceDiagnostics {
		fmt.Fprintln(stderr, diagnostic)
	}
	if !ok {
		return projectPipeline{}, false
	}

	layoutDiagnostics := project.ValidateLayout(manifest, projectFiles(sourceModules))
	if len(layoutDiagnostics) > 0 {
		for _, diagnostic := range layoutDiagnostics {
			fmt.Fprintf(stderr, "%s: %s\n", diagnostic.Code, diagnostic.Message)
		}
		return projectPipeline{}, false
	}

	program, compileDiagnostics := compile.Compile(compile.CompileInput{
		Profile:        targetID,
		Entry:          manifest.Project.Entry,
		Sources:        sourceModules,
		PackageExports: packageExports,
	})
	if len(compileDiagnostics) > 0 {
		for _, item := range compileDiagnostics {
			fmt.Fprintf(stderr, "%s: %s\n", item.Code, item.Message)
		}
		return projectPipeline{}, false
	}
	sources := build.SourcesFromProgram(program)

	styleAssets, styleDiagnostics, ok := loadStyleAssets(cwd, manifest, targetID)
	for _, diagnostic := range styleDiagnostics {
		fmt.Fprintln(stderr, diagnostic)
	}
	if !ok {
		return projectPipeline{}, false
	}

	lockfile, _ := packageio.LoadLockfile(cwd, production, manifest)
	packageGraph, packageDiagnostics := packageio.ResolveProjectGraph(cwd, targetID, manifest, packageio.ResolveOptions{Production: production})
	for _, item := range packageDiagnostics {
		fmt.Fprintf(stderr, "%s: %s\n", item.Code, item.Message)
	}
	if diagnostic.HasErrors(packageDiagnostics) {
		return projectPipeline{}, false
	}

	resolution := build.Resolve(build.ResolutionInput{
		Project:        manifest,
		Target:         targetID,
		Sources:        sources,
		TargetManifest: targetManifest,
		PackageGraph:   packageGraph,
		PackageExports: packageExports,
	})
	if len(resolution.Diagnostics) > 0 {
		for _, diagnostic := range resolution.Diagnostics {
			fmt.Fprintf(stderr, "%s: %s\n", diagnostic.Code, diagnostic.Message)
		}
		return projectPipeline{}, false
	}

	pipeline := projectPipeline{
		Project:        manifest,
		TargetID:       targetID,
		TargetManifest: targetManifest,
		Sources:        sources,
		Resolution:     resolution,
		PackageGraph:   packageGraph,
		LockDigest:     packageio.LockDigest(lockfile),
		StyleAssets:    styleAssets,
	}
	if !generateArtifacts {
		return pipeline, true
	}
	packageManifests, adapterManifestDiagnostics := packageio.LoadProjectManifests(cwd, targetID, packageio.RendererDependencies(manifest.Renderer.ExtensionPackages))
	for _, item := range adapterManifestDiagnostics {
		fmt.Fprintf(stderr, "%s: %s\n", item.Code, item.Message)
	}
	if diagnostic.HasErrors(adapterManifestDiagnostics) {
		return projectPipeline{}, false
	}

	program, lowerDiagnostics := build.FinalizeProgram(program, resolution.Plan, packageExports)
	renderDiagnostics := build.ValidateViewRenderer(resolution.Plan, program.NovaIR.ViewIR)
	for _, item := range lowerDiagnostics {
		fmt.Fprintf(stderr, "%s: %s\n", item.Code, item.Message)
	}
	for _, item := range renderDiagnostics {
		fmt.Fprintf(stderr, "%s: %s\n", item.Code, item.Message)
	}
	if len(lowerDiagnostics) > 0 || build.HasBlockingDiagnostics(renderDiagnostics) {
		return projectPipeline{}, false
	}

	files, artifactDiagnostics := artifact.Generate(artifact.GenerateInput{
		Project:                 manifest,
		Bundle:                  program.NovaIR,
		Plan:                    resolution.Plan,
		TargetManifest:          targetManifest,
		StyleAssets:             styleAssets,
		ExternalAdapterContents: externalAdapterContents(cwd, targetID, resolution.Plan.ExternalOperations, packageManifests),
	})
	if len(artifactDiagnostics) > 0 {
		for _, diagnostic := range artifactDiagnostics {
			fmt.Fprintf(stderr, "%s: %s\n", diagnostic.Code, diagnostic.Message)
		}
		if diagnostic.HasErrors(artifactDiagnostics) {
			return projectPipeline{}, false
		}
	}
	pipeline.Files = files
	return pipeline, true
}

func initProjectFiles(name string) []artifact.File {
	name = projectNameOrDefault(name)
	return []artifact.File{
		{Path: "nova.toml", Content: starterManifest(name)},
		{Path: "src/App.nova", Content: starterSource()},
		{Path: "src/App.css", Content: starterCSS()},
	}
}

func starterManifest(name string) string {
	return `[project]
name = "` + escapeManifestString(name) + `"
version = "0.1.0"
entry = "src/App.nova"

[targets.web]
renderer = "@nova/web"
scoped_styles = ["src/App.css"]
`
}

func starterSource() string {
	return `<contract state Counter>
  count: number <- 0 {
    @increment -> count + 1;
    @decrement -> count - 1;
    @reset -> 0;
  };
/|
<template target <- web>
  <surface class <- "counter-shell">
    <text value <- "Count: " + count /|
    <row>
      <button on_press -> @decrement>
        <text value <- "-" /|
      /|
      <button on_press -> @increment>
        <text value <- "+" /|
      /|
      <button on_press -> @reset>
        <text value <- "Reset" /|
      /|
    /|
  /|
/|
`
}

func starterCSS() string {
	return `.counter-shell {
  align-items: center;
  text-align: center;
}
`
}

func projectNameOrDefault(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "nova-app"
	}
	return name
}

func escapeManifestString(value string) string {
	encoded, _ := json.Marshal(value)
	text := string(encoded)
	if len(text) < 2 {
		return value
	}
	return text[1 : len(text)-1]
}

type inspectStyleAsset struct {
	SourcePath string `json:"sourcePath"`
	Scope      string `json:"scope"`
}

func inspectSummary(pipeline projectPipeline) map[string]any {
	styles := make([]inspectStyleAsset, 0, len(pipeline.StyleAssets))
	for _, style := range pipeline.StyleAssets {
		styles = append(styles, inspectStyleAsset{SourcePath: style.SourcePath, Scope: string(style.Scope)})
	}
	return map[string]any{
		"project": map[string]any{
			"name":    pipeline.Project.Project.Name,
			"version": pipeline.Project.Project.Version,
			"entry":   pipeline.Project.Project.Entry,
		},
		"target":             pipeline.TargetID,
		"entry":              pipeline.Resolution.Plan.Entry,
		"modules":            pipeline.Resolution.Plan.Artifact.Modules,
		"template":           pipeline.Resolution.Plan.Template,
		"permissions":        pipeline.Resolution.Plan.Permissions,
		"externalOperations": pipeline.Resolution.Plan.ExternalOperations,
		"packageGraph":       pipeline.PackageGraph.Packages,
		"permissionSources":  pipeline.PackageGraph.PermissionSources,
		"lockDigest":         pipeline.LockDigest,
		"renderer":           pipeline.Resolution.Plan.Renderer,
		"styles":             styles,
	}
}

func fmtPaths(cwd string, requested []string) ([]string, error) {
	if len(requested) > 0 {
		paths := make([]string, 0, len(requested))
		for _, item := range requested {
			clean := filepath.Clean(filepath.FromSlash(item))
			if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") || filepath.Ext(clean) != ".nova" {
				return nil, fmt.Errorf("invalid Nova source path %s", item)
			}
			paths = append(paths, filepath.ToSlash(clean))
		}
		sort.Strings(paths)
		return paths, nil
	}

	paths := make([]string, 0)
	err := filepath.WalkDir(cwd, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "build":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		if filepath.Ext(path) != ".nova" {
			return nil
		}
		rel, err := filepath.Rel(cwd, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

func runInspect(args []string, cwd string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("inspect", flag.ContinueOnError)
	flags.SetOutput(stderr)
	targetID := flags.String("target", "web", "target to inspect: web or android")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	pipeline, ok := runProjectPipeline(cwd, *targetID, true, true, stderr)
	if !ok {
		return 1
	}
	encoded, err := json.MarshalIndent(inspectSummary(pipeline), "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "NVA-INSPECT-001: %s\n", err.Error())
		return 1
	}
	fmt.Fprintln(stdout, string(encoded))
	return 0
}

func runTest(args []string, cwd string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	flags.SetOutput(stderr)
	fixtures := flags.String("fixtures", "tests/conformance", "conformance fixture root")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	results := conformance.RunFixtureRoot(filepath.Join(cwd, filepath.FromSlash(*fixtures)))
	if len(results) == 0 {
		fmt.Fprintln(stdout, "no conformance fixtures found")
		return 0
	}
	passed := 0
	for _, result := range results {
		if result.Passed() {
			passed++
			continue
		}
		for _, diagnostic := range result.Diagnostics {
			fmt.Fprintf(stderr, "%s: %s: %s\n", result.Name, diagnostic.Code, diagnostic.Message)
		}
	}
	if passed != len(results) {
		fmt.Fprintf(stderr, "%d/%d conformance fixtures passed\n", passed, len(results))
		return 1
	}
	noun := "fixtures"
	if passed == 1 {
		noun = "fixture"
	}
	fmt.Fprintf(stdout, "%d conformance %s passed\n", passed, noun)
	return 0
}

func runFmt(args []string, cwd string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("fmt", flag.ContinueOnError)
	flags.SetOutput(stderr)
	check := flags.Bool("check", false, "check formatting without writing files")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	paths, err := fmtPaths(cwd, flags.Args())
	if err != nil {
		fmt.Fprintf(stderr, "NVA-FMT-001: %s\n", err.Error())
		return 1
	}
	changed := make([]string, 0)
	for _, path := range paths {
		fullPath := filepath.Join(cwd, filepath.FromSlash(path))
		content, err := os.ReadFile(fullPath)
		if err != nil {
			fmt.Fprintf(stderr, "NVA-FMT-001: read %s: %s\n", path, err.Error())
			return 1
		}
		formatted := novaformat.Nova(string(content))
		if formatted == string(content) {
			continue
		}
		changed = append(changed, path)
		if *check {
			continue
		}
		if err := os.WriteFile(fullPath, []byte(formatted), 0o644); err != nil {
			fmt.Fprintf(stderr, "NVA-FMT-001: write %s: %s\n", path, err.Error())
			return 1
		}
	}
	if len(changed) > 0 && *check {
		fmt.Fprintf(stderr, "NVA-FMT-001: unformatted Nova sources: %s\n", strings.Join(changed, ", "))
		return 1
	}
	if len(changed) == 0 {
		fmt.Fprintln(stdout, "all Nova sources formatted")
		return 0
	}
	fmt.Fprintf(stdout, "formatted %s\n", strings.Join(changed, ", "))
	return 0
}

func runLSP(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("lsp", flag.ContinueOnError)
	flags.SetOutput(stderr)
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "NVA-LSP-001: lsp does not accept positional arguments")
		return 2
	}
	if err := lsp.Run(context.Background(), os.Stdin, stdout); err != nil {
		fmt.Fprintf(stderr, "NVA-LSP-001: %s\n", err.Error())
		return 1
	}
	return 0
}

func cleanTargetOutput(root string, targetID string) error {
	cleanTarget := filepath.Clean(filepath.FromSlash(targetID))
	if cleanTarget != targetID || strings.HasPrefix(cleanTarget, ".") || strings.Contains(cleanTarget, string(filepath.Separator)) {
		return fmt.Errorf("refusing to clean invalid target output: %s", targetID)
	}
	return os.RemoveAll(filepath.Join(root, "build", cleanTarget))
}

func loadManifest(cwd string) (project.Manifest, []string, bool) {
	content, err := os.ReadFile(filepath.Join(cwd, "nova.toml"))
	if err != nil {
		return project.Manifest{}, []string{fmt.Sprintf("NVA-LAYOUT-001: read nova.toml: %s", err.Error())}, false
	}
	manifest, diagnostics := project.ParseManifest(string(content))
	out := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		out = append(out, fmt.Sprintf("%s: %s", diagnostic.Code, diagnostic.Message))
	}
	return manifest, out, len(out) == 0
}

func loadSourceModules(cwd string, entry string, packageExports map[string]string) ([]compile.SourceModule, []string, bool) {
	paths, err := discoverSourcePaths(cwd, entry)
	if err != nil {
		return nil, []string{fmt.Sprintf("NVA-LAYOUT-007: %s", err.Error())}, false
	}
	for _, exportPath := range packageExports {
		if exportPath == "" {
			continue
		}
		if !containsPath(paths, exportPath) {
			paths = append(paths, exportPath)
		}
	}
	sort.Strings(paths)

	modules := make([]compile.SourceModule, 0, len(paths))
	diagnostics := make([]string, 0)
	for _, sourcePath := range paths {
		content, err := os.ReadFile(filepath.Join(cwd, filepath.FromSlash(sourcePath)))
		if err != nil {
			diagnostics = append(diagnostics, fmt.Sprintf("NVA-LAYOUT-007: read %s: %s", sourcePath, err.Error()))
			continue
		}
		modules = append(modules, compile.SourceModule{Path: sourcePath, Content: string(content)})
	}
	return modules, diagnostics, len(diagnostics) == 0
}

func containsPath(paths []string, want string) bool {
	for _, path := range paths {
		if path == want {
			return true
		}
	}
	return false
}

func discoverSourcePaths(cwd string, entry string) ([]string, error) {
	seen := make(map[string]bool)
	paths := make([]string, 0)
	srcRoot := filepath.Join(cwd, "src")
	if _, err := os.Stat(srcRoot); err == nil {
		err = filepath.WalkDir(srcRoot, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || filepath.Ext(path) != ".nova" {
				return nil
			}
			rel, err := filepath.Rel(cwd, path)
			if err != nil {
				return err
			}
			normalized := filepath.ToSlash(rel)
			if !seen[normalized] {
				seen[normalized] = true
				paths = append(paths, normalized)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	if entry != "" && !seen[entry] {
		if _, err := os.Stat(filepath.Join(cwd, filepath.FromSlash(entry))); err != nil {
			return nil, err
		}
		paths = append(paths, entry)
	}
	sort.Strings(paths)
	return paths, nil
}

func projectFiles(sources []compile.SourceModule) []project.File {
	files := make([]project.File, 0, len(sources)+1)
	files = append(files, project.File{Path: "nova.toml"})
	for _, source := range sources {
		files = append(files, project.File{Path: source.Path})
	}
	return files
}

func loadStyleAssets(cwd string, manifest project.Manifest, targetID string) ([]artifact.StyleAsset, []string, bool) {
	target := manifest.Targets[targetID]
	assets := make([]artifact.StyleAsset, 0, len(target.Styles)+len(target.ScopedStyles))
	diagnostics := make([]string, 0)
	for _, stylePath := range target.Styles {
		asset, assetDiagnostics, ok := loadStyleAsset(cwd, stylePath, artifact.StyleScopeGlobal)
		diagnostics = append(diagnostics, assetDiagnostics...)
		if ok {
			assets = append(assets, asset)
		}
	}
	for _, stylePath := range target.ScopedStyles {
		asset, assetDiagnostics, ok := loadStyleAsset(cwd, stylePath, artifact.StyleScopeApp)
		diagnostics = append(diagnostics, assetDiagnostics...)
		if ok {
			assets = append(assets, asset)
		}
	}
	return assets, diagnostics, len(diagnostics) == 0
}

func loadStyleAsset(cwd string, stylePath string, scope artifact.StyleScope) (artifact.StyleAsset, []string, bool) {
	diagnostics := make([]string, 0)
	cleanPath, ok := cleanProjectStylePath(stylePath)
	if !ok {
		return artifact.StyleAsset{}, []string{fmt.Sprintf("NVA-STYLE-001: refusing unsafe stylesheet path %s", stylePath)}, false
	}
	content, err := os.ReadFile(filepath.Join(cwd, filepath.FromSlash(cleanPath)))
	if err != nil {
		diagnostics = append(diagnostics, fmt.Sprintf("NVA-STYLE-002: read stylesheet %s: %s", cleanPath, err.Error()))
		return artifact.StyleAsset{}, diagnostics, false
	}
	return artifact.StyleAsset{SourcePath: cleanPath, Content: string(content), Scope: scope}, nil, true
}

func cleanProjectStylePath(stylePath string) (string, bool) {
	cleanPath := filepath.Clean(filepath.FromSlash(stylePath))
	if cleanPath == "." || filepath.IsAbs(cleanPath) || strings.HasPrefix(cleanPath, "..") || filepath.Ext(cleanPath) != ".css" {
		return "", false
	}
	return filepath.ToSlash(cleanPath), true
}

func writeArtifactFiles(root string, files []artifact.File) error {
	for _, file := range files {
		cleanPath := filepath.Clean(filepath.FromSlash(file.Path))
		if strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
			return fmt.Errorf("refusing to write artifact outside output root: %s", file.Path)
		}
		fullPath := filepath.Join(root, cleanPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(fullPath, []byte(file.Content), 0o644); err != nil {
			return err
		}
	}
	return nil
}
