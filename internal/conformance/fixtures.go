package conformance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/dwlhm/nova/internal/artifact"
	"github.com/dwlhm/nova/internal/build"
	"github.com/dwlhm/nova/internal/diagnostic"
	"github.com/dwlhm/nova/internal/lexer"
	"github.com/dwlhm/nova/internal/parser"
	"github.com/dwlhm/nova/internal/project"
	"github.com/dwlhm/nova/internal/routing"
	"github.com/dwlhm/nova/internal/security"
	"github.com/dwlhm/nova/internal/validator"
	"github.com/dwlhm/nova/internal/view"
)

const FixtureSpecFile = "nova.conformance.json"

type FixtureSpec struct {
	Target   string          `json:"target"`
	Expected FixtureExpected `json:"expected"`
}

type FixtureExpected struct {
	DiagnosticCodes []string                  `json:"diagnosticCodes"`
	Artifact        *ExpectedArtifactMetadata `json:"artifact,omitempty"`
	View            *ExpectedViewMetadata     `json:"view,omitempty"`
	Scheduler       *ExpectedScheduler        `json:"scheduler,omitempty"`
}

type ExpectedArtifactMetadata struct {
	Target      string                `json:"target"`
	Entry       string                `json:"entry"`
	Modules     []string              `json:"modules"`
	Permissions []security.Permission `json:"permissions"`
}

type ExpectedViewMetadata struct {
	Bindings      int      `json:"bindings"`
	EventRoutes   int      `json:"eventRoutes"`
	Pages         int      `json:"pages"`
	RoutePatterns []string `json:"routePatterns"`
}

type FixtureResult struct {
	Name        string
	Target      string
	Diagnostics []Diagnostic
}

func (result FixtureResult) Passed() bool {
	return len(result.Diagnostics) == 0
}

func RunFixtureRoot(root string) []FixtureResult {
	fixtures, err := discoverFixtureDirs(root)
	if err != nil {
		return []FixtureResult{{
			Name:        filepath.Base(root),
			Diagnostics: []Diagnostic{fixtureDiagnostic("NVA-CONFORMANCE-020", err.Error())},
		}}
	}
	results := make([]FixtureResult, 0, len(fixtures))
	for _, fixture := range fixtures {
		results = append(results, RunFixtureDir(fixture))
	}
	return results
}

func RunFixtureDir(root string) FixtureResult {
	name := filepath.Base(root)
	spec, diagnostics := loadFixtureSpec(root)
	if len(diagnostics) > 0 {
		return FixtureResult{Name: name, Target: spec.Target, Diagnostics: diagnostics}
	}
	if spec.Target == "" {
		spec.Target = "web"
	}

	actual, actualDiagnostics := evaluateFixture(root, spec.Target)
	expectedCodes := sortedStrings(spec.Expected.DiagnosticCodes)
	actualCodes := diagnosticCodes(actualDiagnostics)
	if !reflect.DeepEqual(expectedCodes, actualCodes) {
		diagnostics = append(diagnostics, fixtureDiagnostic(
			"NVA-CONFORMANCE-010",
			fmt.Sprintf("diagnostic codes mismatch: expected %v, actual %v", expectedCodes, actualCodes),
		))
	}
	if len(actualDiagnostics) > 0 {
		return FixtureResult{Name: name, Target: spec.Target, Diagnostics: diagnostics}
	}
	diagnostics = append(diagnostics, compareExpectedArtifact(spec.Expected.Artifact, actual.Artifact)...)
	diagnostics = append(diagnostics, compareExpectedView(spec.Expected.View, actual.View)...)
	if spec.Expected.Scheduler != nil {
		if !traceSpecified(spec.Expected.Scheduler.Trace) {
			return FixtureResult{
				Name:        name,
				Target:      spec.Target,
				Diagnostics: []Diagnostic{fixtureDiagnostic("NVA-CONFORMANCE-033", "scheduler fixture requires non-empty expected trace")},
			}
		}
		schedulerDiagnostics, _, ok := runSchedulerFixture(actual.Resolution.Plan, actual.Sources, spec.Expected.Scheduler)
		if !ok {
			return FixtureResult{Name: name, Target: spec.Target, Diagnostics: diagnostic.StableSort(append(diagnostics, schedulerDiagnostics...))}
		}
		diagnostics = append(diagnostics, schedulerDiagnostics...)
	}
	return FixtureResult{Name: name, Target: spec.Target, Diagnostics: diagnostic.StableSort(diagnostics)}
}

