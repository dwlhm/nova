package bundler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

type Input struct {
	Root       string
	Target     string
	Variant    string
	GradlePath string
	GradleTask string
	Offline    bool
	Runner     Runner
	Stdout     io.Writer
	Stderr     io.Writer
}

type Result struct {
	Target       string
	ArtifactDir  string
	ManifestPath string
	OutputPath   string
}

type Command struct {
	Path   string
	Args   []string
	Dir    string
	Env    []string
	Stdout io.Writer
	Stderr io.Writer
}

type Runner interface {
	Run(context.Context, Command) error
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, command Command) error {
	cmd := exec.CommandContext(ctx, command.Path, command.Args...)
	cmd.Dir = command.Dir
	cmd.Stdout = command.Stdout
	cmd.Stderr = command.Stderr
	cmd.Env = append(os.Environ(), command.Env...)
	return cmd.Run()
}

type bundleManifest struct {
	Target     string         `json:"target"`
	Format     string         `json:"format"`
	Entrypoint string         `json:"entrypoint,omitempty"`
	Outputs    []bundleOutput `json:"outputs"`
	Tool       map[string]any `json:"tool,omitempty"`
}

type bundleOutput struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

func Bundle(ctx context.Context, input Input) (Result, error) {
	root := input.Root
	if root == "" {
		root = "."
	}
	switch input.Target {
	case "web":
		return bundleWeb(root)
	case "android":
		return bundleAndroid(ctx, input.withDefaults(root))
	default:
		return Result{}, fmt.Errorf("unsupported bundle target %s", input.Target)
	}
}

func (input Input) withDefaults(root string) Input {
	input.Root = root
	if input.Variant == "" {
		input.Variant = "debug"
	}
	if input.GradleTask == "" {
		input.GradleTask = "assembleDebug"
	}
	if input.Runner == nil {
		input.Runner = ExecRunner{}
	}
	return input
}

func bundleWeb(root string) (Result, error) {
	artifactDir := filepath.Join(root, "build", "web")
	for _, required := range []string{"index.html", "assets/nova-runtime.js", "app.bundle.js"} {
		if _, err := os.Stat(filepath.Join(artifactDir, filepath.FromSlash(required))); err != nil {
			return Result{}, fmt.Errorf("web artifact is incomplete: %s", required)
		}
	}

	outputs, err := collectOutputs(artifactDir, "bundle-manifest.json")
	if err != nil {
		return Result{}, err
	}
	manifestPath := filepath.Join(artifactDir, "bundle-manifest.json")
	manifest := bundleManifest{
		Target:     "web",
		Format:     "static-web",
		Entrypoint: "index.html",
		Outputs:    outputs,
	}
	if err := writeJSON(manifestPath, manifest); err != nil {
		return Result{}, err
	}
	return Result{
		Target:       "web",
		ArtifactDir:  artifactDir,
		ManifestPath: manifestPath,
		OutputPath:   filepath.Join(artifactDir, "index.html"),
	}, nil
}

func bundleAndroid(ctx context.Context, input Input) (Result, error) {
	artifactDir := filepath.Join(input.Root, "build", "android")
	for _, required := range []string{"settings.gradle.kts", "build.gradle.kts", "app/build.gradle.kts"} {
		if _, err := os.Stat(filepath.Join(artifactDir, filepath.FromSlash(required))); err != nil {
			return Result{}, fmt.Errorf("android artifact is incomplete: %s", required)
		}
	}

	gradlePath, err := selectGradle(input.GradlePath, artifactDir)
	if err != nil {
		return Result{}, err
	}
	args := []string{}
	if input.Offline {
		args = append(args, "--offline")
	}
	args = append(args, input.GradleTask)

	env := []string{}
	if javaHome := selectJavaHome(); javaHome != "" {
		env = append(env, "JAVA_HOME="+javaHome)
	}
	command := Command{
		Path:   gradlePath,
		Args:   args,
		Dir:    artifactDir,
		Env:    env,
		Stdout: input.Stdout,
		Stderr: input.Stderr,
	}
	if err := input.Runner.Run(ctx, command); err != nil {
		return Result{}, fmt.Errorf("gradle %s failed: %w", input.GradleTask, err)
	}

	outputPath, err := findAndroidOutput(artifactDir, input.Variant)
	if err != nil {
		return Result{}, err
	}
	outputs, err := collectAndroidOutputs(artifactDir, outputPath)
	if err != nil {
		return Result{}, err
	}
	manifestPath := filepath.Join(artifactDir, "bundle-manifest.json")
	manifest := bundleManifest{
		Target:  "android",
		Format:  "apk",
		Outputs: outputs,
		Tool: map[string]any{
			"gradle":  gradlePath,
			"task":    input.GradleTask,
			"offline": input.Offline,
			"variant": input.Variant,
		},
	}
	if err := writeJSON(manifestPath, manifest); err != nil {
		return Result{}, err
	}
	return Result{
		Target:       "android",
		ArtifactDir:  artifactDir,
		ManifestPath: manifestPath,
		OutputPath:   outputPath,
	}, nil
}

func collectOutputs(root string, excludedNames ...string) ([]bundleOutput, error) {
	excluded := make(map[string]bool, len(excludedNames))
	for _, name := range excludedNames {
		excluded[name] = true
	}

	outputs := make([]bundleOutput, 0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if excluded[entry.Name()] {
			return nil
		}
		output, err := fileOutput(root, path)
		if err != nil {
			return err
		}
		outputs = append(outputs, output)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].Path < outputs[j].Path })
	return outputs, nil
}

