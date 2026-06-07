package appjs

import (
	_ "embed"
)

const Version = "0.1.0"

//go:embed src/nova-app-lifecycle.js
var lifecycleJS string

func LifecycleJS() string {
	return lifecycleJS
}
