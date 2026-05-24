package rendererjs

import _ "embed"

// Version matches Nova artifact metadata runtimeVersion.
const Version = "0.1.0"

// Source is the production renderer primitive registry copied into web artifacts.
//
//go:embed src/nova-renderer.js
var Source string
