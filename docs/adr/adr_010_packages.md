# ADR-010: Standard Packages

## Status

Deprecated — referensi historis package registry. Spesifikasi bahasa aktif: [ADR-014](adr_014_func_declaration.md)–[ADR-017](adr_017_contract_declarations.md).

## Depends on

ADR-003

## Scope

- **In scope:** namespace `@nova/*`, `@env/*`, UI primitives production v1.
- **Out of scope:** registry publik (format di ADR-003).

## Decision

### Namespace

```txt
@nova/ui, @nova/web, @nova/android, …   # framework & primitives
@env/storage, @env/network, …          # platform ports
```

Katalog built-in: `internal/standard` (merge primitive untuk LSP/tooling).

### Production v1 surface

- UI primitives untuk template (`button`, `text`, `page`, …).
- Renderer packages per target (`@nova/web`, `@nova/android`).
- `@env/*` mengikuti permission model ADR-006.

Package third-party memakai `nova.package.toml` + semver; import seperti capability lokal.

## Consequences

- Primitive baru → dictionary renderer (ADR-005) + conformance visual/structural jika ada.
