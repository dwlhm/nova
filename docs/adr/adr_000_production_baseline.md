# ADR-000: Production Baseline

## Status

Deprecated — referensi historis. Spesifikasi bahasa aktif: [ADR-013](adr_013_pure_expressions.md)–[ADR-016](adr_016_expression_pipeline.md).

## Scope

- **In scope:** target production, pipeline layer, governance ADR.
- **Out of scope:** detail grammar (ADR-001), scheduler (ADR-002).

## Context

Nova memisahkan compiler Go, kontrak ABI serializable, runtime native per target, dan adapter
`@env/*`. Production artifact tidak membaca source `.nova`.

## Decision

### Target production

```txt
web     -> artifact web + runtime JS (browser)
android -> artifact Android + runtime Java (Android View)
```

Renderer production Android: `@nova/android` (Java di `app/src/main/java`), tanpa framework UI
tambahan di luar primitive Nova.

### Pipeline

```txt
.nova
  -> Go: parse, validate, ViewIR, artifact
  -> app contract JSON (v1) + build.manifest (dev)
  -> Nova runtime native (JS / Java)
  -> provider: render, listen, effect ports
  -> @env/* -> API platform
```

### Aturan

1. Package Go `internal/scheduler`, `internal/app` = referensi conformance, bukan runtime produksi.
2. Runtime produksi memuat ABI + binding provider; tidak parse `.nova`.
3. External platform hanya lewat `@env/*` atau `<import external>` + lifecycle.
4. ADR `Accepted` / `Implemented / Accepted` mengikat implementasi.

## Consequences

- Perubahan semantic wajib conformance trace (ADR-011).
- Dokumen memakai istilah **production v1**, bukan "MVP" sebagai label target akhir.
