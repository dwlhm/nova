# ADR-026: Modular System Architecture

## Status

Accepted

## Context

Kontrak modular Nova sudah dipecah di beberapa ADR:

```txt
ADR-003  capability & module graph
ADR-007  platform capability model
ADR-009  external interop & @env/*
ADR-010  build & target resolution
ADR-012  project layout & package convention
ADR-015  standard package surface (@nova/*)
ADR-022  package distribution & versioning
```

Developer baru harus membaca banyak dokumen untuk memahami satu gambaran: **bagaimana file,
package, capability, platform adapter, dan target runtime menyusun sistem modular Nova**.

ADR ini menjadi **peta arsitektur modular** resmi. ADR rincian di atas tetap menjadi sumber
keputusan spesifik; ADR-026 tidak menggantikan mereka.

---

# Decision

Nova memakai model modular berlapis dengan unit utama berikut.

```txt
Capability module   -> 1 file .nova = 1 unit kompilasi + dependency
Package             -> distribusi & namespace (@nova/*, @env/*, @scope/*)
Platform adapter    -> implementasi target di platform/
Target runtime      -> artifact production (web JS, Android Java)
Project manifest    -> entry, target, permission di nova.toml
```

Prinsip yang mengikat semua lapisan:

```txt
1. Import tidak mengeksekusi side effect.
2. State dan event adalah scheduler-owned; capability tidak memanggil lifecycle secara manual.
3. Dirty operation hanya lewat lifecycle atau external operation yang dideklarasikan.
4. Core compiler tidak mengenal DOM, View, atau API OS secara langsung.
5. Production runtime membaca ABI, bukan source .nova.
```

---

# Layered Model

```txt
┌─────────────────────────────────────────────────────────────┐
│ Project (nova.toml, src/, platform/, tests/)                │
└───────────────────────────┬─────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────┐
│ Capability modules (.nova files)                          │
│  contract / func / template / lifecycle / import            │
└───────────────────────────┬─────────────────────────────────┘
                            │ import graph (acyclic)
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
┌───────────────┐   ┌───────────────┐   ┌───────────────────┐
│ @nova/*       │   │ local ./path  │   │ @env/* + external │
│ packages      │   │ capabilities  │   │ import contract   │
│ (renderer,    │   │               │   │                   │
│  framework)   │   │               │   │                   │
└───────┬───────┘   └───────┬───────┘   └─────────┬─────────┘
        │                   │                     │
        └───────────────────┼─────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ Build resolution (target-aware)                             │
│  module graph, template selection, external impl, permissions │
└───────────────────────────┬─────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ ABI contracts (serializable)                                │
│  ViewIR, scheduler model, routes, permissions, source map   │
└───────────────────────────┬─────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ Target runtime + platform adapters                          │
│  web: JS + DOM    android: Java + View    @env -> native API │
└─────────────────────────────────────────────────────────────┘
```

Lihat juga ADR-000 (production baseline) dan ADR-014 (framework runtime architecture).

---

# Unit 1: Capability Module

**Definisi:** satu file `.nova` adalah satu capability module.

**ADR rincian:** ADR-001, ADR-003.

Isi tipikal:

```txt
<import> / <import external>
<contract type | state | capability>
<func>
<template>
<lifecycle>
```

Aturan modular penting:

| Aturan | Makna |
| --- | --- |
| File = boundary | Validasi, manifest, dan diagnostics per file |
| Import = graph edge | Tidak ada eksekusi saat import |
| Lifecycle tidak diekspor | Hanya scheduler yang memanggil handler |
| Public top-level | type, state, capability, func, template, external op |

Dependency graph yang divalidasi:

```txt
module graph
type graph
state graph
event graph
lifecycle graph
external operation graph
template dependency graph
```

Graph harus acyclic pada production v1 (kecuali event cycle yang didiagnosis, bukan type cycle).

---

# Unit 2: Import & Namespace

Tiga bentuk import normal + satu external:

```nova
<import Counter from "./Counter.nova" /|
<import state count from "./Counter.nova" /|
<import event @increment from "./Counter.nova" /|

<import external storage from "@env/storage">
  operation set { ... }
/|
```

Namespace internal:

```txt
CapabilityRef = module_path + exported_name
```

Contoh:

```txt
./Counter.nova::count
@nova/ui::button
@env/storage::set
```

**ADR rincian:** ADR-003, ADR-009.

---

# Unit 3: Package System

Package adalah cara mendistribusikan capability dan kontrak lintas project.

Namespace:

```txt
@nova/*              standard framework & renderer primitives
@env/*               platform environment capabilities
@scope/package       third-party atau organisasi
./relative/path      local project module
```

Tipe package (ADR-022):

```txt
source-package
renderer-package
external-capability-package
target-runtime-package
adapter-package
tooling-package
conformance-package
```

File distribusi yang direkomendasikan:

