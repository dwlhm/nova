package packageio

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/dwlhm/nova/internal/core/diagnostic"
	"github.com/dwlhm/nova/internal/packages"
	"github.com/dwlhm/nova/internal/project"
)

func GenerateLockfile(root string, targetID string, manifest project.Manifest) (packages.Lockfile, []diagnostic.Diagnostic) {
	graph, diagnostics := ResolveProjectGraph(root, targetID, manifest, ResolveOptions{Production: false})
	if diagnostic.HasErrors(diagnostics) {
		return packages.Lockfile{}, diagnostics
	}
	manifests, loadDiagnostics := LoadProjectManifests(root, targetID, RendererDependencies(manifest.Renderer.ExtensionPackages))
	diagnostics = append(diagnostics, loadDiagnostics...)
	if diagnostic.HasErrors(diagnostics) {
		return packages.Lockfile{}, diagnostics
	}
	return packages.BuildLockfile(graph, manifests, root), diagnostic.StableSort(diagnostics)
}

func WriteLockfile(root string, lockfile packages.Lockfile) error {
	content := packages.FormatLockfile(lockfile)
	path := filepath.Join(root, "nova.lock")
	return os.WriteFile(path, []byte(content), 0o644)
}

func LockfilesEqual(left packages.Lockfile, right packages.Lockfile) bool {
	return reflect.DeepEqual(normalizeLockfile(left), normalizeLockfile(right))
}

func normalizeLockfile(lockfile packages.Lockfile) packages.Lockfile {
	entries := make([]packages.LockEntry, len(lockfile.Entries))
	copy(entries, lockfile.Entries)
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
	for i := range entries {
		if entries[i].TargetAdapters == nil {
			entries[i].TargetAdapters = map[string]string{}
		}
	}
	return packages.Lockfile{Entries: entries}
}
