# Architecture

Dokumen ini adalah peta kerja untuk implementasi Go Nova. ADR di `docs/adr/` tetap menjadi sumber keputusan desain. Indeks lengkap:
[docs/adr/README.md](adr/README.md). Mulai dari `docs/adr/adr_000_production_baseline.md`, lalu
ADR-011, ADR-012, ADR-014, ADR-017, ADR-020, ADR-024, dan ADR-025.

## Pipeline

```txt
cmd/nova
  -> internal/cli
  -> project manifest and source discovery
  -> lexer
  -> parser
  -> validator + types + security
  -> build target resolution
  -> view projection
  -> artifact generation
  -> bundler/dev/conformance
```

Rule utama: source `.nova` diturunkan menjadi kontrak data yang deterministik. Target runtime tidak
membaca source `.nova` langsung dari production artifact.

Production runtime:

```txt
web     -> JavaScript di browser (internal/artifact web bundle)
android -> Java di Android SDK View layer (@nova/android, tanpa Kotlin/Compose default)
```

Go packages `internal/scheduler`, `internal/app`, dan sejenisnya adalah conformance reference;
bukan runtime production userland.

## Layers

```txt
Edge tooling
  cmd/nova, internal/cli, internal/bundler, internal/conformance

Project and target planning
  internal/project, internal/build, internal/target, internal/packages, internal/standard

Language frontend and semantic core
  internal/lexer, internal/parser, internal/types, internal/validator

Runtime contracts
  internal/scheduler, internal/effect, internal/app, internal/persistence, internal/security,
  internal/routing

Rendering and artifacts
  internal/view, internal/capability, internal/artifact

Dev helpers
  internal/dev, internal/tooling, internal/format
```

Dependency direction mengalir dari edge ke core contracts, bukan sebaliknya. Package core tidak boleh
mengambil dependency ke CLI, bundler, atau filesystem host.

## Package Ownership

| Package | Responsibility |
| --- | --- |
| `cmd/nova` | Binary entrypoint tipis; delegasi ke `internal/cli`. |
| `internal/cli` | Parse flag, baca/tulis file, orkestrasi pipeline, dev server, dan command output. |
| `internal/project` | Parse `nova.toml` dan validasi layout project. |
| `internal/lexer` | Tokenisasi source `.nova`, tanpa IO. |
| `internal/parser` | AST/data model dari token Nova. |
| `internal/types` | Type reference parsing, assignability, dan validasi value serializable. |
| `internal/validator` | Semantic validation lintas parser, type system, view, dan capability. |
| `internal/build` | Module graph, template selection, external implementation, permission planning. |
| `internal/target` | Kontrak artifact target dan validasi metadata runtime. |
| `internal/security` | Permission audit dan validasi event host/runtime. |
| `internal/routing` | Matching route target-neutral untuk page projection, dynamic params, wildcard, dan fallback. |
| `internal/view` | Projection template menjadi ViewIR dan dependency metadata. |
| `internal/capability` | Manifest capability dari source/parser contract. |
| `internal/artifact` | Generate file web/android dari build plan dan IR, tanpa menulis disk. |
| `internal/bundler` | Validasi artifact target, manifest bundle, Gradle/process execution. |
| `internal/dev` | Planning dev cycle yang pure dan mudah diuji. |
| `internal/tooling` | Helper tooling kecil yang tidak masuk pipeline utama. |
| `internal/scheduler` | Queue, event envelope, commit, lifecycle, dan runtime scheduler semantics. |
| `internal/effect` | Port external operation dan completion event. |
| `internal/app` | Runtime app shell yang menghubungkan scheduler, view, effect, dan diagnostics. |
| `internal/persistence` | Hydration/snapshot contract. |
| `internal/packages` | Manifest package, lockfile, target adapter, dan permission resolution. |
| `internal/standard` | Surface package resmi `@nova/*` dan `@env/*`. |
| `internal/conformance` | Fixture runner dan trace comparison untuk kontrak lintas target. |
| `internal/diagnostic` | Diagnostic shape, severity, sorting, dan JSONL output. |
| `internal/format` | Formatter source `.nova`. |

## Dependency Rules

- `cmd/nova` hanya memanggil `internal/cli`.
- `internal/cli` boleh mengorkestrasi banyak package, tetapi tidak menjadi tempat semantic rule baru.
- `internal/artifact` boleh bergantung ke build/view/target contract, tetapi tidak boleh menjalankan
  process atau menulis filesystem.
- `internal/bundler` boleh menjalankan process target seperti Gradle.
- `internal/conformance` boleh membaca fixture dan memanggil pipeline untuk membandingkan expected
  contract.
- `internal/dev` tetap pure planning; IO dev server berada di `internal/cli`.
- Core language package (`lexer`, `parser`, `types`, `validator`) tidak boleh import package edge.
- Package yang mengeluarkan slice dari map atau graph harus mengurutkan hasil.

Rule ini diperiksa oleh test arsitektur di `internal/architecture`.

## Adding A Feature

### Syntax Or Language Semantics

1. Update ADR atau dokumen spec yang relevan.
2. Ubah lexer/parser AST.
3. Tambahkan semantic validation/type behavior.
4. Update formatter bila bentuk source berubah.
5. Tambahkan unit test dan conformance fixture bila surface lintas target berubah.

### Target Or Artifact Behavior

1. Ubah target/build contract di `internal/build` atau `internal/target`.
2. Ubah generator di `internal/artifact`.
3. Ubah bundling hanya jika output final atau tool target ikut berubah.
4. Verifikasi dengan example web/android yang relevan.

### CLI Behavior

1. Simpan parsing flag dan IO di `internal/cli`.
2. Dorong planning atau transformasi ke package pure jika behavior perlu diuji terpisah.
3. Tambahkan CLI test untuk output, exit code, dan error.

### Diagnostics

1. Pilih prefix code dari ADR-011.
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
