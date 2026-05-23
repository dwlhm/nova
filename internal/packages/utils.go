package packages

import "github.com/dwlhm/nova/internal/diagnostic"

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