type fixtureActual struct {
	Artifact   ExpectedArtifactMetadata
	View       ExpectedViewMetadata
	Resolution build.ResolutionResult
	Sources    []build.SourceFile
}

func evaluateFixture(root string, targetID string) (fixtureActual, []Diagnostic) {
	targetManifest, ok := build.TargetManifestFor(targetID)
	if !ok {
		return fixtureActual{}, []Diagnostic{fixtureDiagnostic("NVA-TARGET-019", fmt.Sprintf("unsupported target %s", targetID))}
	}
	manifest, manifestDiagnostics := readFixtureManifest(root)
	if len(manifestDiagnostics) > 0 {
		return fixtureActual{}, manifestDiagnostics
	}
	sources, sourceDiagnostics := readFixtureSources(root, manifest.Project.Entry)
	if len(sourceDiagnostics) > 0 {
		return fixtureActual{}, sourceDiagnostics
	}
	if layoutDiagnostics := project.ValidateLayout(manifest, fixtureProjectFiles(sources)); len(layoutDiagnostics) > 0 {
		return fixtureActual{}, projectDiagnostics(layoutDiagnostics)
	}
	if semanticDiagnostics := validateFixtureSources(sources); len(semanticDiagnostics) > 0 {
		return fixtureActual{}, semanticDiagnostics
	}

	resolution := build.Resolve(build.ResolutionInput{
		Project:        manifest,
		Target:         targetID,
		Sources:        sources,
		TargetManifest: targetManifest,
	})
	if len(resolution.Diagnostics) > 0 {
		return fixtureActual{}, buildDiagnostics(resolution.Diagnostics)
	}
	if _, artifactDiagnostics := artifact.Generate(artifact.GenerateInput{
		Project:        manifest,
		Plan:           resolution.Plan,
		Sources:        sources,
		TargetManifest: targetManifest,
	}); len(artifactDiagnostics) > 0 {
		return fixtureActual{}, artifactDiagnostics
	}

	viewMetadata, viewDiagnostics := fixtureViewMetadata(resolution.Plan, sources)
	if len(viewDiagnostics) > 0 {
		return fixtureActual{}, viewDiagnostics
	}
	return fixtureActual{
		Artifact: ExpectedArtifactMetadata{
			Target:      resolution.Plan.Artifact.Target,
			Entry:       resolution.Plan.Artifact.Entry,
			Modules:     resolution.Plan.Artifact.Modules,
			Permissions: resolution.Plan.Artifact.Permissions,
		},
		View:       viewMetadata,
		Resolution: resolution,
		Sources:    sources,
	}, nil
}

func loadFixtureSpec(root string) (FixtureSpec, []Diagnostic) {
	content, err := os.ReadFile(filepath.Join(root, FixtureSpecFile))
	if err != nil {
		return FixtureSpec{}, []Diagnostic{fixtureDiagnostic("NVA-CONFORMANCE-021", fmt.Sprintf("read %s: %s", FixtureSpecFile, err.Error()))}
	}
	var spec FixtureSpec
	if err := json.Unmarshal(content, &spec); err != nil {
		return FixtureSpec{}, []Diagnostic{fixtureDiagnostic("NVA-CONFORMANCE-022", fmt.Sprintf("parse %s: %s", FixtureSpecFile, err.Error()))}
	}
	return spec, nil
}