```txt
nova.toml            application manifest (wajib)
nova.package.toml    package metadata (opsional, library)
nova.lock            pinned versions & content hash (production)
```

Package resmi production v1 (ADR-015):

```txt
@nova/core, @nova/ui, @nova/forms, @nova/navigation, @nova/app
@env/storage, @env/network, @env/clipboard, @env/notify, @env/device
```

**ADR rincian:** ADR-012, ADR-015, ADR-022.

---

# Unit 4: Platform Directory

Project application menyimpan adapter target di `platform/`:

```txt
my-app/
  platform/
    web/
      storage.web.js
    android/
      storage.android.java
```

Naming production:

```txt
<name>.web.js
<name>.android.java
```

Platform directory adalah **project-owned adapter glue**, bukan pengganti `@env/*`.
`@env/*` mendeklarasikan contract; `platform/` atau package adapter memenuhi implementasi
per target.

**ADR rincian:** ADR-007, ADR-012, ADR-009.

---

# Unit 5: Target Resolution

Build memilih implementasi modular per target:

```txt
Input:
  nova.toml
  entry capability path
  target id (web | android)
  semua .nova di module graph
  target manifest (@nova/web, @nova/android, ...)

Output:
  BuildPlan (template, modules, external ops, permissions)
  ViewIR + scheduler model + artifact files
```

Aturan seleksi template (ADR-010):

```txt
<template target <- web>     -> exact target
<template>                   -> polymorphic (semua target)
```

External operation memilih implementation row:

```txt
platform/web/storage.web.js
platform/android/storage.android.java
@env adapter path from package manifest
```

**ADR rincian:** ADR-010, ADR-008.

---

# Modular Composition Patterns

## A. Local capability split

Pisahkan domain per file:

```txt
src/App.nova          entry + root template
src/Counter.nova      state contract
src/components/...    reusable templates
```

## B. Standard package composition

Template memakai primitive dari `@nova/ui` tanpa mengimpor DOM/View:

```nova
<import Button from "@nova/ui/button" /|
```

Renderer target menurunkan primitive ke DOM atau Android View.

## C. Environment capability

Storage/network hanya lewat external import + lifecycle:

```nova
<import external storage from "@env/storage">
  operation set { ... }
/|
```

Permission muncul di build graph dan `permissions.json` artifact.

## D. Page / routing composition

State `route` + `<page>` adalah komposisi view modular (ADR-023), bukan module graph baru.
Routing metadata ikut ViewIR; matching target-neutral di `internal/routing`.

---

# User Capabilities With Low-Level Implementation (JS / Java)

User **boleh** menambah capability yang membawa implementasi native rendah di project mereka.
Nova memisahkan **kontrak** (di `.nova`) dari **implementasi** (di `platform/` atau package adapter).

## Pola yang didukung production hari ini

### 1. Kontrak di `.nova`, implementasi di `platform/`

Capability module user mendeklarasikan operasi dirty lewat `<import external>`:

```nova
<import external storage from "@env/storage">
  operation set {
    input { key: string; value: unknown; }
    output void;
  }
/|

<lifecycle after @save>
  storage.set key <- "draft" value <- payload;
/|
```

Implementasi rendah ditulis di project (bukan di compiler):

```txt
my-app/platform/web/storage.web.js      # dipanggil saat build --target web
my-app/platform/android/storage.android.java   # dipanggil saat build --target android
```

Build resolver memetakan `@env/storage` → path `platform/<target>/storage.<ext>` (lihat
`internal/build/targets.go`). User **mengganti atau menambah** file di path itu dengan JS/Java
milik project; Nova tetap memvalidasi kontrak dan permission dari deklarasi `.nova`.

Ini adalah cara utama menambah perilaku platform tanpa mengubah compiler.

### 2. Beberapa capability `.nova` + satu adapter per target

User boleh memecah domain:

```txt
src/Vault.nova           # state + lifecycle + import external
src/App.nova             # template + memicu event
platform/web/vault.web.js
platform/android/Vault.android.java
```

Asalkan `import external ... from "@env/..."` cocok dengan operasi yang terdaftar di target
manifest, dan file `platform/` mengikuti konvensi path target.

### 3. Renderer / UI tetap terpisah

Primitive UI (`<button>`, `<text>`, …) tidak memerlukan JS/Java per widget di user project.
User menurunkan UI lewat template + `@nova/ui`; hanya **external operation** dan lifecycle
yang menyentuh `platform/` atau `@env` adapter.

### 4. Primitive extension (gap `@nova/ui`)

Untuk tag/kind di luar `@nova/ui`, definisi utamanya hidup di **`renderer-package`**
(`nova.package.toml` + `platform/` dalam paket). Aplikasi hanya mengaktifkan lewat
`renderer.extensions.packages` di `nova.toml` (nama paket + lockfile). Entri inline
`[[renderer.dictionary]]` di app hanya escape hatch lokal.

