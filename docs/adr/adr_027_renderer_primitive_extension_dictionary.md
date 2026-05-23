# ADR-027: Renderer Primitive Extension Dictionary

## Status

Accepted (design contract)

Implementasi toolchain: **belum** — lowering production v1 masih memakai tabel built-in di
`internal/artifact` + fallback generik.

## Context

`@nova/ui` mendefinisikan surface primitive production v1 (`text`, `button`, `surface`, …) di
ADR-015. Template Nova tidak mengenal DOM atau Android View; compiler memproyeksikan markup ke
**ViewIR** (`kind`, `props`, `events`, `children`, bindings), lalu renderer target **menerjemahkan**
node itu ke artifact platform (ADR-006, ADR-008).

User membutuhkan jalur resmi untuk hal yang **belum** ditangani `@nova/ui` — tanpa fork compiler dan
**tanpa** menduplikasi definisi per target di `nova.toml` aplikasi. Pola mental:

```txt
@nova/ui              = dictionary standar (built-in)
renderer-package      = library ter-pack: dictionary + adapter web/android dalam satu unit
nova.toml (app)       = hanya menyebut nama paket + versi (dan policy unknown_kind)
internal/artifact     = merge dictionary ter-resolve -> artifact (translate)
```

Perbedaan dari `@env/*` (ADR-009):

| | `@env/*` external capability | `renderer-package` |
| --- | --- | --- |
| Kontrak | `operation` async + lifecycle | primitive `kind` + props/events (ViewIR) |
| Implementasi | `platform/` per operasi | `platform/` registrasi primitive (satu entry per target) |
| Aktivasi di app | `import external` + permission | **paket di dependency graph** + kind di template |
| Pack | `external-capability-package` | `renderer-package` (ADR-022) |

Renderer extension **bukan** capability component dengan state contract terpisah — itu tetap jalur
`<contract capability>`. Extension package untuk **tag primitive** (`<sparkline />`) yang bisa
dibagikan antar project lewat registry/lockfile.

---

# Decision

## 1. Unit distribusi = renderer-package (primary)

Primitive di luar `@nova/ui` didistribusikan sebagai **`renderer-package`** (ADR-022), dengan:

```txt
nova.package.toml   manifest: kinds, schema props/events, target adapters, abi
platform/web/...    registrasi lowering web (satu modul atau per-kind)
platform/android/... registrasi lowering android
nova.lock           pin hash + permission (jika ada) + adapter hash
```

Aplikasi **hanya** mengaktifkan paket lewat `nova.toml` (nama + versi/constraint), tidak
mendefinisikan ulang path adapter per kind di manifest app.

## 2. `nova.toml` aplikasi = referensi paket, bukan dictionary inline

Kontrak app-level:

```toml
[renderer]
unknown_kind = "error"   # error | warn | passthrough_web

[renderer.extensions]
packages = [
  "@acme/charts",
  "@acme/badge-kit",
]
```

Versi diselesaikan lewat mekanisme yang sama dengan dependency package lain (`nova.lock`,
constraint di lock atau field opsional `packages."@acme/charts" = "1.2.0"` — detail syntax
mengikuti ADR-022 saat parser project diperluas).

**Tidak direkomendasikan** untuk production: `[[renderer.dictionary]]` panjang di `nova.toml`
dengan path `platform/` per entri. Itu hanya **escape hatch lokal** (monorepo dev, spike) dan
bukan cara publish/share library.

## 3. Resolve pipeline (build time)

```txt
<template> ... <sparkline data <- series /> ...
        │
        ▼
   ViewIR { kind: "sparkline", ... }
        │
        ▼
   Primitive resolver
        │
        ├─(1) built-in @nova/ui
        ├─(2) merged dictionary dari renderer.extensions.packages (urutan = urutan deklarasi)
        ├─(3) optional: renderer.dictionary lokal (dev override, explicit flag)
        └─(4) unknown_kind policy
        │
        ▼
   Artifact: import/register adapter dari setiap paket ter-resolve
```

Prioritas konflik `kind`:

```txt
built-in @nova/ui  >  local renderer.dictionary (jika allow_override)  >  later package in packages[]
```

