package externaljava

import (
	_ "embed"
	"strings"
)

// Version matches Nova artifact metadata when wired.
const Version = "0.1.0"

//go:embed NovaExternal.java.tmpl
var novaExternalBody string

// Source returns NovaExternal.java for the generated app namespace.
func Source(packageName string) string {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		packageName = "nova.generated"
	}
	return "package " + packageName + ";\n\n" + novaExternalBody
}