**ADR rincian:** [ADR-027](adr_027_renderer_primitive_extension_dictionary.md).

## Pola yang diizinkan bahasa (ADR-009), registrasi manifest masih terbatas

Source external boleh berupa path relatif:

```nova
<import external analytics from "./platform/web/analytics.web.js">
  operation track {
    input { name: string; payload: unknown; }
    output void;
  }
/|
```

Agar build sukses, `analytics` harus terdaftar di **target capability manifest** dengan
`Source` yang sama (`./platform/web/analytics.web.js` atau alias yang disepakati) dan daftar
operasi + permission. Hari ini manifest target resmi di-hardcode untuk `@env/*` di
`internal/build/targets.go`; registrasi capability custom per project lewat `nova.toml`
masih perlu perluasan implementasi.

Sampai registrasi itu ada, gunakan `@env/*` yang sudah terdaftar dan override file di
`platform/`, atau kontribusi penambahan entry manifest untuk capability baru di toolchain.

## Aturan implementasi JS / Java

| Aturan | Alasan |
| --- | --- |
| Hanya dipanggil dari lifecycle | ADR-005, ADR-009 |
| Input/output harus Nova data serializable | Tidak boleh mengembalikan DOM node, Activity, Promise ke scheduler |
| Permission dideklarasikan di `nova.toml` | ADR-013 |
| Satu operasi = satu bridge ke adapter | Completion lewat event `@...` / `@..._failed` |
| Web: modul JS di artifact / runtime load | Lihat ADR-016 |
| Android: binding lewat `NovaExternalBindings` + kelas Java | Lihat ADR-017 |

Contoh bentuk adapter (konseptual):

```javascript
// platform/web/storage.web.js
export function set(key, value) { /* localStorage, IndexedDB, … */ }
```

```java
// platform/android/storage.android.java
public final class Storage {
  public static void set(String key, Object value) { /* SharedPreferences, … */ }
}
```

Detail signature bridge mengikuti generator artifact target; user mengisi tubuh fungsi native.

## Ringkas untuk user

```txt
Ya — user bisa menambah capability dengan implementasi low-level JS/Java.
Cara praktis: deklarasikan external di .nova, tulis platform/web|android/*.{js,java} di project.
Capability wholly-new (bukan @env) butuh manifest entry; kontrak bahasa sudah siap di ADR-009.
```

**ADR rincian:** ADR-009 (interop), ADR-012 (`platform/`), ADR-013 (permission), ADR-016/017 (bridge).

---

# Build-Time vs Runtime

| Fase | Modular concern |
| --- | --- |
| Parse/validate | Per-file capability, import graph, type/state rules |
| Resolve | Target template, package adapter path, permissions |
| Artifact | Bundle IR, ViewIR, platform files, `@env` binding table |
| Runtime | Scheduler queue, renderer, adapter invocation — **tanpa** re-parse `.nova` |

Go packages `internal/packages` dan `packages.Resolve` mengimplementasikan kontrak distribusi
(ADR-022). Integrasi penuh ke CLI build adalah langkah lanjutan; graph lokal dari `src/` sudah
diresolve lewat `internal/build`.

---

# Diagnostics & Conformance

Modular errors memakai prefix ADR-011, contoh:

```txt
NVA-PKG-*       package resolution
NVA-LAYOUT-*    project layout
NVA-TARGET-*    target / renderer
NVA-SEMANTIC-*  cross-module semantics
```

Conformance memvalidasi bahwa komposisi modular menghasilkan artifact dan scheduler trace yang
konsisten (ADR-020). Fixture `nova.conformance.json` per example/fixture.

---

# Relationship To Other ADRs

| Topik | ADR |
| --- | --- |
| Bahasa & capability boundary | 001 |
| Scheduler & event ownership | 002 |
| Module graph detail | 003 |
| Lifecycle & effects | 005, 009 |
| View/template lowering | 006, 008 |
| Platform categories | 007 |
| Build resolution | 010 |
| Project folders | 012 |
| Permissions | 013 |
| Framework layers | 014 |
| Standard packages | 015 |
| Package lock/version | 022 |
| Page routing composition | 023 |
| Production baseline | 000 |

---

# Non Goals

```txt
Nova tidak menjadi general-purpose module system seperti npm atau Go modules penuh
Tidak ada dynamic plugin loading dari registry publik pada production v1
Tidak ada visibility modifier private/export eksplisit pada production v1
Tidak mengganti ADR-003/012/022 — ADR-026 hanya merangkum
```

---

# Consequences

Keuntungan:

```txt
Satu dokumen untuk onboarding arsitektur modular
Relasi antar capability, package, platform, dan target menjadi eksplisit
Memudahkan review fitur baru: "unit modular mana yang terdampak?"
```

Trade-off:

```txt
Beberapa detail tetap harus dibaca di ADR rincian
Perubahan kontrak modular harus update ADR-026 ringkasan bila diagram/alur berubah
```
