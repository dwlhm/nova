# ADR-003: Modules, Capabilities, Project Layout, Packages

## Status

Deprecated — referensi historis capability/module graph. Spesifikasi bahasa aktif: [ADR-014](adr_014_func_declaration.md)–[ADR-017](adr_017_contract_declarations.md).

## Depends on

ADR-001

## Scope

- **In scope:** capability graph, layout repo, package manifest, provider binding di manifest.
- **Out of scope:** resolver build detail (ADR-007), standard package API (ADR-010).

## Decision

### Capability

- Identitas = resolved path (`./Foo.nova`, `@nova/ui/Button`, `@env/storage`).
- Symbol publik: type, state, capability, func, template, external operation.
- Lifecycle tidak symbol publik; dihubungkan lewat graph event/state import.

Import tidak mengeksekusi side effect.

### Dependency graph

Build memvalidasi: siklus, target compatibility, permission, external implementation per target.

### Layout project (production v1)

```txt
nova.toml
nova.package.toml          # opsional, package registry lokal
src/
  App.nova                 # entry capability
platform/                  # opsional: external per target
packages/                  # opsional: workspace packages
```

`nova.toml` minimal:

```toml
[project]
name = "app"
version = "0.1.0"
entry = "src/App.nova"

[targets.web]
renderer = "@nova/web"

[targets.android]
renderer = "@nova/android"
```

### Package

- Satu package = satu atau lebih capability + manifest `nova.package.toml`.
- Versi semver; lockfile untuk reproducible build.
- Distribution: path lokal, git, atau registry (format lock diimplementasi `internal/packages`).

### Provider binding (opsional)

Manifest dapat mendaftarkan provider custom per capability:

```toml
[providers.maps]
handles = ["@acme/maps"]
targets = ["web", "android"]
entry = "./providers/maps"
```

Build mencatat binding di artifact. Tanpa binding → provider referensi bawaan.

### Platform capability

Capability boleh mendeklarasikan dukungan target (`web`, `android`, …). Build menolak referensi
tanpa implementasi untuk target aktif.

## Consequences

- Perubahan graph rules → validator + fixture conformance.
- Lihat ADR-007 untuk resolusi target saat build.
