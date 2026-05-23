package packages

import (
	"sort"

	"github.com/dwlhm/nova/internal/security"
)

func packageIndex(manifests []Manifest) map[string][]Manifest {
	index := make(map[string][]Manifest)
	for _, manifest := range manifests {
		index[manifest.Name] = append(index[manifest.Name], manifest)
	}
	for name := range index {
		sort.SliceStable(index[name], func(i, j int) bool {
			return compareVersion(index[name][i].Version, index[name][j].Version) > 0
		})
	}
	return index
}

func lockIndex(lockfile Lockfile) map[string]LockEntry {
	index := make(map[string]LockEntry, len(lockfile.Entries))
	for _, entry := range lockfile.Entries {
		index[entry.Name] = entry
	}
	return index
}

func sortedDependencies(deps []Dependency) []Dependency {
	out := make([]Dependency, len(deps))
	copy(out, deps)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].Constraint < out[j].Constraint
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func sortPermissionSources(sources map[security.Permission][]PermissionSource) {
	for permission := range sources {
		sort.SliceStable(sources[permission], func(i, j int) bool {
			if sources[permission][i].Package == sources[permission][j].Package {
				return sources[permission][i].Version < sources[permission][j].Version
			}
			return sources[permission][i].Package < sources[permission][j].Package
		})
	}
}
