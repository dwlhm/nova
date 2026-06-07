# ADR-012: Portable style format (`.nova-style`)

## Status

Deprecated — referensi historis `.nova-style`. Spesifikasi bahasa aktif: [ADR-014](adr_014_func_declaration.md)–[ADR-016](adr_016_expression_pipeline.md).

## Depends on

ADR-005, ADR-008, ADR-009

## Scope

- **In scope:** `.nova-style` authoring, `internal/core/style` parse + IR, import discovery via
  module graph, web-canonical vocabulary, web CSS emit, Android style map, `.css` as web-only
  import, diagnostics, deprecation of TOML style lists.
- **Out of scope:** layout engine (`display`, flex/grid — use View primitives), CSS modules per
  capability, runtime theme switching, package style imports (`@scope/pkg`) in v1.

## Context

Nova today registers styles in `nova.toml` (`styles`, `scoped_styles`). CLI loads files into
`StyleAsset` strings; web copies or scopes CSS; Android parses a class-only CSS subset at
codegen. That pattern diverges from how `.nova` modules work (`<import>` + module graph from
`entry`) and causes drift (for example `:hover` on web, nothing on Android).

We need:

- **Semantic parity** for shared skin (same class names and state intent on web and Android).
- **Explicit dependencies** like other Nova modules (no orphan stylesheet lists in TOML).
- **Clear platform split:** portable `.nova-style` vs web-only `.css`.

## Decision

### Discovery: import in `.nova`, not `nova.toml`

Stylesheets are included only when reachable from `project.entry` through the **module closure**
(same transitive rule as `.nova` imports).

```nova
<import style from "./theme.nova-style" /|
<import style from "./Counter.nova-style" /|
<import stylesheet from "./effects.css" /|
```

| Construct | File types | Role |
| --- | --- | --- |
| `<import style from "…">` | `.nova-style` | Portable skin; parsed in core → StyleSheet IR |
| `<import stylesheet from "…">` | `.css` | **Web-only**; copied/linked in web artifact only |

- **`styles` and `scoped_styles` are removed** from `[targets.*]` in `nova.toml` (breaking
  change when implemented; migrate examples to imports).
- Import carries **no scope**; scope lives only in the style file (see below).
- Duplicate import of the same resolved path in the closure → `NVA-STYLE-003` (error).
- Future: `<import style from "@acme/theme/tokens.nova-style" />` via package exports (same
  pattern as capability imports); out of v1.

### One file, one scope

Every `.nova-style` file declares exactly one header:

```txt
scope: global | app
```

| Scope | Web behavior |
| --- | --- |
| `global` | Emitted CSS without app root prefix |
| `app` | Selectors prefixed with `#nova-root[data-nova-style-scope~="app"]`; sets root scope attr when any app-scoped sheet is linked |

- Mixed scopes in one file → `NVA-STYLE-010` (error).
- Missing or invalid `scope:` → `NVA-STYLE-010` (error).

### CSS is web-only

- `.css` files are accepted only via `<import stylesheet from "…">`.
- **Web:** copy or link into `build/web/assets/styles/…` and `style-manifest.json` (full CSS,
  including pseudo and selectors).
- **Android:** provider **skips** `.css` imports automatically (no parse, no error). Optional
  **info** `NVA-STYLE-013` per skipped file on Android builds (“stylesheet omitted on android”).
- Portable cross-target skin **must** use `.nova-style`, not `.css`.
- Legacy Android class parsing from `.css` in TOML is **removed** when TOML lists are removed.

### Naming: web-canonical vocabulary

Author-facing terms follow **CSS / HTML**, not Android resource names.

| Rule | Description |
| --- | --- |
| N1 | Property names are CSS kebab-case (`font-size`, `background-color`). |
| N2 | Interaction states use CSS pseudo names: `hover`, `focus`, `active`, `disabled`, `checked`. |
| N3 | If Android has no equivalent (for example `hover` on phone), keep the web term; compiler **warns** (`NVA-STYLE-011`) and skips on Android — authors do not rename to `pressed`. |
| N4 | If intent matches but names differ, authors use the web term; compiler maps internally (`active` → Android pressed). |
| N5 | Template skin uses `class <- "literal"` (HTML `class`), not Android theme resource names. |

See [style-format.md](../style-format.md) for the author glossary and examples.

### File shape (v1)

```txt
scope: global | app

token <dotted.name> = <literal>

class <ident> {
  <property>: <value>
  ...
}

state <ident> <pseudo> {
  <property>: <value>
  ...
}
```

- Comments: `//` to end of line.
- Class identifiers: `[a-z][a-z0-9-]*` (kebab-case recommended).
- Portable files **must not** contain raw CSS selectors or `@` rules.

### Portable property subset (v1)

Properties valid for **web + android** builds from `.nova-style`:

