Berikut ADR yang fokus ke **package distribution, registry, dan versioning** Nova.

---

# ADR-022: Package Distribution & Versioning

## Status

Implemented / Accepted

## Context

ADR-003 mendefinisikan package import.
ADR-012 menetapkan namespace package:

```txt
@nova/*
@env/*
@scope/package
@scope/package/path
```

Framework penuh membutuhkan cara mendistribusikan:

```txt
Nova source package
standard package
renderer primitive package
external adapter package
target runtime package
conformance fixture package
```

Package dapat membawa permission dan target implementation. Karena itu package distribution
juga bagian dari security model.

---

# Decision

Nova memakai package manifest dan lockfile.

Recommended files:

```txt
nova.package.toml
nova.lock
```

Project application tetap memakai:

```txt
nova.toml
```

Package official:

```txt
@nova/core
@nova/ui
@nova/forms
@nova/navigation
@nova/app
@env/storage
@env/network
```

Registry adalah distribution mechanism, bukan trust boundary otomatis.
Build tetap melakukan permission audit dan target compatibility check.

---

# Package Types

```txt
source-package
renderer-package
external-capability-package
target-runtime-package
adapter-package
tooling-package
conformance-package
```

Rule:

```txt
1. Package type harus dideklarasikan.
2. Package boleh punya lebih dari satu type jika manifest jelas.
3. External-capability package wajib menyatakan permission.
4. Target-runtime package wajib menyatakan ABI compatibility.
5. Renderer package wajib menyatakan primitive contracts.
```

---

# Package Manifest

Recommended package manifest:

```toml
[package]
name = "@scope/package"
version = "0.1.0"
type = ["source-package"]
language = ">=0.1.0"
abi = ">=0.1.0 <0.2.0"

[exports]
"." = "src/index.nova"
"Button" = "src/Button.nova"

[targets.web]
adapter = "platform/web/index.web.js"

[targets.android]
adapter = "platform/android/Index.android.kt"

[permissions]
storage.read = false
network.request = false
```

Exact syntax dapat berubah, tetapi semantic metadata wajib ada.

---

# Lockfile

`nova.lock` mencatat:

```txt
package name
resolved version
source registry/url
content hash
dependency graph
target adapter hashes
permission declarations
abi compatibility
```

Rule:

```txt
1. Production build harus memakai lockfile.
2. Lockfile changes should be reviewable.
3. Build fails if package content hash does not match lockfile.
4. Package permission changes must be visible as lockfile diff.
5. Target adapter implementation must be pinned.
```

---

# Versioning

Nova memakai SemVer untuk package.

Compatibility dimensions:

```txt
packageVersion
languageVersion
abiVersion
schedulerVersion
viewIrVersion
targetManifestVersion
runtimeVersion
```

Breaking changes:

```txt
remove exported symbol
change event payload type incompatibly
change required prop
change external operation input/output incompatibly
increase required permission
drop supported target
change scheduler semantic expectation
```

Non-breaking changes:

```txt
add optional prop
add new exported symbol
add target implementation
fix adapter bug without semantic change
improve diagnostics without changing diagnostic code contract
```

---

# Dependency Resolution

Resolution input:

```txt
project manifest
package manifests
lockfile
target id
target capability manifest
permission policy
```

Resolution output:

```txt
package graph
module graph
target implementation selection
permission source chain
diagnostics
```

Rule:

```txt
1. Package graph must be deterministic.
2. Version conflict must produce diagnostic with dependency chain.
3. Target-specific dependency may be selected only for active target.
4. Package cannot silently activate adapter for unsupported target.
5. Permission source chain must be inspectable.
```

---

# Registry Trust

Registry package can contain external code and adapters.

Security rule:

```txt
1. Package install does not grant permission.
2. Project manifest grants permission.
3. Build reports which package requested each permission.
4. Lockfile pins exact package content.
5. Official packages are still audited through the same mechanism.
```

Recommended audit output:

```txt
permission: network.request
requested_by:
  - @scope/api-client@1.2.0 operation fetchUser
used_by:
  - src/Profile.nova lifecycle after @load_profile
target:
  web adapter platform/web/client.web.js
  android adapter platform/android/Client.android.kt
```

---

# Standard Package Release

Official packages release together with framework compatibility metadata.

Rule:

```txt
1. @nova/* packages declare supported target matrix.
2. @env/* packages declare permission mapping per target.
3. Runtime packages declare conformance suite version.
4. A Nova release should publish a tested bill of materials for web and android.
```

Recommended BOM:

```txt
nova-language
nova-compiler
nova-cli
@nova/core
@nova/ui
@nova/forms
@nova/navigation
@env/storage
@env/network
nova-web-runtime
nova-android-runtime
```

---

# Consequences

Keuntungan:

```txt
package dan adapter dapat didistribusikan tanpa mengaburkan permission
web/android dependency dapat dipin secara deterministic
SemVer punya makna berdasarkan contract Nova
future target dapat menambahkan adapter package tanpa mengubah source app
```

Trade-off:

```txt
lockfile dan audit metadata menambah kompleksitas tooling
package yang membawa external adapter perlu review lebih ketat
compatibility matrix harus dirawat lintas runtime
```
