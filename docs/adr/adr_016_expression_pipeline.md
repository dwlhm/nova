# ADR-016: Expression syntax and pipeline

## Status

Accepted (locked)

## Depends on

ADR-014, ADR-015, ADR-017

## Scope

- **In scope:** grammar ekspresi pure (func body, state transition, template binding), pipeline `|>`,
  builtin expression forms (`if`, `match`, `map`, `filter`), anonymous functions di pipeline,
  shorthand pipeline, inferensi tipe anon, aturan purity per zona, diagnostics.
- **Out of scope:** deklarasi `func` / `template` header (ADR-014, ADR-015); grammar isi template
  (node DSL); lowering runtime detail; migrasi massal `examples/` / conformance.

## Context

ADR-014 mengunci **deklarasi** named `func`. Grammar isi body dan pipeline — ADR ini.
Evaluasi/lowering ada di `internal/core/expr` (legacy [ADR-013](adr_013_pure_expressions.md), deprecated).

Keputusan desain:

1. Func body = **expression-oriented** (satu nilai keluar); bukan statement imperative.
2. Pipeline `|>` = **dataflow** antar step; nilai kiri mengalir ke step kanan.
3. **Named `func`** = kontrak manual (registry); **anonymous function** di pipeline = kontrak
   site-local yang **compiler infer** dari tipe predecessor.
4. **`->` hanya** untuk return type named func dan anonymous function; **`:`** untuk tipe param,
   cabang `if` / `match`, dan pembuka body named func.
5. Token **`$` tidak dipakai** — binding piped value via shorthand implicit atau `(param) ->`.

## Decision

### 1. Dua kontrak: named func vs pipeline anon

| | Named `func` | Pipeline anonymous |
| --- | --- | --- |
| Signature | Author: `func name(a: T) -> R:` | Compiler infer dari step kiri |
| Registry | `expr.BuildRegistry` | Tidak — ephemeral, inline lower |
| `->` | Return type di header | `(params) -> body` (hanya di pipeline) |
| Reuse | Cross-module via import closure | Hanya di site pipeline |

Named func analog **type definition manual**. Anon di pipeline analog **inline contract** yang
compiler synthesize dari aliran data — param tidak dideklarasikan aneh di scope luar.

### 2. Token dan peran

| Token | Peran |
| --- | --- |
| `:` | Tipe param (`x: number`), field type, pembuka cabang (`if`, `match`), pembuka body named func |
| `->` | Return type named func (`func … -> R:`); **anon func** `(params) -> body` di pipeline |
| `\|>` | Pipeline step |
| `<-` | Named argument (record field, external input) — bukan return |

### 3. Named func body

```txt
FuncBody ::= Expr
```

- Body = **satu ekspresi root** (record, call, pipe, `if`, `match`, …).
- **Ternary `? :` deprecated** — gunakan `if` / `elif` / `else` dengan `:`.
- Prefix call tanpa pipe tetap valid: `applyExpense ledger category amount account note`.

```nova
func visibleLine1(ledger: LedgerV1, filter: HistoryFilter) -> string:
  if filter == "expense":
    ledger.expenseRecent1
  elif filter == "income":
    ledger.incomeRecent1
  elif filter == "transfer":
    ledger.transferRecent1
  else:
    ledger.line1
/|
```

**Zona purity (func):** no state read/write, no `@event`, no external call.

### 4. Builtin expression forms (non-pipeline)

Construct dengan syntax dan evaluator sendiri — **bukan** pemanggilan func registry.

| Form | Syntax | Catatan |
| --- | --- | --- |
| Conditional | `if` / `elif` / `else` + `:` | Setiap cabang = satu expr |
| Match | `match` discriminant `:` + arms + `:` | Arm pattern + expr |
| Primary | literal, ident, record, list, field access | — |
| Call | named func, prefix atau dalam pipe | Lihat §5 |

**Sebutan di spec:** *builtin expression forms* atau *special forms*.