func collectAndroidOutputs(artifactDir string, outputPath string) ([]bundleOutput, error) {
	candidates := []string{
		outputPath,
		filepath.Join(artifactDir, "settings.gradle.kts"),
		filepath.Join(artifactDir, "gradle.properties"),
		filepath.Join(artifactDir, "build.gradle.kts"),
		filepath.Join(artifactDir, "app", "build.gradle.kts"),
		filepath.Join(artifactDir, "app", "src", "main", "AndroidManifest.xml"),
		filepath.Join(artifactDir, "app", "src", "main", "kotlin", "nova", "generated", "MainActivity.kt"),
		filepath.Join(artifactDir, "app", "src", "main", "res", "values", "styles.xml"),
		filepath.Join(artifactDir, "generated", "NovaApp.kt"),
		filepath.Join(artifactDir, "generated", "NovaRoutes.kt"),
		filepath.Join(artifactDir, "generated", "NovaExternalBindings.kt"),
		filepath.Join(artifactDir, "nova-ir", "app.nova-ir.json"),
		filepath.Join(artifactDir, "nova-ir", "metadata.json"),
		filepath.Join(artifactDir, "nova-ir", "permissions.json"),
		filepath.Join(artifactDir, "nova-ir", "target-manifest.json"),
	}
	outputs := make([]bundleOutput, 0, len(candidates))
	seen := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		if _, err := os.Stat(candidate); err != nil {
			continue
		}
		output, err := fileOutput(artifactDir, candidate)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, output)
	}
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].Path < outputs[j].Path })
	return outputs, nil
}

func fileOutput(root string, path string) (bundleOutput, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return bundleOutput{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return bundleOutput{}, err
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return bundleOutput{}, err
	}
	sum := sha256.Sum256(content)
	return bundleOutput{
		Path:   filepath.ToSlash(rel),
		Bytes:  info.Size(),
		SHA256: hex.EncodeToString(sum[:]),
	}, nil
}

func writeJSON(path string, value any) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func selectGradle(explicit string, artifactDir string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if value := os.Getenv("NOVA_GRADLE"); value != "" {
		return value, nil
	}
	for _, candidate := range []string{
		filepath.Join(artifactDir, "gradlew"),
		filepath.Join(artifactDir, "gradlew.bat"),
	} {
		if isExecutable(candidate) {
			return candidate, nil
		}
	}
	if path, err := exec.LookPath("gradle"); err == nil {
		return path, nil
	}
	if path := cachedGradle(); path != "" {
		return path, nil
	}
	return "", errors.New("gradle executable not found; set NOVA_GRADLE or pass --gradle")
}

func cachedGradle() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	matches, err := filepath.Glob(filepath.Join(home, ".gradle", "wrapper", "dists", "gradle-*-bin", "*", "gradle-*", "bin", "gradle"))
	if err != nil || len(matches) == 0 {
		return ""
	}
	return chooseGradle(matches)
}

type gradleCandidate struct {
	path        string
	major       int
	minor       int
	patch       int
	preferred   bool
	parseFailed bool
}

func chooseGradle(paths []string) string {
	candidates := make([]gradleCandidate, 0, len(paths))
	for _, path := range paths {
		candidates = append(candidates, parseGradleCandidate(path))
	}
	sort.Slice(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if left.preferred != right.preferred {
			return left.preferred
		}
		if left.parseFailed != right.parseFailed {
			return !left.parseFailed
		}
		if left.major != right.major {
			return left.major > right.major
		}
		if left.minor != right.minor {
			return left.minor > right.minor
		}
		if left.patch != right.patch {
			return left.patch > right.patch
		}
		return left.path < right.path
	})
	return candidates[0].path
}

var gradleVersionPattern = regexp.MustCompile(`gradle-(\d+)\.(\d+)(?:\.(\d+))?`)

func parseGradleCandidate(path string) gradleCandidate {
	out := gradleCandidate{path: path, parseFailed: true}
	match := gradleVersionPattern.FindStringSubmatch(path)
	if len(match) == 0 {
		return out
	}
	out.major = atoi(match[1])
	out.minor = atoi(match[2])
	out.patch = atoi(match[3])
	out.preferred = out.major == 8
	out.parseFailed = false
	return out
}

func atoi(value string) int {
	if value == "" {
		return 0
	}
	out, _ := strconv.Atoi(value)
	return out
}

func selectJavaHome() string {
	for _, candidate := range []string{
		os.Getenv("NOVA_JAVA_HOME"),
		os.Getenv("JAVA_HOME"),
		"/Applications/Android Studio.app/Contents/jbr/Contents/Home",
		filepath.Join(os.Getenv("HOME"), "Applications", "Android Studio.app", "Contents", "jbr", "Contents", "Home"),
	} {
		if isUsableJavaHome(candidate) {
			return candidate
		}
	}
	return ""
}

func isUsableJavaHome(path string) bool {
	if path == "" {
		return false
	}
	if _, err := os.Stat(filepath.Join(path, "bin", "java")); err != nil {
		return false
	}
	if runtime.GOOS == "darwin" {
		_, err := os.Stat(filepath.Join(path, "lib", "libinstrument.dylib"))
		return err == nil
	}
	return true
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}

func findAndroidOutput(artifactDir string, variant string) (string, error) {
	outputRoot := filepath.Join(artifactDir, "app", "build", "outputs", "apk")
	matches, err := filepath.Glob(filepath.Join(outputRoot, variant, "*.apk"))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		matches, err = filepath.Glob(filepath.Join(outputRoot, "*", "*.apk"))
		if err != nil {
			return "", err
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("android bundle produced no apk under %s", filepath.ToSlash(outputRoot))
	}
	sort.Strings(matches)
	return matches[0], nil
}

func RelativePath(root string, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
