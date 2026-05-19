package packages

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/dwlhm/nova/internal/diagnostic"
	"github.com/dwlhm/nova/internal/security"
)

type PackageType string

const (
	PackageSource             PackageType = "source-package"
	PackageRenderer           PackageType = "renderer-package"
	PackageExternalCapability PackageType = "external-capability-package"
	PackageTargetRuntime      PackageType = "target-runtime-package"
	PackageAdapter            PackageType = "adapter-package"
	PackageTooling            PackageType = "tooling-package"
	PackageConformance        PackageType = "conformance-package"
)

type Manifest struct {
	Name         string
	Version      string
	Types        []PackageType
	Language     string
	ABI          string
	Exports      map[string]string
	Targets      map[string]TargetAdapter
	Permissions  security.PermissionMap
	Dependencies []Dependency
	ContentHash  string
}

type TargetAdapter struct {
	Adapter string
}

type Dependency struct {
	Name       string
	Constraint string
}

type LockEntry struct {
	Name           string
	Version        string
	Source         string
	ContentHash    string
	Dependencies   []Dependency
	TargetAdapters map[string]string
	Permissions    security.PermissionMap
	ABI            string
}

type Lockfile struct {
	Entries []LockEntry
}

type ResolutionInput struct {
	Roots      []Dependency
	Packages   []Manifest
	Lockfile   Lockfile
	Target     string
	Production bool
}

type ResolvedGraph struct {
	Packages          []ResolvedPackage
	PermissionSources map[security.Permission][]PermissionSource
}

type ResolvedPackage struct {
	Name          string
	Version       string
	Types         []PackageType
	TargetAdapter string
	Chain         []string
}

type PermissionSource struct {
	Package    string
	Version    string
	Permission security.Permission
}

type Diagnostic = diagnostic.Diagnostic

func ValidateManifest(manifest Manifest) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	if len(manifest.Types) == 0 {
		diagnostics = append(diagnostics, pkgDiagnostic("NVA-PKG-001", fmt.Sprintf("package %s must declare at least one package type", manifest.Name)))
	}
	if hasPackageType(manifest.Types, PackageExternalCapability) && len(manifest.Permissions) == 0 {
		diagnostics = append(diagnostics, pkgDiagnostic("NVA-PKG-003", fmt.Sprintf("external capability package %s must declare permissions", manifest.Name)))
	}
	if hasPackageType(manifest.Types, PackageTargetRuntime) && manifest.ABI == "" {
		diagnostics = append(diagnostics, pkgDiagnostic("NVA-PKG-009", fmt.Sprintf("target runtime package %s must declare ABI compatibility", manifest.Name)))
	}
	if hasPackageType(manifest.Types, PackageRenderer) && len(manifest.Exports) == 0 {
		diagnostics = append(diagnostics, pkgDiagnostic("NVA-PKG-010", fmt.Sprintf("renderer package %s should declare primitive contracts", manifest.Name)))
	}
	return diagnostics
}

func Resolve(input ResolutionInput) (ResolvedGraph, []Diagnostic) {
	resolver := packageResolver{
		input:       input,
		byName:      packageIndex(input.Packages),
		locks:       lockIndex(input.Lockfile),
		selected:    make(map[string]Manifest),
		graph:       ResolvedGraph{PermissionSources: make(map[security.Permission][]PermissionSource)},
		diagnostics: make([]Diagnostic, 0),
	}

	for _, root := range sortedDependencies(input.Roots) {
		resolver.resolve(root, nil)
	}

	sortPermissionSources(resolver.graph.PermissionSources)
	return resolver.graph, diagnostic.StableSort(resolver.diagnostics)
}

type packageResolver struct {
	input       ResolutionInput
	byName      map[string][]Manifest
	locks       map[string]LockEntry
	selected    map[string]Manifest
	graph       ResolvedGraph
	diagnostics []Diagnostic
}

func (r *packageResolver) resolve(dep Dependency, chain []string) {
	nextChain := append(cloneStrings(chain), dep.Name)
	if selected, ok := r.selected[dep.Name]; ok {
		if !versionSatisfies(selected.Version, dep.Constraint) {
			r.diagnostics = append(r.diagnostics, pkgDiagnostic("NVA-PKG-002", fmt.Sprintf("package %s version %s does not satisfy %s through %s", dep.Name, selected.Version, dep.Constraint, strings.Join(nextChain, " -> "))))
		}
		return
	}

	manifest, ok := r.selectManifest(dep)
	if !ok {
		r.diagnostics = append(r.diagnostics, pkgDiagnostic("NVA-PKG-011", fmt.Sprintf("package %s satisfying %s was not found through %s", dep.Name, dep.Constraint, strings.Join(nextChain, " -> "))))
		return
	}

	r.selected[dep.Name] = manifest
	r.diagnostics = append(r.diagnostics, ValidateManifest(manifest)...)
	adapter := r.targetAdapter(manifest)
	r.auditLock(manifest, adapter)
	r.graph.Packages = append(r.graph.Packages, ResolvedPackage{
		Name:          manifest.Name,
		Version:       manifest.Version,
		Types:         clonePackageTypes(manifest.Types),
		TargetAdapter: adapter,
		Chain:         nextChain,
	})
	r.collectPermissions(manifest)

	for _, child := range sortedDependencies(manifest.Dependencies) {
		r.resolve(child, nextChain)
	}
}

