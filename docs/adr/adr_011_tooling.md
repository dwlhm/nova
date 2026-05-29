# ADR-011: Tooling, Test, Conformance

## Status

Implemented / Accepted

## Depends on

ADR-002, ADR-007, ADR-009

## Scope

- **In scope:** CLI, dev loop, hot reload, conformance, test layers.
- **Out of scope:** implementasi LSP (future).

## Decision

### CLI (`cmd/nova` → `internal/cli`)

```txt
nova build [target]
nova test
nova fmt
nova dev          # watch + dev server
```

Semantic rules di package core, bukan di CLI.

### Test

```txt
go test ./...              # compiler, validator, artifact
nova test                  # fixture .nova + conformance
```

Unit: parser, types, transition pure. Integration: build graph. Conformance: replay event trace,
bandingkan snapshot + effect schedule vs golden.

### Hot reload

Dev mode: rebuild increment capability terdampak; inject snapshot state bila aman; invalidasi
render. Batas: perubahan breaking ABI memerlukan full restart.

### Provider conformance

Provider custom dapat diuji dengan trace fixture yang sama dengan referensi untuk capability
yang di-handle.

## Consequences

- Fixture di `tests/` / `internal/conformance`; perubahan scheduler → update golden.
