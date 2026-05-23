package schedulerjs

import _ "embed"

// Version matches Nova artifact metadata schedulerVersion.
const Version = "0.1.0"

// Source is the production scheduler module copied into web artifacts.
//
//go:embed src/nova-scheduler.js
var Source string
