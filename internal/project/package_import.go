package project

import "strings"

func IsPackageImport(source string) bool {
	if !strings.HasPrefix(source, "@") {
		return false
	}
	parts := strings.Split(source, "/")
	return len(parts) >= 2 && len(parts[0]) > 1 && parts[1] != ""
}