Combinator `map:` / `filter:` **hanya** valid sebagai pipeline stage (§5), bukan di root expr
func body kecuali sebagai bagian `PipeExpr`.

### 5. Pipeline

```txt
PipeExpr  ::= PrefixExpr { "|>" PipeStage }

PipeStage ::= [StageKeyword ":"] PipeBody

StageKeyword ::= "map" | "filter" | "if" | "match"

PipeBody ::=
    ShorthandBody
  | AnonFunc

AnonFunc ::= "(" ParamList ")" "->" Expr

ParamList ::= Param { "," Param }
Param     ::= IDENT [ ":" Type ]
```

Setiap `|>` **mem-bind** nilai kiri ke step kanan. Special form keyword menentukan evaluator;
tanpa keyword = **transform** (default).

#### 5.1 Shorthand (tanpa `(param) ->`)

Allowed bila step **hanya return satu hal** dan binding piped value **implisit jelas**.

| Bentuk | Contoh | Desugar |
| --- | --- | --- |
| Unary call | `ledger \|> encodeLedger` | `encodeLedger(ledger)` |
| Multi-arg call | `ledger \|> applyExpense(draftCategory, draftAmount, fromAccount, note)` | arg1 = piped value |
| Unary expr | `inputNumber \|> + 2` | `inputNumber + 2` |
| Combinator expr | `items \|> map: item.category` | `map` dengan implicit `item` |
| Combinator expr | `items \|> filter: item.amount > 0` | `filter` dengan implicit `item` |
| External primary | `ledger \|> storage.set(key <- "k")` | piped value → slot primary (`value`) |

- **Implicit first argument** untuk call multi-arg setelah `|>`.
- **Implicit element binder** untuk `map:` / `filter:` shorthand: nama default `item` (infer dari
  free ident di expr jika memungkinkan).
- **Deprecate / forbid:** whitespace-only first arg (`ledger \|> applyExpense draftCategory …`).

#### 5.2 Anonymous function (wajib)

Pakai `(params) -> body` bila shorthand **tidak cukup**:

- Body kompleks (nested `if` / `match`, banyak referensi piped value).
- Pipeline `if:` / `match:` dengan cabang multi-arm.
- Explicit param type override: `(row: unknown) -> …` — **hanya** di pipeline (semua special forms).
- Multi-param combinator (future `fold:`): `(acc, item) -> …`.

```nova
ledger |> (l) ->
  if l.transactionCount == 0:
    emptyLedger()
  else:
    l

rows |> map: (row: unknown) -> storedLine1 row
```

**Anon `(params) ->` dilarang** di luar pipeline (func body root, template attr, dll.).

#### 5.3 Special forms di pipeline

| Keyword | Input kiri | Output | Shorthand |
| --- | --- | --- | --- |
| *(none)* | `T` | `U` | unary call / expr / multi-arg call |
| `map:` | `T[]` | `U[]` | `map: item.field` |
| `filter:` | `T[]` | `T[]` | `filter: item.active` |
| `if:` | `T` | `U` | — (anon wajib) |
| `match:` | `T` | `U` | — (anon wajib) |

Body `if:` / `match:` pipeline memakai cabang `:` di dalam anon, sama family dengan §4.

#### 5.4 External operation (lifecycle / effect zone)

```nova
ledger |> encodeLedger
ledger |> storage.set(key <- "finance_ledger_v1")
```

- Primary input slot operation (mis. `value` pada `set`) menerima piped value **implicit** jika
  tidak ditulis.
- **`value <- ledger` redundant** saat piped value sudah `ledger` → `nova_pipe_redundant_binding`.
- Operasi tanpa pipe receiver (mis. `load` dari `void`): named inputs only, no `|>` receiver.

Chain pure transform → single external step diperbolehkan; hindari multi-hop effect script.

### 6. Type inference (anonymous params)

- Compiler infer tipe param anon dari tipe **predecessor** pipeline step.
- `items |> map: item.name` → `item` = element type of `items`.
- `ledger |> (l) -> …` → `l` = type of `ledger`.
- Explicit `(row: unknown)` hanya saat predecessor `unknown` atau infer gagal.

