package packageio

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/core/diagnostic"
	"github.com/dwlhm/nova/internal/packages"
	"github.com/dwlhm/nova/internal/project"
)

type ResolveOptions struct {
	Production bool
}

func LoadLockfile(root string, production bool, manifest project.Manifest) (packages.Lockfile, []diagnostic.Diagnostic) {
	requireLock := production && (len(manifest.Dependencies) > 0 || len(manifest.Renderer.ExtensionPackages) > 0)
	path := filepath.Join(root, "nova.lock")
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if requireLock {
				return packages.Lockfile{}, []diagnostic.Diagnostic{
					diagnostic.Error("NVA-PKG-005", "production build requires nova.lock when project declares dependencies or renderer extensions"),
				}
			}
			return packages.Lockfile{}, nil
		}
		return packages.Lockfile{}, []diagnostic.Diagnostic{
			diagnostic.Error("NVA-PKG-016", fmt.Sprintf("read nova.lock: %s", err.Error())),
		}
	}
	lockfile, parseDiagnostics := packages.ParseLockfile(string(content))
	out := make([]diagnostic.Diagnostic, 0, len(parseDiagnostics))
	for _, item := range parseDiagnostics {
		out = append(out, item)
	}
	return lockfile, out
}

func LockDigest(lockfile packages.Lockfile) string {
	if len(lockfile.Entries) == 0 {
		return ""
	}
	names := make([]string, 0, len(lockfile.Entries))
	for _, entry := range lockfile.Entries {
		names = append(names, entry.Name)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, name := range names {
		for _, entry := range lockfile.Entries {
			if entry.Name != name {
				continue
			}
			parts = append(parts, fmt.Sprintf("%s@%s:%s", entry.Name, entry.Version, entry.ContentHash))
			break
		}
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:])
}

func BuildExportIndex(root string, manifests []packages.Manifest) map[string]string {
	index := make(map[string]string)
	for _, manifest := range manifests {
		packageDir, ok := packageDirectory(root, manifest.Name)
		if !ok {
			continue
		}
		for export, relPath := range manifest.Exports {
			sourcePath := filepath.ToSlash(filepath.Join(packageDir, filepath.FromSlash(relPath)))
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(sourcePath))); err != nil {
				continue
			}
			index[manifest.Name+"/"+export] = sourcePath
		}
	}
	return index
}

func packageDirectory(root string, packageName string) (string, bool) {
	if strings.TrimSpace(packageName) == "" {
		return "", false
	}
	candidates := []string{
		filepath.ToSlash(filepath.Join("packages", strings.TrimPrefix(packageName, "@"))),
	}
	trimmed := strings.TrimPrefix(packageName, "@")
	if parts := strings.SplitN(trimmed, "/", 2); len(parts) == 2 {
		candidates = append(candidates, filepath.ToSlash(filepath.Join("packages", parts[0], parts[1])))
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(candidate), "nova.package.toml")); err == nil {
			return candidate, true
		}
	}
	return "", false
}

func projectDependencies(deps []project.Dependency) []packages.Dependency {
	out := make([]packages.Dependency, 0, len(deps))
	for _, dep := range deps {
		if strings.TrimSpace(dep.Name) == "" {
			continue
		}
		constraint := strings.TrimSpace(dep.Constraint)
		if constraint == "" {
			constraint = "*"
		}
		out = append(out, packages.Dependency{Name: dep.Name, Constraint: constraint})
	}
	return out
}
