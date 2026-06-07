# ADR-017: Contract and module declarations

## Status

Accepted (locked)

## Depends on

ADR-014, ADR-015, ADR-016

## Scope

- **In scope:** deklarasi top-level `type`, `state`, `emits`, `component`, `external`, `lifecycle`,
  `import`; type system minimal; aturan modul; purity per zona; hubungan `component` ↔ `template`.
- **Out of scope:** header/body `func` (ADR-014), header `template` (ADR-015), grammar ekspresi
  (ADR-016), isi body template (node DSL); lowering runtime; migrasi massal examples.

## Context

Grammar legacy (`<contract type>`, `<contract state>`, `<contract capability>`, tag XML) tidak
selaras dengan dialect ADR-014/015 (`keyword … -> Type:` / `keyword Name:` + `/|`).

Keputusan desain:

1. Satu file `.nova` = satu **modul** (unit kompilasi); identitas modul = path file, bukan nama
   block `capability`.
2. **`emits`** — event publik modul, **tanpa nama**.
3. **`component`** — kontrak widget (`props` + `events`), **bernama**; mengganti
   `<contract capability>` untuk UI; memudahkan LSP/import.
4. **`state`** — **boleh banyak** per file; snapshot runtime **flat** (field unik lintas block).
5. Ekspresi di transisi, initial, lifecycle — ADR-016.

## Decision

### 1. Daftar construct top-level

| Construct | Named? | Zona | Peran |
| --- | --- | --- | --- |
| `import` | per simbol | deklarasi | Graph dependency, bind lintas modul |
| `type` | ya | pure | Bentuk data serializable |
| `func` | ya | pure | Transform pure (ADR-014) |
| `state` | ya, plural/file | pure | Snapshot + transisi `@event` |
| `emits` | tidak | deklarasi | Event surface modul |
| `component` | ya | pure | Kontrak props + events widget |
| `template` | ya | pure | Proyeksi UI (ADR-015) |
| `external` | ya | contract | Port operasi platform |
| `lifecycle` | tidak (phase) | effect | Listen event / mount; panggil external |

Penutup construct: `/|` pada indent yang sama dengan keyword pembuka.

Urutan deklarasi yang disarankan (validator atau `nova fmt` boleh reorder):

`import` → `type` → `func` → `emits` → `state` → `component` → `template` → `external` →
`lifecycle`.

### 2. Token bersama (dengan ADR-014/016)

| Token | Peran |
| --- | --- |
| `:` | Pembuka body construct; tipe field/param; cabang `if` / `match` |
| `->` | Return type `func` / `template`; transisi `@event ->` expr; anon `(p) ->` di pipeline |
| `<-` | Nilai initial field state; named arg record/external |
| `\|>` | Pipeline (ADR-016) |
| `/\|` | Penutup construct top-level |

### 3. `type`

```txt
TypeDecl ::=
  "type" IDENT ":"
  TypeBody
  "/|"

TypeBody ::=
    FieldList
  | UnionType
  | "opaque"
```

```nova
type Route:
  path: string
  params?: unknown
  query?: unknown
  fragment?: string
/|

type HistoryFilter:
  "all" | "expense" | "income" | "transfer"
/|

type LedgerV1:
  version: number
  transactionCount: number
  netTotal: number
/|

type Surface:
  opaque
/|
```

| Aturan | Detail |
| --- | --- |
| Field | `name: Type` — optional field `name?: Type` |
| Union | Literal string dipisah `\|` |
| Opaque | `opaque` — boundary ABI (node UI, blob eksternal) |
| Default | Tidak di `type` — hanya di `state` |

**NodeType** template (`Surface`, `Button`, …) = `type` opaque dari package `@nova/ui` atau
registry primitive.

### 4. `state`

```txt
StateDecl ::=
  "state" IDENT ":"
  StateField { StateField }
  "/|"

StateField ::=
  IDENT ":" Type [ "<-" Initial ] [ TransitionBlock ]

TransitionBlock ::=
  "{"
  Transition { Transition }
  "}"

Transition ::=
  "@" IDENT [ EventParams ] "->"
  Expr

Initial ::= Expr
```

```nova
state Router:
  route: Route <- { path: "/expense" }
  {
    @route_changed(next: Route) ->
      next
    @choose_expense ->
      { path: "/expense" }
  }
/|

state Ledger:
  ledger: LedgerV1 <- emptyLedger()
  {
    @save_expense ->
      ledger
        |> applyExpense(draftCategory, draftAmount, fromAccount, note)
    @ledger_hydrated(value: unknown) ->
      ledgerFromStored(value)
  }
/|

state Draft:
  draftCategory: string <- "Makan"
  draftAmount: number <- 0
  fromAccount: string <- "Tunai"
  note: string <- ""
/|
```

