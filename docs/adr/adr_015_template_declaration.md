# ADR-015: Template declaration syntax

## Status

Proposed (locked)

## Depends on

ADR-014, ADR-016, ADR-017

## Scope

- **In scope:** keyword `template`, modifier `entry`, signature (`name`, parameters, return type node), body delimiter `:`, penutup `/|`, formatting, pemilihan entry di build plan, migrasi dari `<template>`.
- **Out of scope:** grammar isi body template (node calls, children `{ }`, binding state, event routes, routing/outlet) — spesifikasi terpisah; lowering ViewIR detail — ADR-005; sintaks ekspresi di argumen — ADR-016.

## Context

View dan func harus **terlihat satu keluarga** tanpa menyatukan keduanya jadi satu keyword. Template memakai header paralel func:

```txt
func     name(params) -> DataType:
template name(params) -> NodeType:
```

Return type template menjawab **bentuk node akar** yang diproduksi (`Surface`, `Page`, `Button`), bukan wadah generik `View`.

Routing di Nova adalah **event + state** (`@route_changed` → `route`), bukan construct template. Isi body (cara proyeksi route, outlet, `match route`, dll.) **tidak** dikunci di ADR ini agar tidak mengaburkan fakta event-driven routing.

## Decision

### 1. Keyword dan bentuk deklarasi

Template dideklarasikan dengan keyword `template`, **bukan** tag `<template>`.

```txt
TemplateDecl ::=
  "template" ["entry"] IDENT TemplateParams? "->" NodeType ":"
  TemplateBody
  "/|"
```

Contoh:

```nova
template TabButton(
  class: string,
  label: string,
  on_press: Event,
) -> Button:
  ...
/|

template FinanceApp() -> Surface:
  ...
/|

template entry FinanceApp() -> Surface:
  ...
/|
```

| Elemen | Aturan |
| --- | --- |
| Keyword | `template` |
| Entry | Opsional `entry` sebelum nama — menandai root screen modul entry |
| Nama | `IDENT` — `FinanceApp`, `TabButton` |
| Parameter | Opsional; jika ada, wajib dalam `(...)` (sama ADR-014) |
| Return type | Wajib `-> NodeType` |
| Pemisah body | `:` setelah header |
| Penutup | `/|` |

### 2. Parameter list

Sama dengan func (ADR-014):

```txt
TemplateParams ::= "(" Param { "," Param } [","] ")"
Param            ::= IDENT ":" Type
```

Template berparameter memodelkan komponen reusable; parameter = props/input template.

### 3. Return type (`NodeType`)

- **Wajib** eksplisit.
- Tipe **nominal node akar** yang diproduksi body, bukan `View` generik.
- Contoh: `Surface`, `Page`, `Scroll`, `Column`, `Row`, `Button`, `Text`.
- Dideklarasikan sebagai `type Surface: opaque` (ADR-017) atau lewat registry primitive `@nova/ui`.

Validator (di spesifikasi body): tipe node paling luar body harus **kompatibel** dengan `NodeType` header. Detail aturan kompatibilitas dan child matrix ada di spesifikasi isi template, bukan di ADR ini.

### 4. Entry template

- Modul **entry** project wajib punya tepat satu `template entry Name`.
- Build plan (`plan.Resolve`) memilih template entry untuk ViewIR, menggantikan pemilihan `<template>` pertama / `target <- web`.
- Tanpa `entry` pada modul non-entry: template adalah komponen biasa, callable dari template lain (detail pemanggilan di spesifikasi body).

### 5. Body delimiter (`:`)

- `:` **wajib** setelah `-> NodeType`.
- Body dimulai baris berikutnya (indent +1).
- Isi legal `TemplateBody` **tidak** didefinisikan di ADR ini.

### 6. Zona pure (deklarasi)

Template adalah construct **pure** (ADR-001): tidak menulis state, tidak memanggil external. Event routing ke `@event` scheduler diperbolehkan di body (detail di spesifikasi body). Larangan dan binding state mengacu ADR-005 semantik ViewIR.

### 7. Pemisahan dari routing

- Navigasi = `@event` mengubah state (biasanya field `route`) — ADR-002.
- Template **tidak** mendefinisikan router bahasa.
- Mendeklarasikan banyak template `-> Page` di entry **bukan** spesifikasi routing; itu daftar kemungkinan proyeksi (detail di spesifikasi body + runtime).
- Primitive `page(path: …)` jika masih dipakai adalah detail **isi** template, bukan bagian deklarasi ADR ini.

### 8. Modul dan komposisi

- Beberapa `template` per file diperbolehkan.
- Template dapat mengimpor modul lain (`import Widgets from "..."`) agar nama template dikenal di closure.
- Import simbol per-template (`import template X`) — out of scope fase ini; cukup import modul.

### 9. Formatting (`nova fmt`)

- Pola sama func: `:` di akhir header, body indent +1, `/|` sejajar `template`.
- Parameter multiline dan trailing comma didukung.

### 10. Migrasi dari syntax lama

| Lama | Baru |
| --- | --- |
| `<template>` | `template Name() -> NodeType:` atau `template entry Name() -> NodeType:` |
| `<template target <- web>` | Target dari build profile (ADR-009); metadata target fase lanjut jika perlu |
| Tag XML `<button …>` di body | Spesifikasi isi template (bukan ADR ini) |
| `attr <- expr` | Spesifikasi isi template |
| `on_press -> @evt` | Spesifikasi isi template |

Tidak ada backward compatibility untuk `<template>`.

### 11. Diagnostics (stable codes)

| Code | Kondisi |
| --- | --- |
| `nova_template_missing_colon` | Header tanpa `:` |
| `nova_template_missing_return` | Tanpa `-> NodeType` |
| `nova_template_invalid_return_type` | `NodeType` tidak dikenal |
| `nova_template_duplicate_entry` | Lebih dari satu `template entry` di modul entry |
| `nova_template_missing_entry` | Modul entry tanpa `template entry` |
| `nova_template_duplicate_name` | Nama template bentrok |

Diagnostic isi body (`nova_template_root_mismatch`, `nova_template_multiple_roots`, …) ada di spesifikasi isi template.

## Consequences

- Lexer: keyword `template`, modifier `entry`; hapus `TAG_TEMPLATE`.
- Parser: `TemplateDecl` dengan field `Entry bool`, `Return NodeType`.
- Plan: `selectTemplate` → cari `template entry`.
- ViewIR: authoring berubah; schema ViewIR ADR-005 tetap untuk artifact.
- LSP: hover `template FinanceApp() -> Surface`.
- ADR-001 / ADR-005 bagian authoring tag-based superseded untuk **deklarasi** setelah implementasi.

## Alignment

Header paralel ADR-014; routing tetap event-driven (ADR-002). Isi template dan isi func dipisah agar func dapat mengadopsi sintaks kode umum tanpa menarik UI ke ADR yang sama.

## Related

- [ADR-014](adr_014_func_declaration.md) — deklarasi `func`
- [ADR-005](adr_005_view.md) — ViewIR, lowering, primitives
- [ADR-002](adr_002_runtime.md) — route sebagai state + event
- Spesifikasi isi template (future) — node calls, children blocks, outlet / `match route`, child matrix
- [ADR-016](adr_016_expression_pipeline.md) — ekspresi di argumen binding / attr
- [ADR-017](adr_017_contract_declarations.md) — `component`, `type` opaque node
