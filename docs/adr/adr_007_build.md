# ADR-007: Build & Target Resolution

## Status

Deprecated — referensi historis build pipeline. Spesifikasi bahasa aktif: [ADR-014](adr_014_func_declaration.md)–[ADR-016](adr_016_expression_pipeline.md).

## Depends on

ADR-003, ADR-005

## Scope

- **In scope:** input/output build, target id, manifest.
- **Out of scope:** CLI flags detail (ADR-011).

## Decision

### Target id production

```txt
web | android
```

Future: `ios`, `desktop` — butuh adapter + conformance.

### Input

```txt
nova.toml, entry capability, target id, source graph, package lock, provider bindings
```

### Output (per target)

```txt
validated graph
scheduler / state IR
ViewIR + route table
external + permission tables
provider bindings
target artifact (bundle)
diagnostics
```

Resolusi dependency **target-aware**: capability tanpa implementasi untuk target aktif = error
build.

### Validasi bertingkat

- Lokal cepat (watch/dev).
- CI penuh sebelum release.

Device: load artifact + apply; tidak parse `.nova`.

## Consequences

- `internal/build`, `internal/target`, `internal/artifact` ownership per `architecture.md`.
