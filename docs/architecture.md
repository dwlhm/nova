# Architecture

Dokumen ini adalah peta kerja untuk implementasi Go Nova.

| Dokumen | Peran |
| --- | --- |
| [design-philosophy.md](design-philosophy.md) | Visi produk (di luar detail `.nova`; sumber arah) |
| [language-design.md](language-design.md) | Prinsip bahasa ringkas (normatif untuk `.nova`) |
| [adr/README.md](adr/README.md) | Indeks ADR & jalur baca per topik |
| [contract.md](contract.md) | Kontrak runtime bundle (`App` v1) |
| [style-format.md](style-format.md) | Penulisan `.nova-style` portable (author) |

**Mulai baca:** [language-design.md](language-design.md) → [adr_000](adr/adr_000_production_baseline.md) →
[adr/README.md](adr/README.md) (indeks ADR 000–011).

## Pipeline

### Core compiler (target-neutral)

```txt
Lexer
  -> Parser
  -> Raw AST            (internal/core/ast, internal/core/parser)
  -> Semantic Analysis  (internal/core/semantic, internal/core/validator, internal/core/types)
  -> Checked AST
  -> Core IR Lowering   (internal/core/ir)
  -> Core Nova IR       (contract.App + ViewIR bundle)
  -> Core Build Plan    (internal/core/plan: module graph + entry template)
```

Orkestrasi per tahap: `internal/core/compile`. Provider layer menambah resolusi target,
renderer, permission audit, dan codegen setelah core selesai.

### End-to-end

```txt
cmd/nova
  -> internal/cli
  -> project manifest and source discovery
  -> internal/core/compile (parse, semantic check)
  -> internal/provider/build (target resolution + renderer validation)
  -> internal/core/compile.Lower + internal/core/ir (Nova IR + core plan)
  -> internal/provider/artifact (codegen)
  -> bundler/dev/conformance
```

Rule utama: source `.nova` diturunkan menjadi kontrak data yang deterministik. Target runtime tidak
membaca source `.nova` langsung dari production artifact.

Production runtime:

```txt
web     -> JavaScript di browser (runtime/nova-scheduler-js + generated web runtime)
android -> Java di Android SDK View layer (@nova/android + runtime/nova-scheduler-java)
```

Scheduler and renderer production code lives in `runtime/nova-scheduler-js`,
`runtime/nova-scheduler-java`, and `runtime/nova-renderer-js` as installable libraries;
`internal/provider/artifact` embeds and copies them into build output.

Go packages `internal/scheduler`, `internal/app`, dan sejenisnya adalah conformance reference;
bukan runtime production userland.

## Layers

```txt
Edge tooling
  cmd/nova, internal/cli, internal/bundler, internal/conformance, internal/packageio

Project and package planning
  internal/project, internal/packages

Core (target-neutral)
  internal/core/lexer, internal/core/parser, internal/core/ast, internal/core/semantic
  internal/core/types, internal/core/validator, internal/core/plan, internal/core/compile
  internal/core/view, internal/core/capability, internal/core/contract, internal/core/ir
  internal/core/routing, internal/core/security, internal/core/scheduler, internal/core/effect
  internal/core/app, internal/core/persistence, internal/core/diagnostic, internal/core/format
  internal/core/style

Provider (target-specific)
  internal/provider/build, internal/provider/target, internal/provider/artifact
  internal/provider/shared, internal/provider/web, internal/provider/android
  internal/provider/standard

Dev helpers
  internal/dev, internal/tooling, internal/lsp
```

Dependency direction mengalir dari edge ke core contracts, bukan sebaliknya. Package core tidak boleh
mengambil dependency ke CLI, bundler, atau filesystem host.

## Package Ownership

| Package | Responsibility |
| --- | --- |
| `cmd/nova` | Binary entrypoint tipis; delegasi ke `internal/cli`. |
| `internal/cli` | Parse flag, baca/tulis file, orkestrasi pipeline, dev server, dan command output. |
| `internal/packageio` | Filesystem loader untuk `nova.package.toml`, adapter package, dan package graph project. |
| `internal/project` | Parse `nova.toml` dan validasi layout project. |
| `internal/core/lexer` | Tokenisasi source `.nova`, tanpa IO. |
| `internal/core/parser` | Parser syntax: token → raw AST (`parser.File`). |
| `internal/core/ast` | Tipe Raw AST dan Checked AST. |
| `internal/core/semantic` | Semantic analysis: raw AST → checked AST. |
| `internal/core/types` | Type reference parsing, assignability, dan validasi value serializable. |
| `internal/core/validator` | Aturan semantic (dipanggil dari `semantic`). |
| `internal/core/plan` | Core build plan: module graph dan template entry. |
| `internal/core/compile` | Orkestrasi pipeline core (parse → check → lower → plan). |
| `internal/core/ir` | Core IR lowering: checked AST → Core Nova IR (`ir.NovaIR`). |
| `internal/provider/build` | Module graph, template selection, external implementation, permission planning, renderer validation. |
| `internal/provider/target` | Kontrak artifact target dan validasi metadata runtime. |
| `internal/core/security` | Permission audit dan validasi event host/runtime. |
| `internal/core/routing` | Matching route target-neutral untuk page projection, dynamic params, wildcard, dan fallback. |
| `internal/core/view` | Projection template menjadi ViewIR dan dependency metadata. |
| `internal/core/contract` | Kontrak runtime bundle (`App` v1, `BuildManifest`). |
| `internal/core/capability` | Manifest capability dari source/parser contract. |
| `internal/core/style` | Parse `.nova-style`, StyleSheet IR, validasi portable (target); koleksi import dari module graph. |
| `internal/provider/shared` | Tipe input codegen, manifest, CSS scoping, helper generik lintas target. |
| `internal/provider/web` | Codegen artifact web (sandboxed; tidak mengimpor `android`). |
| `internal/provider/android` | Codegen artifact Android (sandboxed; tidak mengimpor `web`). |
| `internal/provider/artifact` | Fasad tipis: delegasi `Generate` ke `web` atau `android` berdasarkan target. |
| `internal/provider/standard` | Katalog built-in `@nova/ui` dan merge primitive renderer (ADR-010, ADR-005). |
| `internal/bundler` | Validasi artifact target, manifest bundle, Gradle/process execution. |
| `internal/dev` | Planning dev cycle yang pure dan mudah diuji. |
| `internal/tooling` | Helper tooling kecil yang tidak masuk pipeline utama. |
| `internal/core/scheduler` | Queue, event envelope, commit, lifecycle, dan runtime scheduler semantics. |
| `internal/core/effect` | Port external operation dan completion event. |
| `internal/core/app` | Runtime app shell yang menghubungkan scheduler, view, effect, dan diagnostics. |
| `internal/core/persistence` | Hydration/snapshot contract. |
| `internal/packages` | Manifest package, lockfile, target adapter, dan permission resolution. |
| `internal/conformance` | Fixture runner dan trace comparison untuk kontrak lintas target. |
| `internal/core/diagnostic` | Diagnostic shape, severity, sorting, dan JSONL output. |
| `internal/core/format` | Formatter source `.nova`. |

