package packages

import (
	"fmt"
	"strings"

	"github.com/dwlhm/nova/internal/diagnostic"
	"github.com/dwlhm/nova/internal/security"
)

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
	for _, extension := range input.RendererExtensions {
		resolver.resolve(extension, nil)
	}
	resolver.collectRendererExtensions()

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

func (r *packageResolver) collectRendererExtensions() {
	for _, dep := range r.input.RendererExtensions {
		manifest, ok := r.selected[dep.Name]
		if !ok {
			continue
		}
		if !hasPackageType(manifest.Types, PackageRenderer) {
			r.diagnostics = append(r.diagnostics, pkgDiagnostic("NVA-PKG-015", fmt.Sprintf("renderer extension package %s must declare renderer-package type", manifest.Name)))
			continue
		}
		target := manifest.Targets[r.input.Target]
		r.graph.RendererExtensions = append(r.graph.RendererExtensions, ResolvedRendererPackage{
			Name:                 manifest.Name,
			Version:              manifest.Version,
			TargetAdapter:        target.Adapter,
			TargetAdapterContent: target.Content,
			Primitives:           rendererPrimitives(manifest),
		})
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
