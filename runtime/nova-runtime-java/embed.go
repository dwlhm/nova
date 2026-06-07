package runtimejava

import (
	_ "embed"
	"strings"
)

// Version matches Nova artifact metadata runtimeVersion when wired.
const Version = "0.1.0"

//go:embed NovaRuntime.java.tmpl
var novaRuntimeBody string

//go:embed NovaHydration.java.tmpl
var novaHydrationBody string

// Source returns NovaRuntime.java for the generated app namespace.
func Source(packageName string) string {
	return classSource(packageName, novaRuntimeBody)
}

// HydrationSource returns NovaHydration.java for the generated app namespace.
func HydrationSource(packageName string) string {
	return classSource(packageName, novaHydrationBody)
}

func classSource(packageName string, body string) string {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		packageName = "nova.generated"
	}
	return "package " + packageName + ";\n\n" + body
}
