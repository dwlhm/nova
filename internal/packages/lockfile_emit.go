package packages

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/core/security"
)

func BuildLockfile(graph ResolvedGraph, manifests []Manifest, root string) Lockfile {
	byKey := manifestIndex(manifests)
	entries := make([]LockEntry, 0, len(graph.Packages))
	for _, resolved := range graph.Packages {
		manifest, ok := byKey[manifestKey(resolved.Name, resolved.Version)]
		if !ok {
			continue
		}
		entry := LockEntry{
			Name:           manifest.Name,
			Version:        manifest.Version,
			Source:         packageSourcePath(root, manifest.Name),
			ContentHash:    manifest.ContentHash,
			Dependencies:   cloneDependencies(manifest.Dependencies),
			TargetAdapters: targetAdapterPaths(manifest.Targets),
			Permissions:    clonePermissionMap(manifest.Permissions),
			ABI:            manifest.ABI,
		}
		entries = append(entries, entry)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
	return Lockfile{Entries: entries}
}

func FormatLockfile(lockfile Lockfile) string {
	if len(lockfile.Entries) == 0 {
		return ""
	}
	var builder strings.Builder
	entries := append([]LockEntry(nil), lockfile.Entries...)
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
	for index, entry := range entries {
		if index > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString("[[packages]]\n")
		builder.WriteString(fmt.Sprintf("name = %q\n", entry.Name))
		builder.WriteString(fmt.Sprintf("version = %q\n", entry.Version))
		if entry.Source != "" {
			builder.WriteString(fmt.Sprintf("source = %q\n", entry.Source))
		}
		if entry.ContentHash != "" {
			builder.WriteString(fmt.Sprintf("content_hash = %q\n", entry.ContentHash))
		}
		if entry.ABI != "" {
			builder.WriteString(fmt.Sprintf("abi = %q\n", entry.ABI))
		}
		if len(entry.Dependencies) > 0 {
			builder.WriteString("\n[packages.dependencies]\n")
			for _, dep := range sortedDependencies(entry.Dependencies) {
				builder.WriteString(fmt.Sprintf("%s = %q\n", dep.Name, dep.Constraint))
			}
		}
		if len(entry.Permissions) > 0 {
			builder.WriteString("\n[packages.permissions]\n")
			for _, permission := range sortedPermissions(entry.Permissions) {
				builder.WriteString(fmt.Sprintf("%s = %t\n", permission, entry.Permissions[permission]))
			}
		}
		if len(entry.TargetAdapters) > 0 {
			builder.WriteString("\n[packages.target_adapters]\n")
			for _, targetID := range sortedTargetAdapters(entry.TargetAdapters) {
				builder.WriteString(fmt.Sprintf("%s = %q\n", targetID, entry.TargetAdapters[targetID]))
			}
		}
	}
	builder.WriteString("\n")
	return builder.String()
}

func manifestIndex(manifests []Manifest) map[string]Manifest {
	index := make(map[string]Manifest, len(manifests))
	for _, manifest := range manifests {
		index[manifestKey(manifest.Name, manifest.Version)] = manifest
	}
	return index
}

func manifestKey(name string, version string) string {
	return name + "@" + version
}

func packageSourcePath(root string, packageName string) string {
	_ = root
	trimmed := strings.TrimPrefix(packageName, "@")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) == 2 {
		return filepathJoin("packages", parts[0], parts[1])
	}
	return filepathJoin("packages", trimmed)
}

func filepathJoin(parts ...string) string {
	return strings.Join(parts, "/")
}

func targetAdapterPaths(targets map[string]TargetAdapter) map[string]string {
	out := make(map[string]string, len(targets))
	for targetID, adapter := range targets {
		if adapter.Adapter != "" {
			out[targetID] = adapter.Adapter
		}
	}
	return out
}

func cloneDependencies(deps []Dependency) []Dependency {
	out := make([]Dependency, len(deps))
	copy(out, deps)
	return out
}

func clonePermissionMap(values security.PermissionMap) security.PermissionMap {
	if len(values) == 0 {
		return nil
	}
	out := make(security.PermissionMap, len(values))
	for permission, enabled := range values {
		out[permission] = enabled
	}
	return out
}

func sortedPermissions(values security.PermissionMap) []security.Permission {
	keys := make([]security.Permission, 0, len(values))
	for permission := range values {
		keys = append(keys, permission)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})
	return keys
}

func sortedTargetAdapters(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for targetID := range values {
		keys = append(keys, targetID)
	}
	sort.Strings(keys)
	return keys
}
