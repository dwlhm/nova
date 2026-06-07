package packageio

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/core/diagnostic"
	"github.com/dwlhm/nova/internal/packages"
	"github.com/dwlhm/nova/internal/project"
	"github.com/dwlhm/nova/internal/provider/standard"
)

func ResolveProjectGraph(root string, targetID string, manifest project.Manifest, options ResolveOptions) (packages.ResolvedGraph, []diagnostic.Diagnostic) {
	rendererDeps := rendererDependencies(manifest.Renderer.ExtensionPackages)
	projectDeps := projectDependencies(manifest.Dependencies)
	packageManifests, diagnostics := LoadProjectManifests(root, targetID, rendererDeps)
	if diagnostic.HasErrors(diagnostics) {
		return packages.ResolvedGraph{}, diagnostics
	}

	lockfile, lockDiagnostics := LoadLockfile(root, options.Production, manifest)
	diagnostics = append(diagnostics, lockDiagnostics...)
	if diagnostic.HasErrors(diagnostics) {
		return packages.ResolvedGraph{}, diagnostics
	}

	graph, resolverDiagnostics := packages.Resolve(packages.ResolutionInput{
		Roots:              projectDeps,
		RendererExtensions: rendererDeps,
		Packages:           packageManifests,
		Lockfile:           lockfile,
		Target:             targetID,
		Production:         options.Production,
	})
	diagnostics = append(diagnostics, resolverDiagnostics...)
	return graph, diagnostic.StableSort(diagnostics)
}

func LoadProjectManifests(root string, targetID string, rendererDeps []packages.Dependency) ([]packages.Manifest, []diagnostic.Diagnostic) {
	manifests := standard.OfficialPackages()
	referenced := dependencyNameSet(rendererDeps)
	documents, diagnostics := readProjectPackageDocuments(root)
	if diagnostic.HasErrors(diagnostics) {
		return manifests, diagnostics
	}
	for _, document := range documents {
		manifest, parseDiagnostics := packages.ParseManifest(document.Content)
		diagnostics = append(diagnostics, parseDiagnostics...)
		manifest, adapterDiagnostics := attachAdapterContent(root, document.PackageDir, manifest, targetID, referenced[manifest.Name])
		diagnostics = append(diagnostics, adapterDiagnostics...)
		manifests = append(manifests, manifest)
	}
	manifests = assignContentHashes(root, manifests)
	return manifests, diagnostic.StableSort(diagnostics)
}

type manifestDocument struct {
	Path       string
	PackageDir string
	Content    string
}

func readProjectPackageDocuments(root string) ([]manifestDocument, []diagnostic.Diagnostic) {
	paths, err := discoverPackageManifestPaths(root)
	if err != nil {
		return nil, []diagnostic.Diagnostic{diagnostic.Error("NVA-PKG-013", fmt.Sprintf("discover package manifests: %s", err.Error()))}
	}
	documents := make([]manifestDocument, 0, len(paths))
	diagnostics := make([]diagnostic.Diagnostic, 0)
	for _, manifestPath := range paths {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(manifestPath)))
		if err != nil {
			diagnostics = append(diagnostics, diagnostic.Error("NVA-PKG-013", fmt.Sprintf("read %s: %s", manifestPath, err.Error())))
			continue
		}
		documents = append(documents, manifestDocument{
			Path:       manifestPath,
			PackageDir: filepath.Dir(filepath.FromSlash(manifestPath)),
			Content:    string(content),
		})
	}
	return documents, diagnostic.StableSort(diagnostics)
}

func discoverPackageManifestPaths(root string) ([]string, error) {
	paths := make([]string, 0)
	if _, err := os.Stat(filepath.Join(root, "nova.package.toml")); err == nil {
		paths = append(paths, "nova.package.toml")
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	packagesRoot := filepath.Join(root, "packages")
	if _, err := os.Stat(packagesRoot); err != nil {
		if os.IsNotExist(err) {
			return paths, nil
		}
		return nil, err
	}
	err := filepath.WalkDir(packagesRoot, func(path string, entry fs.DirEntry, err error) error {
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
		if entry.Name() != "nova.package.toml" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

func attachAdapterContent(root string, packageDir string, manifest packages.Manifest, targetID string, required bool) (packages.Manifest, []diagnostic.Diagnostic) {
	diagnostics := make([]diagnostic.Diagnostic, 0)
	for target, adapter := range manifest.Targets {
		if adapter.Adapter == "" {
			continue
		}
		content, err := readPackageAdapter(root, packageDir, adapter.Adapter)
		if err != nil {
			if required && target == targetID {
				diagnostics = append(diagnostics, diagnostic.Error("NVA-RENDER-004", fmt.Sprintf("renderer package %s adapter %s for target %s is missing: %s", manifest.Name, adapter.Adapter, target, err.Error())))
			}
			continue
		}
		adapter.Content = content
		manifest.Targets[target] = adapter
	}
	return manifest, diagnostics
}

func readPackageAdapter(root string, packageDir string, adapterPath string) (string, error) {
	cleanPath := filepath.Clean(filepath.FromSlash(adapterPath))
	if cleanPath == "." || filepath.IsAbs(cleanPath) || strings.HasPrefix(cleanPath, "..") {
		return "", fmt.Errorf("unsafe adapter path %s", adapterPath)
	}
	content, err := os.ReadFile(filepath.Join(root, packageDir, cleanPath))
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func RendererDependencies(refs []project.RendererPackageRef) []packages.Dependency {
	return rendererDependencies(refs)
}

func rendererDependencies(refs []project.RendererPackageRef) []packages.Dependency {
	deps := make([]packages.Dependency, 0, len(refs))
	for _, ref := range refs {
		if strings.TrimSpace(ref.Name) == "" {
			continue
		}
		constraint := strings.TrimSpace(ref.Constraint)
		if constraint == "" {
			constraint = "*"
		}
		deps = append(deps, packages.Dependency{Name: ref.Name, Constraint: constraint})
	}
	return deps
}

func dependencyNameSet(deps []packages.Dependency) map[string]bool {
	out := make(map[string]bool, len(deps))
	for _, dep := range deps {
		out[dep.Name] = true
	}
	return out
}