Konflik antar dua renderer-package → `NVA-RENDER-003` (gagal build).

## 4. Prinsip teknis (tetap)

```txt
1. ViewIR tetap sumber semantic; paket tidak mengubah scheduler/event route.
2. Built-in @nova/ui selalu terdaftar implisit.
3. Adapter hanya mount/update view; dispatch event lewat runtime yang sama.
4. Build merge registrasi ke artifact — bukan dynamic plugin load di production v1.
5. Import .nova dari renderer-package opsional (re-export template/helper); aktivasi primitive
   tidak wajib lewat import — cukup paket terdaftar + kind dipakai di template.
```

---

# Schema: `nova.package.toml` (isi paket)

Contoh paket `@acme/charts`:

```toml
[package]
name = "@acme/charts"
version = "1.0.0"
type = ["renderer-package"]
language = ">=0.1.0"
abi = ">=0.1.0 <0.2.0"

[renderer.primitives.sparkline]
description = "Mini chart"
props = ["data", "width", "height"]
events = ["on_press"]
allow_override = false

[renderer.primitives.sparkline.web]
strategy = "adapter"
# path relatif ke root paket; default: platform/web/sparkline.web.js

[renderer.primitives.sparkline.android]
strategy = "adapter"

[targets.web]
adapter = "platform/web/register.web.js"

[targets.android]
adapter = "platform/android/Register.android.java"
```

Strategi per target (sama semantic seperti desain awal, tetapi **hidup di paket**, bukan di app):

| `strategy` | Arti |
| --- | --- |
| `adapter` | Kelas/modul terdaftar di `targets.*.adapter` atau file per-kind di dalam paket |
| `passthrough` | Web: tag HTML (`tag = "span"`) |
| `builtin_delegate` | Turunkan ke kind `@nova/ui` yang sudah ada |

Satu paket boleh memuat banyak `[renderer.primitives.*]`; satu file `register` web/android
memanggil `definePrimitive` untuk semua kind dalam paket.

### Isi ter-pack (layout paket)

```txt
@acme/charts/
  nova.package.toml
  platform/web/register.web.js
  platform/android/Register.android.java
  # opsional: src/*.nova (helper pure, bukan wajib untuk primitive saja)
```

Publish ke registry → consumer cukup:

```toml
# nova.toml aplikasi
[renderer.extensions]
packages = ["@acme/charts"]
```

---

# Schema: escape hatch lokal (non-primary)

Hanya untuk development atau primitive sekali pakai yang belum dipublish:

```toml
[[renderer.dictionary]]
kind = "local_only_widget"
  [renderer.dictionary.web]
  strategy = "adapter"
  adapter = "./platform/web/local_only_widget.web.js"
```

Rule:

```txt
1. CI/production disarankan packages = [...] saja; tanpa [[renderer.dictionary]].
2. Jika kind sama ada di package dan local, local hanya menang dengan allow_override = true.
3. Path adapter tetap relatif ke root project (ADR-012), bukan ke dalam paket.
```

---

# Hubungan import `.nova`

```nova
<!-- opsional: untuk helper atau re-export dokumentasi -->
<import { chartSeries } from "@acme/charts/series" /|

<template>
  <sparkline data <- chartSeries(points) />
/|
```

Agar `<sparkline />` valid di build:

```txt
@acme/charts harus ada di renderer.extensions.packages (atau transitive dependency
yang bertipe renderer-package dan diekspos sebagai extension — keputusan v2).
```

Import `.nova` saja **tanpa** entri di `renderer.extensions.packages` → diagnostic
`NVA-RENDER-001` atau `NVA-PKG-*` (primitive kind tidak ter-resolve).

---

# Adapter contract (JS / Java)

Tetap seperti desain awal: paket mengekspor registrasi terpusat.

```javascript
// @acme/charts/platform/web/register.web.js
export function register(NovaRenderer) {
  NovaRenderer.definePrimitive("sparkline", { mount, update, applyProps, applyEvents });
}
```

