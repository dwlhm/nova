package rendererjava

import (
	_ "embed"
	"strings"
)

// Version matches Nova artifact metadata when wired.
const Version = "0.1.0"

//go:embed NovaRenderer.java.tmpl
var novaRendererBody string

// Source returns NovaRenderer.java for the generated app namespace.
func Source(packageName string) string {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		packageName = "nova.generated"
	}
	return "package " + packageName + ";\n\n" + novaRendererBody
}