| Property | Notes |
| --- | --- |
| `color` | Text targets |
| `font-size` | Prefer `sp` in portable files |
| `font-weight` | `normal`, `bold`, or `100`–`900` |
| `text-align` | `left`, `center`, `right` |
| `text-transform` | `none`, `uppercase` (v1) |
| `line-height` | Unitless only in portable v1 |
| `padding` | 1–4 values, CSS shorthand (inner spacing) |
| `margin` | 1–4 values, CSS shorthand (outer spacing) |
| `min-height` | |
| `background`, `background-color` | |
| `border`, `border-color`, `border-width` | |
| `border-radius` | |
| `align-content` | Degraded on Android |

Property outside portable subset in `.nova-style` → `NVA-STYLE-009` (error).

### Interaction states (v1)

Declared with `state <class> <pseudo>`, not raw selectors in portable files.

| Pseudo (author) | Web emit | Android map |
| --- | --- | --- |
| `hover` | `:hover` | Skip + `NVA-STYLE-011` on phone |
| `focus` | `:focus` / `:focus-visible` | Focused state on `View` |
| `active` | `:active` | Pressed state |
| `disabled` | `:disabled` | Disabled / not enabled |
| `checked` | `:checked` | Toggle primitives when supported |

Web-only effects (`:hover` in raw CSS, `@media`, etc.) belong in **`.css`** imported via
`<import stylesheet>`.

### Tokens

- Tokens resolve at **compile time** only; no runtime arithmetic in v1.
- Undefined token → `NVA-STYLE-005`; cyclic reference → `NVA-STYLE-006`.
- Duplicate class in the same scope across merged sheets → `NVA-STYLE-004`.

### Template binding

- Portable class styling applies when `class` is a **static string literal** in the contract.
- Non-literal `class` → `NVA-STYLE-007` (Android skips portable class map).
- Unknown class → `NVA-STYLE-008` (warning).

Layout remains ViewIR primitives and runtime base CSS; portable classes are **skin**, not layout.

### Compile pipeline

```txt
entry .nova
  -> core/plan: module closure (existing)
  -> core/style: collect style + stylesheet imports from closure
  -> core/style: parse .nova-style -> StyleSheet IR; record .css as WebStylesheetRef
  -> core/style: validate IR (syntax, tokens, portable subset, duplicate classes)
  -> provider/web: IR -> emitted CSS + .css files -> index + style-manifest.json
  -> provider/android: IR only; ignore WebStylesheetRef
```

Deterministic ordering: sort resolved style paths; stable `order` in web manifest.

Android does not ship portable rules as CSS in the APK; IR is applied at MainActivity codegen.

### Layer ownership

| Package | Responsibility |
| --- | --- |
| `internal/core/style` | Parse `.nova-style`, StyleSheet IR, validation, merge by scope |
| `internal/core/plan` | Extend module walk to collect style/stylesheet imports (or delegate to style) |
| `internal/provider/web` | Emit CSS, bundle `.css`, scoping, manifest |
| `internal/provider/android` | Lower IR to native view styling; skip `.css` |
| `internal/provider/shared` | Codegen helpers only (scoping, types on `GenerateInput`) |
| `internal/cli` | Remove `loadStyleAssets` from TOML; pass IR/refs from compile result |

Core must not import provider. Providers consume IR produced by core.

### Diagnostics (style domain)

| Code | Severity | Condition |
| --- | --- | --- |
| `NVA-STYLE-001` | error | Unsafe style path |
| `NVA-STYLE-002` | error | Unreadable file |
| `NVA-STYLE-003` | error | Duplicate import path or conflicting registration |
| `NVA-STYLE-004` | error | Duplicate class in same scope |
| `NVA-STYLE-005` | error | Unknown token |
| `NVA-STYLE-006` | error | Cyclic token |
| `NVA-STYLE-007` | warning | Non-static `class` (Android portable skip) |
| `NVA-STYLE-008` | warning | Unknown class |
| `NVA-STYLE-009` | error | Property not in portable subset (`.nova-style`) |
| `NVA-STYLE-010` | error | Invalid syntax, or scope missing/duplicate in file |
| `NVA-STYLE-011` | warning | Pseudo ignored on Android (for example `hover`) |
| `NVA-STYLE-013` | info | `.css` stylesheet skipped on Android build |

### Migration

1. Add `<import style>` / `<import stylesheet>` to grammar and module collection.
2. Implement `core/style` parser + IR; wire web/android providers.
3. Update examples: remove `styles` / `scoped_styles` from `nova.toml`; add imports in entry/modules.
4. Remove CLI `loadStyleAssets` and TOML keys `styles`, `scoped_styles`.
5. Until (1)–(4) land, **current** TOML + `.css` behavior remains in production code.

## Consequences

- Style dependencies are auditable in source beside templates.
- Android no longer pretends to understand full CSS; only `.nova-style` IR drives native skin.
- ADR-005 references this ADR; [style-format.md](../style-format.md) is the author guide.
- Architecture test must allow `internal/core/style`; providers depend on style IR types from core.

## Alignment

Selaras dengan [design-philosophy.md](../design-philosophy.md) (semantic parity, explicit
dependencies) dan [architecture.md](../architecture.md) (core target-neutral IR, provider codegen).
