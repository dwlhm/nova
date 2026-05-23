package packages

import "fmt"

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
