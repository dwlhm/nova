Berikut ADR yang fokus ke **arsitektur framework dan kontrak runtime** Nova.

---

# ADR-014: Framework Runtime Architecture

## Status

Implemented / Accepted

## Context

ADR-001 sampai ADR-013 mendefinisikan bahasa, scheduler, module, type system, template,
lowering, target resolution, error, layout, dan security.

Namun Nova sebagai framework multiplatform penuh masih membutuhkan keputusan yang mengikat:

```txt
lapisan framework
kontrak antar lapisan
runtime target awal
artifact yang dihasilkan
batas tanggung jawab compiler, runtime, adapter, dan package
```

Target awal Nova adalah:

```txt
web
android
```

Target future:

```txt
ios
desktop
future target
```

Nova perlu tetap target-neutral pada semantic core, tetapi tetap menghasilkan aplikasi nyata
yang native terhadap platform awal.

---

# Decision

Nova memakai arsitektur framework berlapis:

```txt
Nova source
  -> compiler frontend
  -> semantic core
  -> framework IR contracts
  -> target runtime package
  -> target adapter
  -> platform artifact
```

Lapisan resmi Nova:

```txt
1. language core
2. compiler and semantic model
3. scheduler semantics
4. framework ABI contracts
5. standard package surface
6. target runtime
7. platform adapter
8. artifact builder
9. conformance suite
10. developer tooling
```

Implementasi Go di repository tetap menjadi **reference/conformance prototype** untuk semantic
dan tooling awal. Aplikasi produksi target awal memakai runtime native:

```txt
web      -> TypeScript/JavaScript runtime
android  -> Kotlin/JVM Android runtime
```

Runtime native wajib mengikuti scheduler semantics, view IR semantics, permission model,
dan diagnostic contract yang sama.

---

# Framework Units

Nova framework terdiri dari unit berikut:

```txt
nova-compiler
nova-semantic-core
nova-scheduler-spec
nova-view-ir
nova-target-manifest
nova-standard-packages
nova-web-runtime
nova-android-runtime
nova-conformance
nova-cli
```

Makna:

```txt
nova-compiler              parse, validate, type check, lower
nova-semantic-core         pure semantic model and diagnostics
nova-scheduler-spec        canonical event and commit semantics
nova-view-ir               renderer-neutral UI contract
nova-target-manifest       target capability and permission metadata
nova-standard-packages     @nova/* and @env/* contracts
nova-web-runtime           browser runtime and DOM adapter
nova-android-runtime       Android runtime and renderer adapter
nova-conformance           cross-runtime test suite
nova-cli                   build, dev, test, package, inspect
```

---

# Framework ABI Contracts

Semua target harus memahami kontrak ABI berikut.

```txt
SemanticModel
SchedulerIR
ViewIR
TargetIR
ExternalOperationTable
PermissionTable
EventRouteTable
DiagnosticStream
SourceMap
HydrationSnapshot
TargetCapabilityManifest
```

Rule:

```txt
1. ABI adalah data contract, bukan object native.
2. ABI harus serializable untuk tooling, tests, dan devtools.
3. Target runtime boleh memakai format internal yang lebih efisien setelah validasi ABI.
4. Perubahan ABI harus versioned dan diuji dengan conformance.
5. Runtime target tidak boleh membaca source .nova langsung pada production artifact.
```

---

# Responsibility Boundaries

## Compiler

Compiler bertanggung jawab atas:

```txt
lexing
parsing
semantic graph
type checking
purity validation
target resolution
permission audit
IR generation
source map generation
diagnostic generation
```

Compiler tidak menjalankan:

```txt
lifecycle userland
external operation
platform adapter
runtime scheduler queue aplikasi
```

## Semantic Core

Semantic core bertanggung jawab atas fungsi pure:

```txt
state transition planning
template projection
dependency metadata derivation
contract compatibility check
permission requirement derivation
```

## Target Runtime

Target runtime bertanggung jawab atas:

```txt
logical scheduler queue
state cell storage
lifecycle execution shell
event envelope validation
renderer port invocation
external operation invocation
runtime error routing
devtools bridge
```

## Platform Adapter

Platform adapter bertanggung jawab atas:

```txt
DOM / Android event input
native rendering output
native storage/network/device API
permission prompt/check
native lifecycle mapping
artifact integration
```

---

# Initial Target Contract

Target awal wajib menyediakan:

```txt
web:
  host adapter
  DOM renderer adapter
  external JS adapter bridge
  browser permission mapping
  dev server integration
  CSR artifact
  optional SSR/hydration path

android:
  host adapter
  Compose renderer adapter
  external Kotlin adapter bridge
  Android permission mapping
  Gradle artifact integration
  activity/process lifecycle mapping
  state restoration hook
```

Target dianggap usable jika bisa:

```txt
1. build entry .nova yang sama untuk web dan android
2. mount root capability
3. render template polymorphic atau target-specific
4. route user input menjadi scheduler event
5. commit state secara atomic
6. menjalankan lifecycle dirty boundary
7. memanggil external operation sesuai permission
8. melaporkan diagnostic/source map yang konsisten
9. lulus conformance suite target MVP
```

---

# Framework Execution Flow

Canonical production flow:

```txt
nova build --target web
  -> read nova.toml
  -> resolve project and packages
  -> compile semantic model
  -> audit permissions
  -> produce web artifact
  -> bundle web runtime + target IR + adapters

nova build --target android
  -> read nova.toml
  -> resolve project and packages
  -> compile semantic model
  -> audit permissions
  -> produce Android artifact
  -> generate Gradle/Kotlin integration + target IR + adapters
```

Runtime flow:

```txt
host/platform event
  -> adapter validates route
  -> scheduler enqueue
  -> pure transition plan
  -> atomic commit
  -> view invalidation
  -> renderer update
  -> lifecycle shell
  -> external operation
  -> completion event
```

---

# Artifact Boundary

Build artifact harus menyertakan:

```txt
runtime package reference or bundled runtime
compiled scheduler IR
compiled view/target IR
event route table
external operation table
permission table
source map for diagnostics
framework ABI version
target manifest version
```

Artifact tidak boleh menyertakan:

```txt
unvalidated source graph as runtime source of truth
platform permission not declared by manifest
external operation not present in build graph
native handle serialized into Nova data
```

---

# Versioning

Framework runtime compatibility memakai versi berikut:

```txt
languageVersion
abiVersion
schedulerVersion
viewIrVersion
targetManifestVersion
standardPackageVersion
runtimeVersion
```

Rule:

```txt
1. Compiler menolak runtime yang tidak kompatibel dengan ABI artifact.
2. Patch runtime boleh memperbaiki adapter tanpa mengubah semantic.
3. Perubahan semantic scheduler membutuhkan schedulerVersion baru.
4. Package @nova/* menyatakan minimal languageVersion dan abiVersion.
5. Conformance result harus mencatat seluruh versi di atas.
```

---

# Consequences

Keuntungan:

```txt
semantic core tetap kecil
target runtime bisa native
web dan android dapat berbagi contract yang sama
tooling dapat membaca artifact tanpa menjalankan platform
future target dapat ditambahkan melalui adapter + conformance
```

Trade-off:

```txt
ABI harus dijaga dengan disiplin
runtime web dan android perlu implementasi conformance masing-masing
fitur framework baru harus masuk melalui contract, bukan shortcut platform
```
