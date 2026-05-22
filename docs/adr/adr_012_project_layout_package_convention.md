Berikut ADR yang fokus ke **project layout dan package convention** Nova.

---

# ADR-012: Project Layout & Package Convention

## Status

Implemented / Accepted

## Context

Nova membutuhkan konvensi project agar:

```txt
entry point jelas
target build konsisten
external implementation mudah ditemukan
package import stabil
docs dan tests mudah dipahami
```

ADR ini tidak mengunci semua tooling, tetapi memberi layout default yang harus didukung oleh
compiler/build tool resmi.

---

# Decision

Nova project memakai layout default:

```txt
nova.toml
src/
platform/
tests/
docs/
examples/
```

Entry point dideklarasikan di `nova.toml`.

File `.nova` adalah capability module.

---

# Recommended Layout

```txt
my-app/
  nova.toml
  src/
    App.nova
    Counter.nova
    components/
      Button.nova
    domain/
      User.nova
  platform/
    web/
      storage.web.js
    android/
      storage.android.kt
    ios/
      storage.ios.swift
    desktop/
      storage.desktop.rs
  tests/
    scheduler/
    parser/
    fixtures/
  docs/
    adr/
  examples/
```

Only `nova.toml` and entry source are required.
Other directories are conventions.

---

# Manifest

Default manifest:

```txt
nova.toml
```

Recommended sections:

```toml
[project]
name = "my-app"
version = "0.1.0"
entry = "src/App.nova"

[targets.web]
renderer = "@nova/web"

[targets.android]
renderer = "@nova/android"   # production Java native View

[permissions]
storage.read = true
storage.write = true
```

Manifest should describe:

```txt
project metadata
entry capability
target configuration
dependencies
permissions
build options
```

---

# Source Directory

`src/` contains Nova capability files.

Naming convention:

```txt
PascalCase.nova for capability/component files
camelCase for state and function names
PascalCase for contract type names
@snake_case or @verb_noun for scheduler events
```

Examples:

```txt
AudioLab.nova
Counter.nova
UserProfile.nova
```

```nova
<contract type UserProfile>
  name: string;
/|

<contract state Counter>
  count: number <- 0 {
    @increment -> count + 1;
  };
/|
```

---

# Platform Directory

`platform/` contains target-specific external implementations and adapter glue owned by the project.

Recommended:

```txt
platform/web/
platform/android/
platform/ios/
platform/desktop/
platform/common/
```

Target-specific external files:

```txt
storage.web.js
storage.android.kt
storage.ios.swift
storage.desktop.rs
storage.common.js
```

Resolver may support package-specific conventions, but project-local convention should stay simple.

---

# Tests Directory

`tests/` may contain:

```txt
parser fixtures
validator fixtures
scheduler conformance tests
target build smoke tests
template rendering snapshots
external adapter contract tests
```

Recommended layout:

```txt
tests/
  fixtures/
    valid/
    invalid/
  scheduler/
  renderer/
  target/
```

Test format is tooling-specific and not locked by this ADR.

---

# Docs Directory

`docs/` contains project documentation.

Recommended:

```txt
docs/
  adr/
  examples/
  target-notes/
```

ADR files use:

```txt
docs/adr/adr_001_language_specification.md
```

Naming convention:

```txt
adr_NNN_slug.md
```

---

# Package Naming

Package import forms:

```txt
@nova/*
@env/*
@scope/package
@scope/package/path
```

Reserved:

```txt
@nova/*  official standard Nova packages
@env/*   target environment capabilities
```

Third-party package should use scoped namespace:

```txt
@company/ui
@dwlhm/audio
```

Unscoped package names are discouraged for public packages.

---

# Package Manifest

Package manifest should declare:

```txt
name
version
exports
targets
renderer primitives
external capabilities
permissions
Nova language version
```

Example shape:

```toml
[package]
name = "@company/ui"
version = "0.1.0"
nova = "0.1"

[exports]
Button = "src/Button.nova"

[targets.web]
adapter = "platform/web"
```

Exact package manifest format may share `nova.toml` with projects.

---

# File Naming

Recommended:

```txt
Capability files       PascalCase.nova
External implementation camelCase.target.ext
Fixtures               snake_case.nova
ADR                    adr_NNN_slug.md
Generated files        inside build output directory
```

Generated files should not be written into `src/` by default.

---

# Build Output

Default build output directory:

```txt
build/
```

Recommended target outputs:

```txt
build/web/
build/android/
build/ios/
build/desktop/
```

Build output should contain metadata:

```txt
target id
Nova compiler version
package lock digest
capability manifest digest
diagnostic summary
```

---

# Versioning

Project/package version follows semver convention.

Compatibility-relevant changes:

```txt
contract type change
contract capability props/emits change
external operation input/output change
event payload change
renderer primitive contract change
permission requirement change
```

Package authors should treat breaking contract changes as major version changes.

---

# Alternatives Considered

## No Standard Layout

Allow every project to define arbitrary layout.

Rejected because:

```txt
Tooling and examples become inconsistent.
Target resolution needs predictable defaults.
New users need a stable convention.
```

## Framework-Generated Source Layout Only

All projects must use a strict generated structure.

Rejected because:

```txt
Nova should allow small projects.
Libraries and apps have different needs.
Only a small set of defaults should be required.
```

## Platform Files Beside Nova Files Only

Target implementation files must live next to `.nova` files.

Rejected because:

```txt
Project may want to separate platform code for review/security.
Package target adapters often have their own directory structure.
```

---

# Consequences

## Positive

```txt
1. Projects have predictable entry and target configuration.
2. Tooling can discover source, docs, tests, and platform code.
3. Package imports have clear reserved namespaces.
4. ADR and generated output naming is consistent.
5. Security audit can locate platform code more easily.
```

## Negative

```txt
1. Some projects may need custom layout configuration.
2. Package manifest format still needs final tooling spec.
3. More conventions to document.
4. Migration tooling may be needed if conventions evolve.
```

---

# Final Position

Nova project convention is:

```txt
manifest-led
src for Nova capabilities
platform for target code
tests for conformance/fixtures
docs for ADR and notes
scoped package imports
```

Core rule:

```txt
The manifest names the app. The source graph defines the app. Target directories adapt the app.
```
