package packages

import "fmt"

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
	if hasPackageType(manifest.Types, PackageRenderer) && len(manifest.Exports) == 0 && len(manifest.Renderer.Primitives) == 0 {
		diagnostics = append(diagnostics, pkgDiagnostic("NVA-PKG-010", fmt.Sprintf("renderer package %s should declare primitive contracts", manifest.Name)))
	}
	return diagnostics
}
