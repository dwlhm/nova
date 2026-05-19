Berikut ADR yang fokus ke **testing dan conformance strategy** Nova.

---

# ADR-020: Testing & Conformance Strategy

## Status

Implemented / Accepted

## Context

Nova membutuhkan beberapa runtime:

```txt
Go reference/conformance prototype
Web TypeScript/JavaScript runtime
Android Kotlin runtime
future iOS/Desktop runtime
```

Jika setiap runtime hanya diuji secara lokal, semantic Nova akan drift.

Conformance perlu memastikan:

```txt
event ordering sama
state commit sama
type/permission diagnostics sama
ViewIR event route sama
external completion semantics sama
target artifact metadata kompatibel
```

---

# Decision

Nova memakai conformance suite resmi berbasis fixture dan expected trace.

Conformance suite dibagi menjadi:

```txt
language conformance
semantic conformance
scheduler conformance
view conformance
target manifest conformance
external adapter conformance
security conformance
artifact conformance
```

Runtime target Web dan Android wajib lulus subset MVP sebelum dianggap supported.

---

# Conformance Artifacts

Fixture conformance berisi:

```txt
source files
nova.toml
target manifest
input events
external operation stubs
expected diagnostics
expected scheduler trace
expected state snapshots
expected view metadata
expected artifact metadata
```

Recommended layout:

```txt
tests/conformance/
  language/
  scheduler/
  view/
  target/
  security/
  external/
  artifacts/
```

Fixture tidak boleh bergantung pada wall-clock, random, network nyata, atau platform resource nyata.

---

# Scheduler Trace

Scheduler conformance memakai trace data.

```txt
Trace {
  events: EventEnvelope[]
  commits: StateCommit[]
  lifecycleCalls: LifecycleTrace[]
  externalCalls: ExternalCallTrace[]
  errors: SchedulerError[]
}
```

Rule:

```txt
1. LogicalSequence harus match.
2. Commit batch harus match.
3. Completion event harus muncul sebagai event baru.
4. Transition failure harus membatalkan commit event aktif.
5. Lifecycle output partial tidak boleh terlihat jika handler gagal sebelum return.
```

---

# Diagnostic Conformance

Diagnostic expected mencakup:

```txt
code
severity
message pattern
file
span
related spans
hint
```

Rule:

```txt
1. Diagnostic code harus stabil.
2. Message boleh berubah selama code dan expected/actual tetap jelas.
3. Span harus deterministic.
4. Ordering diagnostic harus deterministic atau disortir oleh stable key.
```

---

# View Conformance

View conformance memvalidasi:

```txt
ViewIR shape
binding dependency metadata
event route table
list key warnings
target template selection
accessibility metadata
```

Renderer platform tidak harus pixel-identical.

Target visual smoke test boleh memakai screenshot, tetapi conformance semantic memakai IR/metadata.

---

# Target Conformance

Target Web dan Android wajib menyediakan test harness.

Web harness:

```txt
headless browser or DOM test environment
runtime event injection
DOM event route verification
permission denial stub
external operation stub
hydration mismatch fixture
```

Android harness:

```txt
JVM unit tests for scheduler/runtime
instrumented tests for Compose renderer and permission bridge
fake lifecycle owner
fake external adapter
snapshot restoration fixture
```

Rule:

```txt
1. Harness boleh native, expected trace tetap sama.
2. Platform tests tidak boleh mengganti semantic expected.
3. Adapter-specific behavior harus ditandai sebagai target fixture.
```

---

# External Adapter Tests

External adapter contract tests memvalidasi:

```txt
input marshalling
output validation
permission enforcement
error mapping
sync/async completion ordering
forbidden native handle rejection
```

Stubs harus bisa mengembalikan:

```txt
success
typed domain failure
permission denial
adapter exception
invalid output
timeout/cancelled
```

---

# Security Tests

Security conformance memvalidasi:

```txt
dirty operation rejected in pure zone
missing permission rejected at build
undeclared event rejected before enqueue
invalid payload rejected before transition
external output with native handle rejected
package permission surfaced to project
```

Security tests harus dijalankan untuk:

```txt
compiler
web runtime
android runtime
target package manifests
```

---

# Compatibility Matrix

Setiap release Nova mencatat matrix:

```txt
languageVersion
abiVersion
schedulerVersion
viewIrVersion
targetManifestVersion
runtimeVersion
standardPackageVersion
conformanceSuiteVersion
```

Target support status:

```txt
experimental
preview
supported
deprecated
removed
```

MVP target awal:

```txt
web      -> preview once CSR + storage/network + DOM renderer pass
android  -> preview once Compose renderer + storage/network + lifecycle pass
```

---

# CI Policy

Official repository CI should run:

```txt
go test ./...
compiler fixture tests
conformance expected trace generation check
web runtime tests
android runtime unit tests
artifact metadata validation
```

Target-specific slow/instrumented tests may run in nightly CI, but release cannot be marked
supported without passing them.

---

# Consequences

Keuntungan:

```txt
runtime web dan android tidak drift dari semantic core
future target punya jalur validasi jelas
diagnostic dan artifact tetap stabil
security regression lebih mudah ditangkap
```

Trade-off:

```txt
conformance suite perlu dirawat seperti public API
target adapter butuh harness native
beberapa fitur tidak boleh dianggap selesai sampai punya fixture lintas target
```