| Aturan | Detail |
| --- | --- |
| Plural per file | Beberapa `state Name:` diperbolehkan |
| Snapshot | **Flat** — semua field semua block = satu snapshot |
| Uniqueness | Nama field **unik lintas** semua `state` dalam file |
| Nama block | Hanya **grouping** authoring; tidak namespaced di runtime |
| Initial | `<- Expr` — ADR-016 |
| Transisi | `@event -> Expr` — satu ekspresi = nilai field baru |
| Akses expr | Boleh baca field block lain + payload event (ADR-016 §7) |
| Purity | Transisi pure — no external, no emit |
| Field tanpa `{ }` | Hanya initial; tidak ada handler event |

Semantik: `State × @event → State'` per field; commit oleh scheduler.

### 5. `emits`

```txt
EmitsDecl ::=
  "emits" ":"
  "{" EventSig { "," EventSig } [","] "}"
  "/|"

EventSig ::=
  "@" IDENT [ "(" ParamList ")" ] ":" Type
```

```nova
emits:
  {
    @route_changed(next: Route): Route,
    @save_expense: void,
    @ledger_hydrated(value: unknown): void,
  }
/|
```

| Aturan | Detail |
| --- | --- |
| Nama | **Tidak ada** — identitas modul = path file |
| `void` | Event tanpa payload |
| Validasi | Setiap `@event` di transisi `state` atau `lifecycle` wajib terdaftar di `emits` modul
  yang sama (atau re-export policy fase lanjut) |
| vs `component` | `emits` = surface **modul**; bukan props widget |

Satu block `emits` per file (fase ini).

### 6. `component`

```txt
ComponentDecl ::=
  "component" IDENT ":"
  "props:" FieldList
  "events:" EventSigList
  "/|"

EventSigList ::=
  "{" EventSig { "," EventSig } [","] "}"
  | "{" "}"
```

```nova
component TabButton:
  props:
    class: string
    label: string
    on_press: Event
  events:
    {
      @pressed: void,
    }
/|

component Button:
  props:
    label: string
    class: string
    disabled?: boolean
    on_press: Event
  events:
    {}
/|
```

| Aturan | Detail |
| --- | --- |
| Nama | **Wajib** — import & LSP symbol |
| `props` | Input widget — `name: Type` |
| `events` | Event yang komponen **boleh emit** ke parent |
| `Event` | Tipe opaque untuk slot routing (`on_press -> @save`) |
| `events: {}` | Diperbolehkan — komponen input-only |

**LSP / internal:** symbol `component TabButton` dengan anak `props.*`, `events.@*`.

Hubungan `template`:

- Template yang mengimplementasikan component: parameter header **match** `props` component.
- Import: `import TabButton from "./Widgets.nova"`.

```nova
component TabButton:
  props:
    class: string
    label: string
    on_press: Event
  events:
    { @pressed: void, }
/|

template TabButton(class: string, label: string, on_press: Event) -> Button:
  ...
/|
```

### 7. `external`

```txt
ExternalDecl ::=
  "external" IDENT "from" ImportPath ":"
  Operation { Operation }
  "/|"

Operation ::=
  "operation" IDENT ":"
  "input:" FieldList
  "output:" Type
```

```nova
external storage from "@env/storage":
  operation load:
    input:
      key: string
    output: unknown

  operation set:
    input:
      key: string
      value: unknown
    output: void
/|
```

| Aturan | Detail |
| --- | --- |
| Panggilan | **Hanya** dari `lifecycle` |
| Pipeline | ADR-016 §5.4 — `ledger |> storage.set(key <- "k")` |

### 8. `lifecycle`

```txt
LifecycleDecl ::=
  "lifecycle" LifecyclePhase ":"
  LifecycleBody
  "/|"

LifecyclePhase ::=
  "mount" | "dispose"
  | "before" "@" IDENT
  | "after" "@" IDENT
  | "error"
```

```nova
lifecycle mount:
  void -> @restore
/|

lifecycle after @restore:
  void
    |> storage.load(
         key <- "finance_ledger_v1",
         onSuccess <- @ledger_hydrated,
         onFailure <- @load_failed,
       )
/|

lifecycle after @save_expense:
  ledger |> storage.set(key <- "finance_ledger_v1")
/|
```

| Aturan | Detail |
| --- | --- |
| Body | Expr / pipeline — ADR-016 |
| Emit | `void -> @event` atau via external completion |
| Import | `import state ledger`, `import event @save_expense` dari modul lain |

### 9. `import`

```nova
import FinancePure from "./FinancePure.nova"
import state ledger, route, draftCategory from "./FinanceStore.nova"
import event @save_expense, @ledger_hydrated from "./FinanceStore.nova"
import Button, Surface from "@nova/ui"
import TabButton from "./Widgets.nova"
```

