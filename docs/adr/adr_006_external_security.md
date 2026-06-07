# ADR-006: External Interop & Security

## Status

Deprecated — referensi historis external/permission. Spesifikasi bahasa aktif: [ADR-014](adr_014_func_declaration.md)–[ADR-017](adr_017_contract_declarations.md).

## Depends on

ADR-001, ADR-003

## Scope

- **In scope:** `<import external>`, `@env/*`, permission manifest.
- **Out of scope:** implementasi adapter per OS (ADR-009).

## Decision

### External script

Bahasa lain (JS, Kotlin, Java, …) = dirty. Hanya lewat `<import external>` dengan kontrak
operasi typed; dipanggil dari lifecycle.

### @env/*

Namespace capability platform (`@env/storage`, `@env/network`, …). Satu pola permission untuk
semua target.

### Permission

Default deny. Manifest aplikasi + build menghasilkan tabel permission; setiap effect harus
terdaftar. Build gagal jika lifecycle memanggil operasi tanpa izin.

Data antar zona (core, ABI, persistence) serializable.

### Provider custom

Provider tidak membuka permission di luar manifest capability yang di-handle.

## Consequences

- Perubahan permission schema → ADR-000 artifact + security validator.
- Audit: graph lifecycle + external table di CI.