```java
// @acme/charts/platform/android/Register.android.java
public final class ChartsRegister {
  public static void register(NovaPrimitiveRegistry registry) { ... }
}
```

Build aplikasi meng-inline atau meng-import modul dari **hash ter-pin** di `nova.lock`, sama
spirit dengan adapter `@env/*`.

---

# Validasi & diagnostics

```txt
1. Setiap kind di ViewIR ter-resolve ke built-in atau salah satu renderer.extensions.packages.
2. Paket ter-resolve, hash cocok lockfile (production).
3. Target aktif punya adapter row di manifest paket.
4. Konflik kind antar paket → gagal build.
```

```txt
NVA-RENDER-001  unknown view kind for target
NVA-RENDER-002  unknown kind allowed with warning
NVA-RENDER-003  kind conflict between packages or built-in
NVA-RENDER-004  package adapter missing for target
NVA-PKG-*       resolution / lock / permission (ADR-022)
```

LSP: merge `standard.RendererPrimitives()` + primitives dari paket ter-resolve + lock path lokal.

---

# Hubungan ADR lain

| Topik | ADR |
| --- | --- |
| Package types & lockfile | ADR-022 |
| Template / ViewIR | ADR-006 |
| Lowering | ADR-008 |
| Built-in surface | ADR-015 |
| External OS capability | ADR-009 |
| Modular overview | ADR-026 |

ADR-026: tidak ada dynamic runtime registry; **paket ter-pack + pin lock** tetap deterministik.

---

# Implementasi bertahap

```txt
Phase A
  - Perluas packages.Manifest: renderer.primitives + parser nova.package.toml
  - project.ParseManifest: [renderer.extensions] packages = [...]

Phase B
  - packages.Resolve mengumpulkan dictionary dari renderer-package graph
  - Resolver + NVA-RENDER-*

Phase C (web/android)
  - Merge register.* dari setiap paket ke artifact

Phase D
  - Registry publish + nova.lock untuk renderer-package
  - Transitive renderer-package (opsional)

Phase E
  - Escape hatch [[renderer.dictionary]] lokal (dev)
  - Schema props validator per primitive
```

---

# Alternatives considered

## Dictionary penuh di `nova.toml` (desain awal)

Ditolak sebagai jalur utama: tidak reusable, duplikasi path web/android di setiap app, sulit
versioning dan audit — bertentangan dengan permintaan “packed library, cukup nama paket”.

## Hanya `import external` / `@env`

Ditolak untuk UI primitive: salah model (async operation, permission OS), bukan ViewIR kind.

## Hanya capability component

Ditolak untuk tag sederhana tanpa state contract; tetap valid untuk modul stateful.

## Dynamic plugin runtime

Ditolak (ADR-026).

---

# Consequences

## Positive

```txt
1. Satu paket = satu versi dictionary + adapter; konsumen hanya tulis nama di nova.toml.
2. Selaras dengan ADR-022 (renderer-package, lockfile, registry).
3. Tim bisa publish @acme/charts sekali; banyak app hanya depend.
4. Permission/adapter hash ter-review di lock diff.
```

## Negative

```txt
1. Perlu tooling package manifest + resolve sebelum fitur terasa di app.
2. Dua konsep package (renderer vs source vs @env) harus dokumentasi jelas.
3. Escape hatch lokal tetap ada untuk dev tetapi bisa membingungkan jika dipakai di production.
```

---

# Final position

```txt
Publish primitive gap  -> renderer-package (@scope/name) + nova.package.toml + platform/*
Pakai di app           -> nova.toml: renderer.extensions.packages = ["@scope/name"]
Core                   -> merge dictionary paket saat build; translate ke artifact

Jangan: daftar panjang [[renderer.dictionary]] di nova.toml untuk library bersama.
Boleh:  [[renderer.dictionary]] singkat hanya untuk spike lokal.
```

Aturan singkat:

```txt
Widget/tag shared antar project?     -> renderer-package, pin di lockfile.
Cukup di satu app, belum dipublish?  -> optional local renderer.dictionary + platform/.
Modul state + event contract?        -> capability component (.nova).
Storage/network/OS?                  -> @env external-capability-package.
```
