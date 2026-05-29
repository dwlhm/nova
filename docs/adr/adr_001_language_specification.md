# ADR-001: Language Specification

## Status

Implemented / Accepted

## Depends on

ADR-000

## Scope

- **In scope:** construct, purity, import, state/event/template/lifecycle syntax.
- **Out of scope:** scheduler trace (ADR-002), ViewIR (ADR-005), build (ADR-007).

Ringkasan penulisan: [language-design.md](../language-design.md).

## Decision

Nova adalah bahasa deklarasi aplikasi multi-surface. Bukan general-purpose.

### Tujuh construct

```txt
<import> | <import external>
<contract type> | <contract state> | <contract capability>
<func> | <template> | <lifecycle>
```

Tidak ada: `<state>`, `<action>`, `<effect>`, `<dirty>`, `<use>`, `<bind>`.

| Construct | Zona | Boleh | Tidak boleh |
| --- | --- | --- | --- |
| type, state, func, template, capability | pure | transform, bind, emit `@event` | side effect, external call |
| lifecycle | effect | listen/emit event, external op | tulis state langsung, definisi transisi |
| external import | contract | deklarasi operasi | dipanggil di luar lifecycle |

### Aturan mengikat

1. Satu file `.nova` = satu capability (unit kompilasi).
2. Transisi state hanya di `<contract state>`.
3. Event scheduler berprefix `@`; tanpa payload → `void`.
4. Template hanya baca state/props dan emit `@event`.
5. Dirty hanya di `<lifecycle>` dan runtime commit.
6. Import inert (tidak menjalankan lifecycle).
7. Data antar boundary serializable (ADR-004).

### Import

```nova
<import Counter from "./Counter.nova" /|
<import state count from "./Counter.nova" /|
<import event @increment from "./Counter.nova" /|
<import button from "@nova/ui" /|
```

```nova
<import external storage from "./storage.web.js">
  operation set { input { key: string; value: unknown; } output void; }
 /|
```

External hanya dipanggil dari lifecycle.

### Type & state

Field type: `name: string;` — nilai: `name <- "x";`

```nova
<contract state Counter>
  count: number <- 0 {
    @increment -> count |> add 1;
    @reset -> 0;
  };
/|
```

Transisi = `State × Event → State`; commit oleh scheduler, bukan assignment di template.

Satu `@event` boleh memengaruhi beberapa field state.

### Func

Pure pipeline: `value |> transform arg ...` — tidak baca/tulis state, tidak dispatch event.

### Template

```nova
<template>
  <button on_press -> @increment> + /|
/|
```

Binding data: `attr <- source`. Event: `slot -> @event` atau `-> @event(payload)`.

`<template target <- web>` membatasi target; tanpa target = polymorphic jika dependency mendukung.

### Contract capability

Props + `emits { @name: Type; }`. Pemanggilan: `<Button label <- "x" @pressed -> @save /|`.

### Lifecycle

Fase: `mount`, `dispose`, `before @event`, `after @event`, `error`.

```nova
<lifecycle after @increment>
  count |> storage.set key <- "counter";
 /|
```

Lifecycle tidak diekspor sebagai symbol; tidak di-import sebagai callable.

### Sintaks data

Record `{ a <- 1; b?: string; }`, array `[a, b]`, union `"a" | "b"`, opaque `<contract type Id /|`.

## Consequences

- Construct baru hanya lewat ADR + update `language-design.md` + conformance.
- Grammar lengkap dan edge case parser: lihat implementasi `internal/parser` dan fixture conformance.
