package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/artifact"
	"github.com/dwlhm/nova/internal/build"
	"github.com/dwlhm/nova/internal/bundler"
	"github.com/dwlhm/nova/internal/lexer"
	"github.com/dwlhm/nova/internal/parser"
	"github.com/dwlhm/nova/internal/project"
	"github.com/dwlhm/nova/internal/validator"
)

func Run(args []string, cwd string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: nova build|dev --target web|android")
		return 2
	}
	switch args[0] {
	case "build":
		return runBuild(args[1:], cwd, stdout, stderr)
	case "dev":
		return runDev(args[1:], cwd, stdout, stderr)
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
}

type buildResult struct {
	Project      project.Manifest
	TargetID     string
	OutputRoot   string
	ArtifactPath string
	Bundle       *bundler.Result
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
	}, stdout, stderr); !ok {
		return 1
	}
	return 0
}

func buildProject(ctx context.Context, cwd string, options buildOptions, stdout io.Writer, stderr io.Writer) (buildResult, bool) {
	targetID := options.TargetID
	if targetID == "" {
		targetID = "web"
	}
	outRoot := options.OutRoot
	if outRoot == "" {
		outRoot = "."
	}

	targetManifest, ok := build.TargetManifestFor(targetID)
	if !ok {
		fmt.Fprintf(stderr, "NVA-TARGET-019: unsupported build target %s\n", targetID)
		return buildResult{}, false
	}

	manifest, diagnostics, ok := loadManifest(cwd)
	for _, diagnostic := range diagnostics {
		fmt.Fprintln(stderr, diagnostic)
	}
	if !ok {
		return buildResult{}, false
	}

	sources, sourceDiagnostics, ok := loadSources(cwd, manifest.Project.Entry)
	for _, diagnostic := range sourceDiagnostics {
		fmt.Fprintln(stderr, diagnostic)
	}
	if !ok {
		return buildResult{}, false
	}

	layoutDiagnostics := project.ValidateLayout(manifest, projectFiles(sources))
	if len(layoutDiagnostics) > 0 {
		for _, diagnostic := range layoutDiagnostics {
			fmt.Fprintf(stderr, "%s: %s\n", diagnostic.Code, diagnostic.Message)
		}
		return buildResult{}, false
	}

	for _, source := range sources {
		semanticDiagnostics := validator.Validate(source.File)
		if len(semanticDiagnostics) > 0 {
			for _, diagnostic := range semanticDiagnostics {
				fmt.Fprintf(stderr, "NVA-SEMANTIC-001: %s\n", diagnostic.Message)
			}
			return buildResult{}, false
		}
	}

	styleAssets, styleDiagnostics, ok := loadStyleAssets(cwd, manifest, targetID)
	for _, diagnostic := range styleDiagnostics {
		fmt.Fprintln(stderr, diagnostic)
	}
	if !ok {
		return buildResult{}, false
	}

	resolution := build.Resolve(build.ResolutionInput{
		Project:        manifest,
		Target:         targetID,
		Sources:        sources,
		TargetManifest: targetManifest,
	})
	if len(resolution.Diagnostics) > 0 {
		for _, diagnostic := range resolution.Diagnostics {
			fmt.Fprintf(stderr, "%s: %s\n", diagnostic.Code, diagnostic.Message)
		}
		return buildResult{}, false
	}

	files, artifactDiagnostics := artifact.Generate(artifact.GenerateInput{
		Project:        manifest,
		Plan:           resolution.Plan,
		Sources:        sources,
		TargetManifest: targetManifest,
		StyleAssets:    styleAssets,
	})
	if len(artifactDiagnostics) > 0 {
		for _, diagnostic := range artifactDiagnostics {
			fmt.Fprintf(stderr, "%s: %s\n", diagnostic.Code, diagnostic.Message)
		}
		return buildResult{}, false
	}

	outputRoot := filepath.Join(cwd, filepath.FromSlash(outRoot))
	if err := cleanTargetOutput(outputRoot, targetID); err != nil {
		fmt.Fprintf(stderr, "NVA-TOOL-001: %s\n", err.Error())
		return buildResult{}, false
	}
	if err := writeArtifactFiles(outputRoot, files); err != nil {
		fmt.Fprintf(stderr, "NVA-TOOL-001: %s\n", err.Error())
		return buildResult{}, false
	}

	result := buildResult{
		Project:      manifest,
		TargetID:     targetID,
		OutputRoot:   outputRoot,
		ArtifactPath: filepath.Join(outputRoot, "build", targetID),
	}

	fmt.Fprintf(stdout, "built %s artifact in %s\n", targetID, filepath.ToSlash(filepath.Join(outRoot, "build", targetID)))
	if options.BundleTarget {
		bundleResult, err := bundler.Bundle(ctx, bundler.Input{
			Root:       outputRoot,
			Target:     targetID,
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
		fmt.Fprintf(stdout, "bundled %s output at %s\n", targetID, bundler.RelativePath(outputRoot, bundleResult.OutputPath))
		fmt.Fprintf(stdout, "bundle manifest written to %s\n", bundler.RelativePath(outputRoot, bundleResult.ManifestPath))
	}
	return result, true
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

func loadSources(cwd string, entry string) ([]build.SourceFile, []string, bool) {
	paths, err := discoverSourcePaths(cwd, entry)
	if err != nil {
		return nil, []string{fmt.Sprintf("NVA-LAYOUT-007: %s", err.Error())}, false
	}

	sources := make([]build.SourceFile, 0, len(paths))
	diagnostics := make([]string, 0)
	for _, sourcePath := range paths {
		content, err := os.ReadFile(filepath.Join(cwd, filepath.FromSlash(sourcePath)))
		if err != nil {
			diagnostics = append(diagnostics, fmt.Sprintf("NVA-LAYOUT-007: read %s: %s", sourcePath, err.Error()))
			continue
		}
		file, parserDiagnostics := parser.Parse(lexer.Tokenize(string(content)))
		for _, diagnostic := range parserDiagnostics {
			diagnostics = append(diagnostics, fmt.Sprintf("NVA-PARSE-001: %s in %s", diagnostic.Message, sourcePath))
		}
		sources = append(sources, build.SourceFile{Path: sourcePath, File: file})
	}
	return sources, diagnostics, len(diagnostics) == 0
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

func projectFiles(sources []build.SourceFile) []project.File {
	files := make([]project.File, 0, len(sources)+1)
	files = append(files, project.File{Path: "nova.toml"})
	for _, source := range sources {
		files = append(files, project.File{Path: source.Path})
	}
	return files
}

func loadStyleAssets(cwd string, manifest project.Manifest, targetID string) ([]artifact.StyleAsset, []string, bool) {
	if targetID != "web" {
		return nil, nil, true
	}
	target := manifest.Targets[targetID]
	assets := make([]artifact.StyleAsset, 0, len(target.Styles))
	diagnostics := make([]string, 0)
	for _, stylePath := range target.Styles {
		cleanPath, ok := cleanProjectStylePath(stylePath)
		if !ok {
			diagnostics = append(diagnostics, fmt.Sprintf("NVA-STYLE-001: refusing unsafe stylesheet path %s", stylePath))
			continue
		}
		content, err := os.ReadFile(filepath.Join(cwd, filepath.FromSlash(cleanPath)))
		if err != nil {
			diagnostics = append(diagnostics, fmt.Sprintf("NVA-STYLE-002: read stylesheet %s: %s", cleanPath, err.Error()))
			continue
		}
		assets = append(assets, artifact.StyleAsset{SourcePath: cleanPath, Content: string(content)})
	}
	return assets, diagnostics, len(diagnostics) == 0
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
