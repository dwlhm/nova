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
2. ADR aktif [014](adr_014_func_declaration.md)–[016](adr_016_expression_pipeline.md)
3. ADR deprecated (000–013) hanya referensi historis

## Indeks — aktif

| ADR | Topik | Status |
| --- | --- | --- |
| [014](adr_014_func_declaration.md) | Deklarasi `func` (header, `:`, signature) | Proposed (locked) |
| [015](adr_015_template_declaration.md) | Deklarasi `template` (header, `entry`, `NodeType`) | Proposed (locked) |
| [016](adr_016_expression_pipeline.md) | Ekspresi pure, pipeline `\|>`, special forms, anon | Accepted (locked) |

## Indeks — deprecated

ADR 000–013 **tidak** menjadi sumber kebenaran grammar bahasa; implementasi legacy masih di kode.

| ADR | Topik |
| --- | --- |
| [000](adr_000_production_baseline.md) | Baseline production, pipeline |
| [001](adr_001_language_specification.md) | Bahasa: construct, purity, syntax (legacy) |
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
| [012](adr_012_style_format.md) | `.nova-style`, import discovery, core/style |
| [013](adr_013_pure_expressions.md) | Pure expression eval, lowering (legacy) |

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

Salin [TEMPLATE.md](TEMPLATE.md). Link ADR terkait; jangan duplikasi panjang. Perubahan grammar
ekspresi/deklarasi masuk ADR aktif (014–016) atau ADR baru yang menggantikan bagian spesifik.

## ADR lama (dihapus)

ADR 012–027 dan file terpisah untuk scheduler/view/build/tooling telah digabung ke indeks di
atas. Referensi historis di git history.