func readFixtureManifest(root string) (project.Manifest, []Diagnostic) {
	content, err := os.ReadFile(filepath.Join(root, "nova.toml"))
	if err != nil {
		return project.Manifest{}, []Diagnostic{fixtureDiagnostic("NVA-LAYOUT-001", fmt.Sprintf("read nova.toml: %s", err.Error()))}
	}
	manifest, diagnostics := project.ParseManifest(string(content))
	return manifest, projectDiagnostics(diagnostics)
}

func readFixtureSources(root string, entry string) ([]build.SourceFile, []Diagnostic) {
	paths, err := discoverFixtureSources(root, entry)
	if err != nil {
		return nil, []Diagnostic{fixtureDiagnostic("NVA-LAYOUT-007", err.Error())}
	}
	sources := make([]build.SourceFile, 0, len(paths))
	diagnostics := make([]Diagnostic, 0)
	for _, sourcePath := range paths {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sourcePath)))
		if err != nil {
			diagnostics = append(diagnostics, fixtureDiagnostic("NVA-LAYOUT-007", fmt.Sprintf("read %s: %s", sourcePath, err.Error())))
			continue
		}
		file, parserDiagnostics := parser.Parse(lexer.Tokenize(string(content)))
		for _, parserDiagnostic := range parserDiagnostics {
			diagnostics = append(diagnostics, fixtureDiagnostic("NVA-PARSE-001", fmt.Sprintf("%s in %s", parserDiagnostic.Message, sourcePath)))
		}
		sources = append(sources, build.SourceFile{Path: sourcePath, File: file})
	}
	return sources, diagnostics
}

