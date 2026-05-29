# ADR-008: Diagnostics

## Status

Implemented / Accepted

## Depends on

ADR-001

## Scope

- **In scope:** bentuk diagnostic, severity, stable codes.
- **Out of scope:** CLI UX (ADR-011).

## Decision

### Shape

```txt
Diagnostic { code, severity, message, span, related }
```

Severity: `error` | `warning` | `info`. Build gagal pada `error`.

### Stable codes

Prefix per domain (contoh): `nova_parse_*`, `nova_type_*`, `nova_view_*`, `nova_build_*`,
`nova_perm_*`. Kode tidak berubah semantik tanpa ADR.

### Output

- Human-readable ke stderr / IDE.
- JSONL untuk tooling (`internal/diagnostic`).

Sort deterministik (file, span, code).

## Consequences

- Kode baru = entri di registry diagnostic + test snapshot.
