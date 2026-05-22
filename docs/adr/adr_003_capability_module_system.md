Berikut ADR yang fokus ke **capability dan module system** Nova.

---

# ADR-003: Capability & Module System

## Status

Implemented / Accepted

## Context

ADR-001 menetapkan bahwa setiap file `.nova` adalah **capability boundary**.
Namun model module perlu menjawab hal berikut:

```txt
apa yang dianggap sebagai capability
apa yang dapat di-import
bagaimana namespace dibentuk
kapan import mengeksekusi sesuatu
bagaimana dependency graph divalidasi
bagaimana capability lintas platform dipilih
```

Nova membutuhkan module system yang cukup eksplisit untuk compiler dan runtime,
tetapi tetap kecil agar surface language tidak berubah menjadi general-purpose module language.

---

# Decision

Nova menggunakan model:

```txt
file .nova = capability module
import = deklarasi dependency graph
top-level construct = public symbol
lifecycle = runtime handler, bukan callable symbol
external import = dirty capability contract
```

Import tidak pernah menjalankan side effect.

Side effect hanya dapat terjadi ketika scheduler menjalankan `<lifecycle>` atau ketika target adapter
menjalankan operation dari `<import external>`.

---

# Capability Unit

Capability adalah unit kompilasi dan unit dependency.

Satu file `.nova` dapat berisi:

```txt
<import>
<import external>
<contract type>
<contract state>
<contract capability>
<func>
<template>
<lifecycle>
```

File tidak wajib berisi semua construct.

Contoh capability:

```txt
Counter.nova
CounterStorage.nova
Button.nova
AudioLab.nova
```

Capability identity berasal dari resolved module path.

```txt
./Counter.nova       -> local capability
@nova/ui/Button      -> package capability
@env/storage         -> platform external capability
```

---

# Public Surface

Top-level construct berikut menghasilkan symbol publik:

```txt
contract type
contract state
contract capability
func
template target
external operation contract
```

Lifecycle tidak diekspor sebagai callable symbol.

Reason:

```txt
Lifecycle adalah scheduler-owned runtime handler.
Lifecycle tidak boleh dipanggil manual oleh capability lain.
Dependency lifecycle terjadi lewat event/state import, bukan function call.
```

---

# Import Forms

Nova mendukung tiga import normal:

```nova
<import Counter from "./Counter.nova" /|
<import state count from "./Counter.nova" /|
<import event @increment from "./Counter.nova" /|
```

Dan satu import external:

```nova
<import external storage from "@env/storage">
  operation set {
    input {
      key: string;
      value: unknown;
    }

    output void;
  }
/|
```

Makna:

```txt
import capability -> menambahkan capability ke dependency graph
import state      -> memberi akses baca state scheduler-owned
import event      -> memberi hak listen/emit event
import external   -> mendeklarasikan dirty operation boundary
```

Import tidak melakukan mount capability secara langsung.
Scheduler dan build graph yang menentukan capability instance aktif.

---

# Aliasing

Import boleh memakai alias untuk menghindari konflik nama.

```nova
<import state count as cartCount from "./Cart.nova" /|
<import event @submit as @cart_submit from "./Cart.nova" /|
```

Rule:

```txt
1. Alias hanya berlaku di file pengimpor.
2. Alias tidak mengubah nama public symbol di capability asal.
3. Alias event tetap harus memakai prefix @.
4. Compiler menolak dua symbol lokal dengan nama efektif yang sama.
```

---

# Namespace

Nova memakai namespace berbasis module path.

```txt
CapabilityRef = package_or_path + exported_name
```

Contoh:

```txt
./Counter.nova::Counter
./Counter.nova::count
@nova/ui::Button
@env/storage::set
```

Surface syntax tidak wajib menulis `::` untuk pemakaian umum.
Compiler menggunakan namespace internal untuk mencegah collision.

---

# Dependency Graph

Compiler membangun graph berikut:

```txt
module graph
type graph
state graph
event graph
lifecycle graph
external operation graph
template dependency graph
```

Rule (production v1):