## Dependency Rules

- `cmd/nova` hanya memanggil `internal/cli`.
- `internal/cli` boleh mengorkestrasi banyak package, tetapi tidak menjadi tempat semantic rule baru.
- Package `internal` hanya boleh bergantung pada exposed contract package lain. Jangan membuat package
  mengetahui detail implementasi, format private, cara discovery, atau lifecycle internal package lain.
- Glue lintas package harus tinggal di package dengan ownership domain yang tepat. Jika belum ada
  tempat yang sesuai, buat package baru dengan responsibility sempit dan dependency direction jelas.
- `internal/provider/artifact` boleh bergantung ke build/view/target contract, tetapi tidak boleh menjalankan
  process atau menulis filesystem.
- `internal/bundler` boleh menjalankan process target seperti Gradle.
- `internal/conformance` boleh membaca fixture dan memanggil pipeline untuk membandingkan expected
  contract.
- `internal/dev` tetap pure planning; IO dev server berada di `internal/cli`.
- Package di `internal/core/*` tidak boleh import `internal/provider/*` atau package edge.
- `internal/core/ir` menurunkan checked AST ke Core Nova IR (`contract.App` + ViewIR) tanpa detail web/android/codegen.
- `internal/core/plan` memegang core build plan; `internal/provider/build` memperkaya dengan target, renderer, dan external resolution.
- `internal/provider/build` + `internal/provider/target` menangani perencanaan target;
  `internal/provider/artifact` menangani codegen.
- Package yang mengeluarkan slice dari map atau graph harus mengurutkan hasil.

Rule ini diperiksa oleh test arsitektur di `internal/architecture`.

## Adding A Feature

### Syntax Or Language Semantics

1. Update [language-design.md](language-design.md) bila prinsip ringkas berubah; update ADR-001 dan ADR terkait untuk detail.
2. Ubah lexer/parser (raw AST).
3. Tambahkan semantic validation/type behavior (checked AST).
4. Update core IR lowering atau core build plan bila kontrak turunan berubah.
5. Update formatter bila bentuk source berubah.
6. Tambahkan unit test dan conformance fixture bila surface lintas target berubah.

### Target Or Artifact Behavior

1. Ubah target/build contract di `internal/provider/build` atau `internal/provider/target`.
2. Ubah generator di `internal/provider/artifact`.
3. Ubah bundling hanya jika output final atau tool target ikut berubah.
4. Verifikasi dengan example web/android yang relevan.

### CLI Behavior

1. Simpan parsing flag dan IO di `internal/cli`.
2. Dorong planning atau transformasi ke package pure jika behavior perlu diuji terpisah.
3. Tambahkan CLI test untuk output, exit code, dan error.

### Diagnostics

1. Pilih prefix code dari ADR-008.
2. Pastikan ordering deterministic.
3. Update unit test dan conformance expected jika diagnostic menjadi contract.

### Scheduler Semantics

1. Tambahkan atau ubah blok `scheduler` di `nova.conformance.json` bila transisi state, lifecycle,
   atau external operation ikut berubah.
2. `steps` memuat aksi simulasi (`enqueue`); `trace` memuat expected event, commit, lifecycle,
   external call, dan error trace yang dibandingkan dengan scheduler Go reference.
3. Jalankan `go run ./cmd/nova test` setelah mengubah expected trace.

## ADR Boundary

Buat ADR baru ketika perubahan:

- Mengubah source language atau ABI.
- Mengubah scheduler, lifecycle, permission, persistence, atau ViewIR semantics.
- Menambah target runtime resmi.
- Mengubah project layout/package convention.
- Mengubah conformance contract.

Perubahan kecil pada implementasi lokal cukup didokumentasikan di test dan kode.
