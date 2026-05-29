# Nova Language Design

Panduan normatif untuk menulis `.nova`. Arah arsitektur:
[design-philosophy.md](design-philosophy.md). Spesifikasi mengikat ringkas:
[adr_001_language_specification.md](adr/adr_001_language_specification.md).

---

## Apa itu Nova

Bahasa deklarasi aplikasi multi-surface — bukan general-purpose. Kamu mendeklarasikan:

- **keadaan** (`<contract state>`) dan aturan perubahannya (`@event`),
- **tampilan** (`<template>`),
- **efek ke luar** (`<lifecycle>`, `<import external>`, `@env/*`).

Satu file `.nova` = satu **capability** (unit kompilasi, dependency, audit izin).

---

## Loop yang harus kamu pegang

```txt
snapshot state
  -> render (template + provider)
  -> user/platform input
  -> @event
  -> transisi pure di <contract state>
  -> commit snapshot
  -> <lifecycle> bila perlu (storage, network, …)
  -> render lagi
```

**State** = penanda akibat interaksi. **Event** = pemicu, bukan lapisan terpisah. Urutan
commit dan lifecycle dijamin runtime (ADR-002); tidak perlu construct khusus untuk scheduler.

---

## Enam prinsip

| # | Prinsip | Praktik |
| --- | --- | --- |
| 1 | **Zona eksplisit** | Pure di contract/func/template; effect hanya lifecycle & external |
| 2 | **Pure by default** | Tanpa side effect di func/template/transisi |
| 3 | **Event, bukan aksi tersembunyi** | Perilaku lewat `@event` + handler state |
| 4 | **File = capability** | Satu concern utama per file |
| 5 | **Import inert** | Import hanya graph dependency; tidak menjalankan lifecycle |
| 6 | **Serializable** | Payload state/event/ABI tanpa handle platform |

---

## Tujuh construct

| Construct | Zona | Fungsi |
| --- | --- | --- |
| `<import>` / `<import external>` | deklarasi | dependency; kontrak operasi luar |
| `<contract type>` | pure | bentuk data |
| `<contract state>` | pure | state + `@event → hasil` |
| `<contract capability>` | pure | props + event yang di-emit komponen |
| `<func>` | pure | transform data (`\|>`) |
| `<template>` | pure | view + routing event ke `@event` |
| `<lifecycle>` | effect | listen event, panggil external/`@env` |

Tidak ada `<state>`, `<action>`, `<effect>`, `<dirty>`, `use`, `bind`. Router native tidak
ada di bahasa — pakai state `route` + `<page>` (ADR-005).

---

## Zona di source

```txt
┌──────────────────────────────────────┐
│ Pure: type, state, capability,       │
│       func, template                 │
└──────────────────┬───────────────────┘
                   │ snapshot, @event
┌──────────────────▼───────────────────┐
│ Effect: <lifecycle>                  │
└──────────────────┬───────────────────┘
                   │ ports
┌──────────────────▼───────────────────┐
│ Provider + @env/* (render, IO)       │
└──────────────────────────────────────┘
```

---

## Menulis per construct

### Import

```nova
<import Counter from "./Counter.nova" /|
<import state count from "./Counter.nova" /|
<import event @increment from "./Counter.nova" /|
<import button from "@nova/ui" /|
```

External = dirty; hanya dipanggil dari lifecycle:

```nova
<import external storage from "@env/storage">
  operation set {
    input { key: string; value: unknown; }
    output void;
  }
/|
```

### Type

```nova
<contract type Route>
  path: string;
  params?: unknown;
/|
```

Tipe field: `name: string;` — nilai default: `name <- "x";`

### State

```nova
<contract state Counter>
  count: number <- 0 {
    @increment -> count + 1;
    @decrement -> count - 1;
    @reset -> 0;
  };
/|
```

- Transisi hanya di sini, bukan di lifecycle/template.
- Satu `@event` boleh mengubah beberapa field.
- Tanpa payload: `@tick: void` — emit `void -> @tick;` di lifecycle.

### Func

```nova
<func add value: number amount: number returns number>
  value + amount
/|
```

Pipeline: `count |> add 1`. Func tidak baca state, tidak dispatch event, tidak panggil external.

### Template

```nova
<template>
  <button on_press -> @increment>
    <text value <- "+" /|
  /|
/|
```

- Binding: `attr <- expr`
- Event: `on_press -> @event` atau `-> @event(payload)`
- Baca state/props; jangan panggil external.

**Routing** — state `route` + `<page>`:

```nova
<page path <- "/settings">
  <text value <- "Settings" /|
/|
```

Perubahan route lewat transisi, mis. `@route_changed(next: Route) -> next`.

### Capability (komponen)

```nova
<contract capability Button>
  props { label: string; disabled?: boolean; }
  emits { @pressed: void; }
/|

<template>
  <button disabled <- disabled on_press -> @pressed>
    <text value <- label /|
  /|
/|
```

Pemakaian: `<Button label <- "Save" @pressed -> @save /|`

### Lifecycle

```nova
<import event @restore from "./Store.nova" /|
<import state exportJson from "./Store.nova" /|

<lifecycle mount>
  void -> @restore;
/|

<lifecycle after @save>
  exportJson |> storage.set key <- "ledger" value <- exportJson;
/|
```

Fase: `mount`, `dispose`, `before @event`, `after @event`, `error`. Lifecycle tidak callable
dan tidak menulis state langsung.

---

## Checklist review

1. Satu concern utama per file.
2. Semua perubahan state di `<contract state>`.
3. Event publik: prefix `@`; tanpa payload → `void`.
4. Template: bind + emit saja.
5. Storage/network/device → lifecycle atau `@nova/*` yang sudah memodelkan event sama.
6. Platform API → `<import external>` / `@env/*`.
7. Import state/event lintas capability; hindari state global duplikat.

---

## Konvensi nama

| Konsep | Konvensi | Contoh |
| --- | --- | --- |
| Event | `@snake_case` | `@increment`, `@route_changed` |
| State / type contract | `PascalCase` | `Counter`, `Route` |
| File capability | `PascalCase.nova` | `Counter.nova` |
| Route (v1) | field `route` | lihat ADR-005 |

---

## ADR terkait

| Pertanyaan | ADR |
| --- | --- |
| Spec mengikat & edge grammar | 001 |
| Runtime, lifecycle, persistence | 002 |
| Module, layout, package | 003 |
| Tipe | 004 |
| ViewIR, `<page>`, renderer | 005 |
| External, permission | 006 |
| Build | 007 |
| Diagnostic codes | 008 |
| Target web/Android | 009 |
| `@nova/*`, `@env/*` | 010 |
| CLI, conformance | 011 |

---

## Menambah construct atau syntax

1. Pastikan tidak bisa diekspresikan dengan tujuh construct + package/provider.
2. Update dokumen ini dan ADR-001; tambah fixture conformance.
3. Jangan buka dirty zone userland baru tanpa ADR.

Lihat [architecture.md](architecture.md) untuk ownership paket Go.