func (r *packageResolver) selectManifest(dep Dependency) (Manifest, bool) {
	candidates := r.byName[dep.Name]
	if len(candidates) == 0 {
		return Manifest{}, false
	}
	if lock, ok := r.locks[dep.Name]; ok && lock.Version != "" {
		for _, candidate := range candidates {
			if candidate.Version == lock.Version && versionSatisfies(candidate.Version, dep.Constraint) {
				return candidate, true
			}
		}
	}
	for _, candidate := range candidates {
		if versionSatisfies(candidate.Version, dep.Constraint) {
			return candidate, true
		}
	}
	return Manifest{}, false
}

func (r *packageResolver) targetAdapter(manifest Manifest) string {
	if len(manifest.Targets) == 0 || r.input.Target == "" {
		return ""
	}
	target, ok := manifest.Targets[r.input.Target]
	if !ok {
		r.diagnostics = append(r.diagnostics, pkgDiagnostic("NVA-PKG-006", fmt.Sprintf("package %s does not support target %s", manifest.Name, r.input.Target)))
		return ""
	}
	return target.Adapter
}

func (r *packageResolver) auditLock(manifest Manifest, adapter string) {
	if !r.input.Production {
		return
	}
	lock, ok := r.locks[manifest.Name]
	if !ok {
		r.diagnostics = append(r.diagnostics, pkgDiagnostic("NVA-PKG-005", fmt.Sprintf("production build requires lockfile entry for %s", manifest.Name)))
		return
	}
	if lock.Version != "" && lock.Version != manifest.Version {
		r.diagnostics = append(r.diagnostics, pkgDiagnostic("NVA-PKG-012", fmt.Sprintf("lockfile version %s for %s does not match resolved version %s", lock.Version, manifest.Name, manifest.Version)))
	}
	if lock.ContentHash != "" && manifest.ContentHash != "" && lock.ContentHash != manifest.ContentHash {
		r.diagnostics = append(r.diagnostics, pkgDiagnostic("NVA-PKG-004", fmt.Sprintf("package %s content hash %s does not match lockfile hash %s", manifest.Name, manifest.ContentHash, lock.ContentHash)))
	}
	if adapter == "" || r.input.Target == "" {
		return
	}
	pinned, ok := lock.TargetAdapters[r.input.Target]
	if !ok {
		r.diagnostics = append(r.diagnostics, pkgDiagnostic("NVA-PKG-008", fmt.Sprintf("target adapter for package %s target %s must be pinned in lockfile", manifest.Name, r.input.Target)))
		return
	}
	if pinned != adapter {
		r.diagnostics = append(r.diagnostics, pkgDiagnostic("NVA-PKG-007", fmt.Sprintf("target adapter for package %s target %s changed from %s to %s", manifest.Name, r.input.Target, pinned, adapter)))
	}
}

func (r *packageResolver) collectPermissions(manifest Manifest) {
	for permission, enabled := range manifest.Permissions {
		if !enabled {
			continue
		}
		r.graph.PermissionSources[permission] = append(r.graph.PermissionSources[permission], PermissionSource{
			Package:    manifest.Name,
			Version:    manifest.Version,
			Permission: permission,
		})
	}
}

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

func versionSatisfies(version string, constraint string) bool {
	constraint = strings.TrimSpace(constraint)
	if constraint == "" || constraint == "*" {
		return true
	}
	parts := strings.Fields(constraint)
	if len(parts) == 0 {
		return true
	}
	if len(parts) == 1 && !strings.HasPrefix(parts[0], ">") && !strings.HasPrefix(parts[0], "<") && !strings.HasPrefix(parts[0], "=") {
		return version == parts[0]
	}
	for _, part := range parts {
		if !versionSatisfiesPart(version, part) {
			return false
		}
	}
	return true
}

func versionSatisfiesPart(version string, part string) bool {
	switch {
	case strings.HasPrefix(part, ">="):
		return compareVersion(version, strings.TrimPrefix(part, ">=")) >= 0
	case strings.HasPrefix(part, ">"):
		return compareVersion(version, strings.TrimPrefix(part, ">")) > 0
	case strings.HasPrefix(part, "<="):
		return compareVersion(version, strings.TrimPrefix(part, "<=")) <= 0
	case strings.HasPrefix(part, "<"):
		return compareVersion(version, strings.TrimPrefix(part, "<")) < 0
	case strings.HasPrefix(part, "="):
		return version == strings.TrimPrefix(part, "=")
	default:
		return version == part
	}
}

func compareVersion(left string, right string) int {
	leftParts := versionParts(left)
	rightParts := versionParts(right)
	for i := 0; i < 3; i++ {
		if leftParts[i] > rightParts[i] {
			return 1
		}
		if leftParts[i] < rightParts[i] {
			return -1
		}
	}
	return 0
}

func versionParts(version string) [3]int {
	version = strings.TrimPrefix(version, "v")
	raw := strings.Split(version, ".")
	parts := [3]int{}
	for i := 0; i < len(raw) && i < len(parts); i++ {
		value, _ := strconv.Atoi(strings.TrimLeftFunc(raw[i], func(r rune) bool {
			return r < '0' || r > '9'
		}))
		parts[i] = value
	}
	return parts
}

func hasPackageType(types []PackageType, want PackageType) bool {
	for _, typ := range types {
		if typ == want {
			return true
		}
	}
	return false
}

func clonePackageTypes(types []PackageType) []PackageType {
	out := make([]PackageType, len(types))
	copy(out, types)
	return out
}

func cloneStrings(values []string) []string {
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func pkgDiagnostic(code string, message string) Diagnostic {
	return Diagnostic{Code: code, Severity: diagnostic.SeverityError, Message: message}
}