Validator: body anon must type-check against inferred param; return step must chain to next step.

### 7. Zona konteks (purity & akses)

Grammar expr **sama**; validator per zona:

| Konteks | State read | Payload | Func registry | External |
| --- | --- | --- | --- | --- |
| func body | ✗ | params only | ✓ | ✗ |
| state transition | ✓ | ✓ | ✓ | ✗ |
| template binding | ✓ | props | ✓ | ✗ |
| lifecycle | ✓ (read) | — | ✓ (pure steps) | ✓ (effect step) |

### 8. Prefix vs pipeline

Keduanya **equivalent** semantik; pipe = optional compose sugar.

```nova
applyExpense ledger draftCategory draftAmount fromAccount note
ledger |> applyExpense(draftCategory, draftAmount, fromAccount, note)
```

### 9. Diagnostics (stable codes)

| Code | Kondisi |
| --- | --- |
| `nova_expr_impure` | state / emit / external di zona pure |
| `nova_expr_ternary_deprecated` | `? :` ternary |
| `nova_pipe_implicit_arg_deprecated` | first arg via whitespace after `\|>` |
| `nova_pipe_redundant_binding` | named input duplicates piped value |
| `nova_pipe_anon_outside_pipeline` | `(p) ->` di luar pipeline |
| `nova_pipe_shorthand_ambiguous` | shorthand tidak bisa infer binder |
| `nova_pipe_type_mismatch` | output step N ≠ input step N+1 |
| `nova_anon_param_unresolved` | field/ident tidak ada di inferred type |
| `nova_func_return_mismatch` | body expr type ≠ header `-> Type` |

Prefix `nova_expr_` / `nova_pipe_` selaras ADR-008.

### 10. Explicitly forbidden

- Token `$` sebagai pipe placeholder.
- Ternary `? :` (migrate ke `if`).
- `(params) ->` di luar pipeline.
- Implicit first-arg pipe via whitespace only.
- Anon dengan explicit types di luar pipeline.

## Examples (normative)

```nova
func bump(n: number) -> number:
  n |> + 2
/|

func labels(items: Item[]) -> string[]:
  items
    |> filter: item.amount > 0
    |> map: item.category
/|

func ledgerFromRows(rows: unknown[]) -> string[]:
  rows |> map: (row: unknown) -> storedLine1 row
/|

# state transition
@save_expense ->
  ledger
    |> applyExpense(draftCategory, draftAmount, fromAccount, note)
    |> encodeLedger

# lifecycle
<lifecycle after @save_expense>
  ledger |> storage.set(key <- "finance_ledger_v1")
/|
```

## Consequences

- **Lexer:** implicit `$` **tidak** ditambahkan; optional unary expr token untuk `|> + 2`.
- **Parser:** `PipeStage` dengan shorthand vs `AnonFunc`; deprecate ternary; combinator keywords.
- **Validator:** zona purity; pipe chain typing; shorthand binder resolution.
- **Lowering:** shorthand desugar ke anon/internal temps; combinators → runtime helpers;
  named func → `NovaExpr.*` (`internal/core/expr`).
- **ADR-014 / ADR-015:** body grammar; deklarasi header unchanged.
- **Migrasi:** finance ternary → `if`; audiolab pipe whitespace → paren/shorthand.

## Alignment

Selaras [design-philosophy.md](../design-philosophy.md) (pure by default, expression-oriented) dan
[language-design.md](../language-design.md). Named func untuk logic reusable; pipeline + anon
 untuk dataflow site-local; lifecycle pipe untuk effect receiver.

## Related

- [ADR-013](adr_013_pure_expressions.md) — eval/lowering legacy (deprecated)
- [ADR-014](adr_014_func_declaration.md) — deklarasi named func
- [ADR-015](adr_015_template_declaration.md) — deklarasi template; expr di binding argumen
- [ADR-017](adr_017_contract_declarations.md) — `state` transisi, `lifecycle` expr
