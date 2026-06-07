package externaljs

import (
	_ "embed"
)

// Version matches Nova artifact metadata when wired.
const Version = "0.1.0"

//go:embed src/nova-external-core.js
var coreSource string

// CoreJS returns the shared NovaExternal runtime helpers for browser bundles.
func CoreJS() string {
	return coreSource
}
