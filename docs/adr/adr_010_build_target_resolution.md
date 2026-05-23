Berikut ADR yang fokus ke **build dan target resolution** Nova.

---

# ADR-010: Build & Target Resolution

## Status

Implemented / Accepted

## Context

Nova adalah bahasa/framework lintas platform.
Build system harus memilih:

```txt
capability file
package dependency
external implementation
renderer adapter
host adapter
permission mapping
target artifact
```

Pemilihan ini harus deterministic dan dapat dijelaskan oleh diagnostics.

---

# Decision

Nova build memakai **target-aware dependency resolution**.

Input build:

```txt
project manifest
entry capability
target id
source files
package graph
target capability manifest
```

Output build:

```txt
validated semantic graph
scheduler IR
view IR / target IR
external operation table
permission table
target artifact
diagnostics
```

---

# Target IDs

Target resmi production:

```txt
web
android
```

Target future:

```txt
ios
desktop
```

Future target dapat ditambahkan selama menyediakan:

```txt
host adapter
renderer adapter
external capability manifest
conformance test result
permission mapping
artifact builder
```

---

# Project Manifest

Project manifest direkomendasikan bernama:

```txt
nova.toml
```

Minimal manifest:

```toml
[project]
name = "audiolab"
version = "0.1.0"
entry = "src/App.nova"

[targets.web]
renderer = "@nova/web"

[targets.android]
renderer = "@nova/android"          # production: Java native View

[permissions]
storage.read = true
storage.write = true
```

Manifest detail final dapat berubah, tetapi build membutuhkan informasi semantic yang sama.

---

# Resolution Inputs

Resolver membaca:

```txt
local file imports
package imports
external imports
target annotations
target manifest
package manifest
permission declaration
```

Resolver tidak menjalankan lifecycle atau external code.

---

# Import Resolution

Local import:

```nova
<import Counter from "./Counter.nova" /|
```

Resolved relative terhadap file pengimpor.

Package import:

```nova
<import Button from "@nova/ui/Button" /|
```

Resolved melalui package graph.

Environment import:

```nova
<import external storage from "@env/storage">
  ...
/|
```

Resolved melalui target capability manifest.

---

# Target-Specific Source Resolution

External implementation dapat target-specific.

Resolution order:

```txt
1. exact target file
2. target family file
3. common/portable file
4. package default target implementation
5. fail build
```

Example:

```txt
storage.web.js
storage.mobile.java
storage.common.js
```

For target `web`:

```txt
storage.web.js
```

For target `android`:

```txt
storage.mobile.java if family mobile is configured
else storage.common.js
else fail
```

---

# Template Resolution

Template selection:

```txt
1. template with exact target
2. target-polymorphic template
3. fail build
```

Example:

```nova
<template target <- web>
  ...
/|

<template>
  ...
/|
```

For `web`, exact target wins.
For `android`, polymorphic template is used if valid.

---

# Capability Compatibility

When target picks a capability implementation, compiler checks:

```txt
props compatibility
emitted event compatibility
external operation compatibility
type compatibility
permission compatibility
renderer primitive availability
```

Rule:

```txt
1. Required props cannot disappear.
2. Emitted events must preserve payload type.
3. External operation input/output must match contract.
4. Target implementation may add optional capability, but Nova code cannot rely on it unless declared.
```

---

# Build Graph

Build graph nodes:

```txt
SourceModule
PackageModule
CapabilityManifest
TypeContract
StateContract
EventContract
Template
Lifecycle
ExternalOperation
TargetAdapter
Permission
Artifact
```

Build graph edges:

```txt
imports
reads state
emits event
listens event
calls external
uses renderer primitive
requires permission
targets platform
```

This graph is the basis for diagnostics and security audit.

---

# Build Phases

Canonical build:

```txt
1. load manifest
2. resolve entry
3. resolve module/package graph
4. parse source
5. build symbol table
6. validate types and purity
7. resolve target capability
8. validate permissions
9. lower scheduler/view IR
10. generate target artifact
11. emit diagnostics and metadata
```

Compiler may cache phases if output remains equivalent.

---

# Diagnostics

Resolution diagnostics must include:

```txt
target
import source
requesting file
missing capability or symbol
candidate files considered
reason candidate was rejected
suggested fix when possible
```

Example diagnostic:

```txt
NVA-TARGET-004: @env/storage has no android implementation
  requested by src/App.nova
  target: android
  considered: storage.web.js, storage.common.js
```

---

# Reproducibility

Build should be reproducible.

Rule:

```txt
1. Package versions must be locked by project tooling.
2. Target capability manifest version must be known.
3. Resolver order must be deterministic.
4. Generated artifacts should include build metadata.
5. Build must not depend on filesystem glob order without sorting.
```

---

# Alternatives Considered

## Runtime Target Resolution

All capability selection happens at app startup.

Rejected because:

```txt
Missing target implementation would fail late.
Permissions cannot be audited fully at build time.
Generated artifacts need target-specific lowering.
```

## Single Universal Artifact

One artifact contains all targets.

Out of production v1 scope because:

```txt
Mobile/native targets need target-specific packaging.
Security permissions differ per platform.
Bundle size and diagnostics become worse.
```

## Platform Branches in Source

Source code writes conditions for platform.

Rejected because:

```txt
Platform logic would leak into pure core.
Target-specific templates and capability resolution are cleaner.
```

---

# Consequences

## Positive

```txt
1. Missing target support fails at build time.
2. Build output is deterministic.
3. Permission audit has graph context.
4. Renderer and external selection are explicit.
5. Diagnostics can explain why resolution failed.
```

## Negative

```txt
1. Build system must maintain target manifests.
2. Package graph needs lock/version metadata.
3. Target-specific source conventions must be documented.
4. Multi-target build runs more validation work.
```

---

# Final Position

Nova build and target resolution is:

```txt
manifest-driven
target-aware
deterministic
graph-validated
permission-checked
artifact-specific
```

Core rule:

```txt
The target is chosen at build time. The semantic core stays the same.
```
