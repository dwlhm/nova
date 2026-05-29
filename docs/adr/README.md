# Architecture Decision Records

ADR mencatat keputusan yang mengikat implementasi. Ringkas; detail operasional di kode dan
`internal/`.

| Dokumen | Peran |
| --- | --- |
| [design-philosophy.md](../design-philosophy.md) | Arah produk |
| [language-design.md](../language-design.md) | Normatif untuk menulis `.nova` |
| ADR di folder ini | Spesifikasi per area |

## Mulai di sini

1. [language-design.md](../language-design.md)
2. [adr_000](adr_000_production_baseline.md)
3. ADR sesuai area di bawah

## Indeks

| ADR | Topik |
| --- | --- |
| [000](adr_000_production_baseline.md) | Baseline production, pipeline |
| [001](adr_001_language_specification.md) | Bahasa: construct, purity, syntax |
| [002](adr_002_runtime.md) | Scheduler, lifecycle, route, persistence |
| [003](adr_003_modules.md) | Capability, layout, package, provider binding |
| [004](adr_004_type_system.md) | Types, serializable |
| [005](adr_005_view.md) | ViewIR, routing, renderer, primitives |
| [006](adr_006_external_security.md) | External, `@env`, permission |
| [007](adr_007_build.md) | Build, target resolution |
| [008](adr_008_diagnostics.md) | Diagnostic codes |
| [009](adr_009_targets.md) | Web & Android runtime |
| [010](adr_010_packages.md) | `@nova/*`, `@env/*` |
| [011](adr_011_tooling.md) | CLI, test, conformance, dev |

## Istilah

| Istilah | Arti |
| --- | --- |
| **production v1** | Target `web` + `@nova/android` yang didukung hari ini |
| **Nova runtime** | Scheduler + transisi di device (JS/Java) |
| **Provider** | Render, listen, effect ports; bisa bawaan atau custom |

## Renderer default

```toml
[targets.web]
renderer = "@nova/web"

[targets.android]
renderer = "@nova/android"
```

## ADR baru

Salin [TEMPLATE.md](TEMPLATE.md). Link ADR terkait; jangan duplikasi panjang. Update conformance
jika mengubah scheduler, ViewIR, atau permission.

## ADR lama (dihapus)

ADR 012–027 dan file terpisah untuk scheduler/view/build/tooling telah digabung ke indeks di
atas. Referensi historis di git history.
