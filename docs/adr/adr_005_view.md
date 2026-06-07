# ADR-005: View, ViewIR, Routing, Renderer

## Status

Deprecated — referensi historis ViewIR/template authoring. Deklarasi template aktif: [ADR-015](adr_015_template_declaration.md); ekspresi: [ADR-016](adr_016_expression_pipeline.md).

## Depends on

ADR-001, ADR-004

## Scope

- **In scope:** template → ViewIR, page routing, lowering, style injection, primitive registry.
- **Out of scope:** bundler Gradle (ADR-009), provider delivery SPA/SSR (pilihan provider).

## Decision

### Template → ViewIR

```txt
(StateSnapshot, props, template) -> ViewIR + dependency metadata
```

ViewIR: node tree, attributes, bindings, event routes, list/conditional projection. Tidak
mengandung node DOM/View native.

Template tidak memanggil external; tidak menulis state.

### Page & routing

Primitive ViewIR `page` (bukan construct bahasa):

```nova
<page path <- "/settings">
  <text value <- "Settings" /|
/|
```

Route aktif dari state `route` (string atau `Route { path }`). Artifact menyertakan route table
target-neutral (`pattern`, `nodePath`, `params`, `fallback`).

### Lowering

Build menurunkan ViewIR ke keluaran target:

```txt
web     -> DOM/JS binding, metadata SSR/SSG/CSR (strategi di provider)
android -> Java View binding
```

Semantic ViewIR dan event route tidak berubah per strategi delivery.

### Style & asset

Style lewat manifest build / metadata artifact (token, file path). Injection ke renderer di
provider; tidak hardcode path platform di core.

Portable authoring: [ADR-012](adr_012_style_format.md) — `.nova-style` via `<import style>`,
web-only `.css` via `<import stylesheet>`, one scope per file, discovery from module graph (not
`nova.toml`). Author guide: [style-format.md](../style-format.md).

### Primitive & extension

Primitive bawaan: `@nova/ui`, `@nova/web`, `@nova/android` (ADR-010). Extension primitive
mendaftar di dictionary renderer; provider custom mengimplement node ID yang dikontrakkan.

Android production: renderer Java native; ukuran APK — hindari dependency UI di luar primitive
yang dipakai (detail bundler ADR-009).

## Consequences

- Perubahan ViewIR schema → bump artifact version + conformance snapshot.
