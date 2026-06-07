# Finance example & pure expression runtime — progress log

Resume from the **last completed checkpoint** (search for latest `done`). Do not restart platform work if CP-1..CP-4 are `done`.

**Principles:** pure functional Nova (ADR-001/002/013), no backward compat, single `ledger` source of truth (CP-5+).

---

## Checkpoints

| ID | Status | Summary | Verified by |
|----|--------|---------|-------------|
| CP-0 | `done` | Progress log + [adr_013_pure_expressions.md](adr/adr_013_pure_expressions.md) | doc exists |
| CP-1 | `done` | `internal/core/expr` parse/eval/lower | `go test ./internal/core/expr/...` |
| CP-2 | `done` | IR lowering + `NovaExprJS` in bundle | `go test ./internal/core/ir/...` |
| CP-3 | `done` | Android `NovaRuntime.evaluate` + `NovaExpr.java` emit | `go test ./internal/provider/capability/view/codegen/android/...` |
| CP-4 | `done` | Web `nova-runtime.js` includes `NovaExpr` | artifact build (nova build) |
| CP-5 | `done` | `FinancePure` (types+reducers) + `FinanceStore` `ledger` SSOT | `nova build` in `examples/finance`, `nova test` |
| CP-6 | `done` | Persistence `finance_ledger_v1` + `@ledger_hydrated` | `nova build`, `nova test` |
| CP-7 | `done` | formatCurrency labels, active classes, copy, export panel removed | `nova build`, `nova test` |
| CP-8 | `done` | Conformance fixture `tests/conformance/web/finance` | `go run ./cmd/nova test` |

---

## Session log

### 2026-06-03 — session 1

**Done**

- Added `internal/core/expr` (AST, parse, pure eval, JS/Java lower, emit).
- Conformance scheduler/lifecycle uses expr registry from all modules.
- `expressionToJS` lowers func calls to `NovaExpr.*`.
- Bundle carries `NovaExprJS`; web runtime appends it.
- Android emits `NovaExpr.java`, `NovaExprInvoke.java`; extended `NovaRuntime.evaluate`.
- `GenerateInput.Sources` for artifact emit.
- Architecture map updated for `internal/core/expr`.
- ADR-013 drafted.
- Removed debug `bar-visual` bindings from `App.nova`.

**Next (resume here)**

- Finance example checkpoints CP-0..CP-8 complete. Optional follow-ups: scheduler trace fixture (blocked on `emptyLedger()` initial eval in conformance runtime), Gradle device build.

### 2026-06-03 — session 4 (CP-7)

**Done**

- `netTotalLabel`, `incomeTotalLabel`, `expenseTotalLabel`, `feeTotalLabel`, `transferTotalLabel` via `formatCurrency` di store.
- Kelas aktif: `tab-button-selected`, `type-button-selected`, `filter-button-selected` (+ state class di `FinanceStore`).
- Label tab/filter tanpa prefix `●`; highlight lewat class.
- Panel Export JSON dihapus dari `App.nova`.
- Copy UX: saldo bersih, riwayat, komposisi arus kas, hint transfer.

**Verified:** `nova build` dari `examples/finance`; `nova test`.

### 2026-06-03 — session 5 (CP-8)

**Done**

- Fixture `tests/conformance/web/finance/` (salinan `nova.toml` + `src/*.nova` + style dari `examples/finance`).
- `nova.conformance.json`: artifact 4 modul, `storage.read`/`storage.write`, 4 halaman routing, 65 state cells, 76 bindings.

**Verified:** `go run ./cmd/nova test` (11 fixtures).

### 2026-06-03 — session 3 (CP-6)

**Done**

- `FinancePersistence.nova`: satu `storage.load` / `storage.set` pada kunci `finance_ledger_v1` (nilai = state `ledger`); rantai 10× load dihapus.
- Event `@ledger_hydrated(value)` menggantikan `@loaded*` + `@ledger_ready`; terminal `@ledger_synced` untuk Android hydration codegen.
- `FinancePure.nova`: `ledgerFromStored` + akses field dari blob JSON (object hasil `JSON.parse` di web storage).
- Semua field store disinkronkan dari `ledgerFromStored value` pada hydrate.

**Verified:** `nova build` dari `examples/finance`; `go test ./internal/core/ir/...`; `nova test`.

### 2026-06-03 — session 2 (CP-5)

**Done**

- `FinancePure.nova`: `LedgerV1`, `emptyLedger`, `applyExpense`/`applyIncome`/`applyTransfer`, `ledgerFromStored`, `visibleLine*`, `encodeLedger`, chart/pie helpers.
- `FinanceStore.nova`: state `ledger` as SSOT; flat fields (`netTotal`, `allLine*`, …) synced from `apply*` on save and `@ledger_ready`; removed inline funcs and `barVisual` state.
- Types duplicated in `FinancePure` + `FinanceStore` (per-file semantic checker does not resolve capability-imported types yet).

**Verified:** `go run ../../cmd/nova build .` from `examples/finance`; `go run ./cmd/nova test`.

---

## Key files (platform)

| Area | Path |
|------|------|
| Expr core | `internal/core/expr/*.go` |
| IR lower | `internal/core/ir/expr_lower.go`, `bundle.go` |
| Conformance | `internal/conformance/expr_eval.go`, `scheduler_runtime.go` |
| Web emit | `internal/provider/capability/view/external/runtime/web_emit.go` |
| Android emit | `internal/provider/capability/view/external/runtime/android_emit.go` |
| Android eval | `internal/provider/capability/view/codegen/android/runtime_java.go` |

---

## Verify commands

```bash
go test ./internal/core/expr/... ./internal/conformance/... ./internal/core/ir/...
go run ./cmd/nova build examples/finance --target android   # needs network for Gradle on device CI
go run ./cmd/nova test
```

---

## Known issues

- Capability `<import … from "./Other.nova">` does not export types into the importer’s type env; duplicate `contract type` per module until semantic import typing lands.
- Gradle `assembleDebug` may fail in sandbox (network/IP); Nova compile to `build/android` still succeeds.
- `NVA-STYLE-007` on finance dynamic `class` bindings (pie/bar segments) — pre-existing; needs stateful style binding (CP-7/T2.1 from plan).