| Kind | Syntax |
| --- | --- |
| Modul | `import Path from "…"` |
| State field | `import state f1, f2 from "…"` |
| Event | `import event @e1, @e2 from "…"` |
| Component / UI | `import Name from "…"` — resolves `component` + template closure |

Import **inert** — tidak menjalankan lifecycle.

Tidak ada `import func` — func via module closure (ADR-014).

### 10. Type system minimal

| Kind | Syntax |
| --- | --- |
| Primitif | `string`, `number`, `boolean`, `void` |
| Named | `LedgerV1`, `Route` |
| Union | `"a" \| "b"` |
| Optional field | `name?: T` |
| Array | `T[]` |
| Unknown | `unknown` — external / hydrate |
| Opaque | `opaque` pada `type` |
| Event slot | `Event` — hanya di `component.props` |

Pemisahan return:

- `func … -> LedgerV1` — data type (ADR-014)
- `template … -> Surface` — opaque node type (ADR-015)

### 11. Zona purity

| Zona | Tulis state | Emit `@` | External | Expr |
| --- | --- | --- | --- | --- |
| `func` | ✗ | ✗ | ✗ | ADR-016 pure |
| `state` transisi | ✓ (read) | ✗ | ✗ | ADR-016 |
| `template` | read | route slot | ✗ | ADR-016 di binding |
| `lifecycle` | read | ✓ | ✓ | ADR-016 |
| `component` | — | deklarasi only | — | — |

### 12. Legacy mapping (tidak ada backward compatibility)

| Lama | Baru |
| --- | --- |
| `<contract type X>` | `type X:` … `/\|` |
| `<contract state X>` | `state X:` … `/\|` (plural OK) |
| `<contract capability X>` emits only | `emits:` … `/\|` |
| `<contract capability Button>` props+emits | `component Button:` … `/\|` |
| `<import external …>` | `external name from "…":` … `/\|` |
| `<lifecycle after @e>` | `lifecycle after @e:` … `/\|` |
| `<import … /|>` | `import …` (tanpa tag) |

### 13. Diagnostics (stable codes)

| Code | Kondisi |
| --- | --- |
| `nova_type_duplicate_name` | Nama `type` bentrok |
| `nova_state_duplicate_field` | Field bentrok lintas `state` block |
| `nova_state_unknown_event` | `@event` transisi tidak di `emits` |
| `nova_state_transition_impure` | external/emit di transisi |
| `nova_emits_duplicate_event` | `@event` ganda di `emits` |
| `nova_emits_orphan` | `emits` event tanpa handler (warning) |
| `nova_component_duplicate_name` | Nama `component` bentrok |
| `nova_component_template_props_mismatch` | Template param ≠ component props |
| `nova_external_outside_lifecycle` | Panggil external di func/state |
| `nova_lifecycle_unknown_event` | Phase `@event` tidak dikenal |
| `nova_import_unknown_symbol` | Import simbol tidak ada di modul sumber |

## Examples (normative module split)

**Pure library** (`FinancePure.nova`):

```nova
type LedgerV1: …
/|

func emptyLedger() -> LedgerV1: …
/|

func applyExpense(ledger: LedgerV1, category: string, amount: number, account: string, note: string) -> LedgerV1: …
/|
```

**Store** (`FinanceStore.nova`):

```nova
import FinancePure from "./FinancePure.nova"

emits:
  { @save_expense: void, @route_changed(next: Route): Route, }
/|

state Router: …
/|

state Ledger: …
/|

state Draft: …
/|
```

**Persistence** (`FinancePersistence.nova`):

```nova
emits:
  { @ledger_synced: void, }
/|

external storage from "@env/storage":
  operation set:
    input:
      key: string
      value: unknown
    output: void
/|

import state ledger from "./FinanceStore.nova"
import event @save_expense from "./FinanceStore.nova"

lifecycle after @save_expense:
  ledger |> storage.set(key <- "finance_ledger_v1")
/|
```

## Consequences

- Lexer: keyword `type`, `state`, `emits`, `component`, `external`, `lifecycle`; hapus tag
  `<contract …>`.
- Parser: AST per construct; plural `state`; `emits` tanpa IDENT.
- Validator: flat snapshot uniqueness; emits ↔ transitions; component ↔ template props.
- LSP: symbol tree `component` → props/events; `emits` sebagai modul metadata.
- Manifest: modul ref = file path; hilangkan `ContractCapabilityDecl.Name` untuk emits-only.
- Migrasi: examples/conformance rewrite ke syntax baru.

## Alignment

Selaras ADR-014/015/016. Model · View · Effect: `state` + `emits` = model; `template` + `component`
= view contract; `lifecycle` + `external` = effect.

## Related

- [ADR-014](adr_014_func_declaration.md) — `func`
- [ADR-015](adr_015_template_declaration.md) — `template`
- [ADR-016](adr_016_expression_pipeline.md) — ekspresi di transisi / lifecycle
