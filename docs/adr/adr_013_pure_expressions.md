# ADR-013: Pure expression evaluation and lowering

## Status

Deprecated — referensi historis eval/lowering (`internal/core/expr`). Spesifikasi aktif: [ADR-014](adr_014_func_declaration.md)–[ADR-017](adr_017_contract_declarations.md).

## Depends on

ADR-001, ADR-002

## Decision

Nova pure zones (`<func>`, `<contract state>` transitions, template bindings) share one expression semantics:

1. **Parse** tokens to an AST (`internal/core/expr`) per grammar ADR-016.
2. **Evaluate** deterministically as `(snapshot, payload) → value` in conformance/scheduler reference.
3. **Lower** to `NovaExpr.*` calls plus `state.*` / `payload.*` for web and Android runtimes.
4. **Emit** `NovaExpr` helpers from `<func>` bodies at bundle/artifact time (no I/O, no state reads inside func).

Pipeline shorthand and anonymous functions desugar before lowering. Legacy prefix call arity (non-final
prefix atoms; final arg spans rest) remains until parser migrates to ADR-016 paren forms.

## Consequences

- Android `NovaRuntime.evaluate` must not echo unknown expressions as UI text.
- Finance and other examples may use `<func>` in transitions without workarounds.
- Architecture allows `internal/core/expr` imports from `ir`, `conformance`, and view runtime emit.

## Related

- [ADR-014](adr_014_func_declaration.md)–[ADR-017](adr_017_contract_declarations.md) — spesifikasi aktif
