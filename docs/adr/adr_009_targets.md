# ADR-009: Target Runtimes (Web & Android)

## Status

Deprecated — referensi historis target runtime. Spesifikasi bahasa aktif: [ADR-013](adr_013_pure_expressions.md)–[ADR-016](adr_016_expression_pipeline.md).

## Depends on

ADR-000, ADR-002, ADR-007

## Scope

- **In scope:** bundling web, project Android, runtime JS/Java, provider defaults.
- **Out of scope:** iOS/desktop (future).

## Decision

### Web

- Artifact: static assets + generated JS scheduler + View binding.
- Runtime: `runtime/nova-scheduler-js` + renderer JS.
- Provider default `@nova/web`: render DOM, listen events, effect ports (fetch, storage, …).
- Delivery (SPA, SSR, SSG) = keputusan provider/bundler, bukan core; kontrak state/event sama.

### Android

- Artifact: Gradle project + generated Java.
- Runtime: `runtime/nova-scheduler-java` + `@nova/android` renderer.
- UI: Android `View` framework; tanpa library UI tambahan untuk primitive bawaan.
- APK: hanya primitive yang dipakai; bundler tidak menarik dependency UI tidak perlu.

### Provider

Target memuat Nova runtime + provider terikat dari artifact. Custom provider menggantikan
implementasi node/effect untuk capability terdaftar (ADR-003).

### Conformance

Trace event scheduler + snapshot harus match antara referensi Go dan runtime target untuk
fixture yang sama.

## Consequences

- `internal/bundler` menjalankan toolchain target (Gradle, dll.).
- Perubahan ABI scheduler → update kedua runtime + fixture.
