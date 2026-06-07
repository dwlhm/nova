# ADR-014: Func declaration syntax

## Status

Proposed (locked)

## Depends on

ADR-001, ADR-004, ADR-016

## Scope

- **In scope:** keyword `func`, signature (`name`, parameters, return type), body delimiter `:`, penutup `/|`, formatting, migrasi dari `<func>`, aturan modul/registry di level deklarasi.
- **Out of scope:** grammar isi body func (ekspresi, pipeline, special forms) — ADR-016; lowering runtime — `internal/core/expr`; `template` — ADR-015.

## Context

Deklarasi `<func name: type name: type returns R>` sulit dibaca dan tidak selaras dengan dialect deklarasi modern. Func dan `template` harus **terlihat satu keluarga** (`keyword name(params) -> Type:`) sambil zona pure tetap terpisah.

Isi body func (sintaks kode umum) dibahas di dokumen lain agar ADR ini hanya mengunci **bentuk deklarasi**.

## Decision

### 1. Keyword dan bentuk deklarasi

Func dideklarasikan dengan keyword `func`, **bukan** tag `<func>`.

```txt
FuncDecl ::=
  "func" IDENT FuncParams? "->" Type ":"
  FuncBody
  "/|"
```

Contoh:

```nova
func barHeightClass(bar: ChartBar) -> string:
  ...
/|

func applyExpense(
  ledger: LedgerV1,
  category: string,
  amount: number,
  account: string,
  note: string,
) -> LedgerV1:
  ...
/|

func emptyLedger() -> LedgerV1:
  ...
/|
```

| Elemen | Aturan |
| --- | --- |
| Nama | `IDENT` — `barHeightClass`, `emptyLedger` |
| Parameter | Opsional; jika ada, wajib dalam `(...)` |
| Return type | Wajib eksplisit setelah `->` |
| Pemisah body | Titik dua `:` setelah header (satu baris header atau header multiline) |
| Penutup construct | `/|` setelah body, pada dedent yang sama dengan `func` |

### 2. Parameter list

```txt
FuncParams ::= "(" Param { "," Param } [","] ")"
Param      ::= IDENT ":" Type
```

- Koma antara parameter **diperbolehkan dan disarankan**.
- Trailing comma **diperbolehkan**.
- Tanpa parameter: `func name() -> Type:` — `()` **wajib**.
- Satu parameter per baris **diperbolehkan**; `nova fmt` boleh memformat multiline.

Parameter hanya mendeklarasikan nama dan tipe (ADR-004). Semantik pemanggilan dan pipeline ada di ADR-016.

### 3. Return type

- Tipe data menurut ADR-004: primitif (`string`, `number`, `boolean`), record bernama, union, opaque.
- **Bukan** tipe node UI (`Surface`, `Button`, …) — itu domain `template` (ADR-015).

### 4. Body delimiter (`:`)

- `:` **wajib** segera setelah header (`-> Type` atau setelah `)` jika tanpa return eksplisit tidak diizinkan — return wajib).
- Body dimulai pada baris setelah `:` (indent +1 setelah format).
- `:` di header **bukan** `:` pada named argument di dalam body (konteks parser membedakan).

Isi legal `FuncBody` **tidak** didefinisikan di ADR ini. Satu zona pure: ekspresi tunggal menurut ADR-016.

### 5. Zona pure (deklarasi)

Func adalah construct **pure** (ADR-001). Batas perilaku body (larangan state read, emit, external) tetap mengikat; detail validasi body mengacu ADR-016.

### 6. Modul dan registry

- Func didefinisikan top-level dalam file `.nova` (satu capability per file, ADR-003).
- Beberapa `func` per file diperbolehkan.
- `expr.BuildRegistry` mengumpulkan func dari **module closure** build; import capability menarik modul ke graph, bukan simbol per-func (`import func x` tidak ada di fase ini).
- Nama func harus unik dalam scope registry modul closure (konflik = error).

### 7. Formatting (`nova fmt`)

- `:` selalu di akhir baris header (atau baris terakhir header multiline).
- Body indent satu tingkat dari `func`.
- `/|` pada kolom indent yang sama dengan `func`.

### 8. Migrasi dari syntax lama

| Lama | Baru |
| --- | --- |
| `<func name a: T b: U returns R>` | `func name(a: T, b: U) -> R:` |
| Tag `>` penutup header | `:` |
| Parameter flat tanpa `()` | `()` + koma opsional |

Tidak ada backward compatibility untuk tag `<func>`. Migrasi massal `examples/` dan `tests/conformance/`.

### 9. Diagnostics (stable codes)

| Code | Kondisi |
| --- | --- |
| `nova_func_missing_colon` | Header tanpa `:` sebelum body |
| `nova_func_missing_return` | Tanpa `-> Type` |
| `nova_func_duplicate_name` | Nama func bentrok di registry |
| `nova_func_invalid_return_type` | Return type bukan tipe data (mis. `Surface`) |

Validasi isi body (`nova_func_state_read`, `nova_expr_impure`, …) — ADR-016.

## Consequences

- Lexer: keyword `func` top-level; hapus token `<func`.
- Parser: `parseFunc` baru; `FuncDecl` AST tidak berubah struktural (name, params, return, body tokens).
- Validator: cek `:` dan return type sebelum parse body sebagai ekspresi.
- LSP / fmt: signature `func name(a: T, b: U) -> R`.
- ADR-001: bagian deklarasi `<func>` superseded oleh ADR ini setelah implementasi.
- ADR-016: grammar isi body func; eval/lowering di `internal/core/expr`.

## Alignment

Selaras dengan [design-philosophy.md](../design-philosophy.md) (pure by default, scan-able source) dan [language-design.md](../language-design.md). Header func dan template (ADR-015) memakai pola yang sama; isi body sengaja dipisah per zona.

## Related

- [ADR-015](adr_015_template_declaration.md) — deklarasi `template` (header paralel)
- [ADR-016](adr_016_expression_pipeline.md) — grammar ekspresi, pipeline, special forms