func discoverFixtureSources(root string, entry string) ([]string, error) {
	seen := make(map[string]bool)
	paths := make([]string, 0)
	srcRoot := filepath.Join(root, "src")
	if _, err := os.Stat(srcRoot); err == nil {
		err = filepath.WalkDir(srcRoot, func(path string, dirEntry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if dirEntry.IsDir() || filepath.Ext(path) != ".nova" {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			normalized := filepath.ToSlash(rel)
			seen[normalized] = true
			paths = append(paths, normalized)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	if entry != "" && !seen[entry] {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(entry))); err != nil {
			return nil, err
		}
		paths = append(paths, entry)
	}
	sort.Strings(paths)
	return paths, nil
}

func validateFixtureSources(sources []build.SourceFile) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	for _, source := range sources {
		for _, validation := range validator.Validate(source.File) {
			diagnostics = append(diagnostics, fixtureDiagnostic("NVA-SEMANTIC-001", validation.Message))
		}
	}
	return diagnostics
}

func fixtureViewMetadata(plan build.BuildPlan, sources []build.SourceFile) (ExpectedViewMetadata, []Diagnostic) {
	sourceMap := make(map[string]parser.File, len(sources))
	stateNames := make(map[string]bool)
	for _, source := range sources {
		sourceMap[source.Path] = source.File
		for _, contract := range source.File.ContractStates {
			for _, state := range contract.States {
				stateNames[state.Name] = true
			}
		}
	}
	file, ok := sourceMap[plan.Template.SourceFile]
	if !ok || plan.Template.Index < 0 || plan.Template.Index >= len(file.Templates) {
		return ExpectedViewMetadata{}, []Diagnostic{fixtureDiagnostic("NVA-CONFORMANCE-023", "selected template is unavailable")}
	}
	ir, diagnostics := view.Project(file.Templates[plan.Template.Index], stateNames)
	if len(diagnostics) > 0 {
		out := make([]Diagnostic, 0, len(diagnostics))
		for _, viewDiagnostic := range diagnostics {
			out = append(out, fixtureDiagnostic("NVA-RENDER-002", viewDiagnostic.Message))
		}
		return ExpectedViewMetadata{}, out
	}
	return ExpectedViewMetadata{
		Bindings:      len(ir.Metadata.Bindings),
		EventRoutes:   len(ir.Metadata.EventRoutes),
		Pages:         len(ir.Metadata.Pages),
		RoutePatterns: fixtureRoutePatterns(ir.Metadata.Pages),
	}, nil
}

func fixtureRoutePatterns(pages []view.PageRef) []string {
	patterns := make([]string, 0, len(pages))
	for _, page := range pages {
		pattern, ok := fixtureBindingString(page.Path)
		if !ok {
			continue
		}
		patterns = append(patterns, routing.DescribePattern(pattern).Pattern)
	}
	return patterns
}

func fixtureBindingString(binding view.Binding) (string, bool) {
	tokens := trimFixtureTokens(binding.Tokens)
	if len(tokens) == 1 && tokens[0].Type == lexer.STRING {
		return tokens[0].Literal, true
	}
	if binding.Text != "" {
		return binding.Text, true
	}
	return "", false
}

func trimFixtureTokens(tokens []lexer.Token) []lexer.Token {
	start := 0
	for start < len(tokens) && (tokens[start].Type == lexer.EOF || tokens[start].Type == lexer.COMMENT || tokens[start].Type == lexer.SEMICOLON) {
		start++
	}
	end := len(tokens)
	for end > start && (tokens[end-1].Type == lexer.EOF || tokens[end-1].Type == lexer.COMMENT || tokens[end-1].Type == lexer.SEMICOLON) {
		end--
	}
	return tokens[start:end]
}

func compareExpectedArtifact(expected *ExpectedArtifactMetadata, actual ExpectedArtifactMetadata) []Diagnostic {
	if expected == nil || reflect.DeepEqual(*expected, actual) {
		return nil
	}
	return []Diagnostic{fixtureDiagnostic("NVA-CONFORMANCE-011", fmt.Sprintf("artifact mismatch: expected %+v, actual %+v", *expected, actual))}
}

func compareExpectedView(expected *ExpectedViewMetadata, actual ExpectedViewMetadata) []Diagnostic {
	if expected == nil {
		return nil
	}
	if expected.Bindings != actual.Bindings || expected.EventRoutes != actual.EventRoutes || expected.Pages != actual.Pages {
		return []Diagnostic{fixtureDiagnostic("NVA-CONFORMANCE-012", fmt.Sprintf("view metadata mismatch: expected %+v, actual %+v", *expected, actual))}
	}
	if expected.RoutePatterns != nil && !reflect.DeepEqual(expected.RoutePatterns, actual.RoutePatterns) {
		return []Diagnostic{fixtureDiagnostic("NVA-CONFORMANCE-013", fmt.Sprintf("route patterns mismatch: expected %v, actual %v", expected.RoutePatterns, actual.RoutePatterns))}
	}
	return nil
}

func discoverFixtureDirs(root string) ([]string, error) {
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	fixtures := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, dirEntry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if dirEntry.IsDir() {
			return nil
		}
		if dirEntry.Name() == FixtureSpecFile {
			fixtures = append(fixtures, filepath.Dir(path))
		}
		return nil
	})
	sort.Strings(fixtures)
	return fixtures, err
}

func fixtureProjectFiles(sources []build.SourceFile) []project.File {
	files := make([]project.File, 0, len(sources)+1)
	files = append(files, project.File{Path: "nova.toml"})
	for _, source := range sources {
		files = append(files, project.File{Path: source.Path})
	}
	return files
}

func projectDiagnostics(diagnostics []project.Diagnostic) []Diagnostic {
	out := make([]Diagnostic, 0, len(diagnostics))
	for _, item := range diagnostics {
		out = append(out, fixtureDiagnostic(item.Code, item.Message))
	}
	return out
}

func buildDiagnostics(diagnostics []build.Diagnostic) []Diagnostic {
	out := make([]Diagnostic, 0, len(diagnostics))
	for _, item := range diagnostics {
		out = append(out, fixtureDiagnostic(item.Code, item.Message))
	}
	return out
}

func diagnosticCodes(diagnostics []Diagnostic) []string {
	codes := make([]string, 0, len(diagnostics))
	for _, item := range diagnostics {
		codes = append(codes, item.Code)
	}
	return sortedStrings(codes)
}

func sortedStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func fixtureDiagnostic(code string, message string) Diagnostic {
	return Diagnostic{Code: code, Severity: diagnostic.SeverityError, Message: message}
}