```txt
1. Runtime module graph harus acyclic.
2. Type reference lintas file juga harus acyclic pada production v1.
3. Recursive type dalam file yang sama ditunda sampai type checker mendukungnya.
4. Lifecycle dependency tidak boleh menciptakan mount/dispose loop.
5. Event emission cycle boleh ada karena scheduler queue memproses event secara FIFO.
6. Event cycle harus dapat didiagnosis oleh tooling sebagai potensi infinite loop.
```

Event cycle tidak ditolak secara default karena workflow seperti retry atau polling membutuhkan loop.
Namun compiler boleh memberi warning jika loop tidak memiliki guard yang terlihat.

---

# Capability Manifest

Setiap capability yang lolos build menghasilkan manifest internal.

```txt
CapabilityManifest {
  ref: CapabilityRef
  imports: ImportRef[]
  states: StateRef[]
  events: EventRef[]
  types: TypeRef[]
  funcs: FuncRef[]
  templates: TemplateRef[]
  lifecycles: LifecycleRef[]
  externalOperations: ExternalOperationRef[]
  targetConstraints: TargetConstraint[]
}
```

Manifest dipakai oleh:

```txt
compiler validation
scheduler graph construction
renderer lowering
target resolution
diagnostics
security audit
```

---

# Visibility Rules

Nova tidak memiliki modifier visibility pada production v1.

Rule:

```txt
1. Semua top-level named construct adalah public dalam module graph.
2. Body function, transition, template, dan lifecycle tetap private implementation.
3. Lifecycle tidak dapat di-import.
4. External operation hanya dapat dipanggil oleh file yang meng-import external capability tersebut.
5. State dari file lain harus di-import eksplisit dengan `import state`.
6. Event dari file lain harus di-import eksplisit dengan `import event`.
```

Visibility yang lebih kompleks seperti `private` atau `export` eksplisit ditunda.

---

# Package Imports

Package import memakai prefix `@`.

```nova
<import Button from "@nova/ui/Button" /|
<import external storage from "@env/storage" /|
```

Kategori package:

```txt
@nova/*     standard Nova package
@env/*      target environment capability
@scope/*    third-party package
./* ../*    local project module
```

Resolver tidak boleh menganggap `@env/*` sebagai package normal.
`@env/*` selalu diselesaikan oleh target adapter atau platform capability registry.

---

# Alternatives Considered

## Import Executes Module

Model seperti beberapa bahasa scripting, ketika import menjalankan body module.

Rejected because:

```txt
Import side effect akan melanggar lifecycle-bounded effects.
Urutan import dapat memengaruhi state.
Build graph menjadi platform-dependent.
```

## Explicit Export List

Setiap file harus menulis export list.

Out of production v1 scope because:

```txt
Surface language menjadi lebih besar.
Belum ada kebutuhan visibility yang kuat.
Compiler sudah dapat membangun symbol graph dari top-level construct.
```

## Class-Like Component Model

Capability diperlakukan seperti class dengan constructor dan methods.

Rejected because:

```txt
Nova ingin state scheduler-owned, bukan object-owned.
Lifecycle tidak boleh menjadi method callable.
Template tetap deklaratif, bukan instance method.
```

---

# Consequences

## Positive

```txt
1. Module graph eksplisit dan dapat divalidasi statis.
2. Import tidak memiliki side effect.
3. Capability boundary konsisten dengan file .nova.
4. Lifecycle tetap scheduler-owned.
5. State dan event lintas capability harus diberi hak akses eksplisit.
6. Target resolution dapat bekerja dari manifest capability.
```

## Negative

```txt
1. Cross-file cycle ditolak pada production v1.
2. Semua top-level construct public sampai visibility modifier ditambahkan.
3. Tooling perlu menampilkan graph agar dependency mudah dipahami.
4. Event cycle butuh warning dan debugger queue yang baik.
```

---

# Final Position

Nova module system adalah:

```txt
capability-first
graph-driven
side-effect-free on import
explicit for state/event access
target-aware through manifest
```

Core rule:

```txt
Import declares capability dependency.
Scheduler decides runtime execution.
```
