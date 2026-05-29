# ADR-004: Type System

## Status

Implemented / Accepted

## Depends on

ADR-001

## Scope

- **In scope:** tipe data Nova, assignability, serializable boundary.
- **Out of scope:** diagnostic message format (ADR-008).

## Decision

Strict data-oriented types. Nilai di state, event, props, func, template, external contract =
**serializable Nova data value**.

### Primitif & komposit

```txt
string | number | boolean | unknown | void | null
literal union | array | record | opaque nominal type
```

Tidak ada implicit coercion (`"1"` bukan `number`).

### Opaque & alias

```nova
<contract type UserId /|
<contract type Status> "idle" | "loading" | "error";
```

### void vs null

- `void`: event/operation tanpa payload; emit `void -> @increment;`
- `null`: nilai data opsional.

### Assignability

Record structural; field opsional `?`; union narrowing di transisi mengikuti validator.

### Batas port

Handle platform (DOM node, JNI ref, dll.) tidak boleh masuk state/event. Adapter marshal ke
data value sebelum masuk scheduler.

## Consequences

- Perubahan assignability → `internal/types` + fixture type diagnostic.
