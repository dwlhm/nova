# ADR Implementation Tasks

Checklist gap implementasi (bukan ADR). ADR aktif: **000–011** di `docs/adr/README.md`.

| ADR lama (dihapus) | ADR baru |
| --- | --- |
| 002 scheduler, 005 effect, 014 framework, 018 nav, 019 persistence | **002** runtime |
| 003 capability, 007 platform, 012 layout, 022 packages, 026 modular | **003** modules |
| 006 template, 008 lowering, 023 routing, 024 style, 025 APK, 027 primitives | **005** view |
| 009 external, 013 security | **006** |
| 010 build | **007** |
| 011 diagnostics | **008** |
| 016 web, 017 android | **009** |
| 015 standard packages | **010** |
| 020 test, 021 tooling | **011** |

Bagian di bawah masih memakai nomor lama di judul; arti tugas sama, lihat map di atas.

Status awal:

```txt
Checked: 2026-05-23
Verification:
  go test ./...
  go run ./cmd/nova test
Result:
  Go unit tests pass
  2 conformance fixtures pass
```

## Legend

- `[ ]` belum dikerjakan.
- `[~]` sebagian sudah ada, tapi belum memenuhi kontrak ADR.
- `[x]` sudah cukup memenuhi kontrak ADR yang dimaksud.
- `P0` blocker kontrak desain yang eksplisit belum ada.
- `P1` gap production semantics atau build/runtime utama.
- `P2` gap tooling, conformance, atau kelengkapan lanjutan.

## P0: Belum Implement Sama Sekali

### ADR-027: Renderer Primitive Extension Dictionary

Status: `[x]`

Evidence:

- Toolchain sekarang membaca `renderer.extensions.packages`, resolve package graph, merge renderer
  primitive dictionary, dan meneruskan adapter package ke artifact generation.
- `internal/standard` tetap mengekspos built-in primitives dan menyediakan merge primitive dari
  package graph untuk LSP/tooling.

Tasks:

- [x] Perluas `internal/packages.Manifest` untuk menyimpan `renderer.primitives`, props/events schema,
      target strategy, dan adapter path.
- [x] Tambahkan parser `nova.package.toml` untuk package manifest, termasuk `renderer-package`.
- [x] Perluas `project.ParseManifest` untuk membaca:

  ```toml
  [renderer]
  unknown_kind = "error"

  [renderer.extensions]
  packages = ["@scope/package"]
  ```

- [x] Hubungkan `packages.Resolve` ke pipeline CLI/build agar renderer-package graph tersedia saat
      artifact generation.
- [x] Implement primitive dictionary merge:
      built-in `@nova/ui` > local dictionary override > renderer packages.
- [x] Tambahkan diagnostics:
      `NVA-RENDER-001`, `NVA-RENDER-002`, `NVA-RENDER-003`, `NVA-RENDER-004`.
- [x] Web artifact: import atau inline `platform/web/register.web.js` dari package ter-resolve.
- [x] Android artifact: generate registry/adapter glue untuk `platform/android/Register.android.java`.
- [x] LSP: merge `standard.RendererPrimitives()` dengan primitives dari package graph.
- [x] Tambahkan conformance fixture untuk unknown kind, conflict kind, adapter missing, dan package
      renderer extension happy path.

## P1: Package And Modular Build Integration

### ADR-022 + ADR-026: Package Distribution, Lockfile, Modular System

Status: `[~]`

Yang sudah ada:

- `internal/packages` punya model package, lockfile, resolver deterministic, target adapter check,
  permission sources, dan lock audit in-memory.
- `internal/standard` mengekspos package resmi `@nova/*` dan `@env/*`.

Gap:

- Pipeline CLI/build utama belum memanggil `packages.Resolve`.
- Belum ada parser file `nova.package.toml`.
- Belum ada parser file `nova.lock`.
- `nova.toml` belum punya dependency/package root yang dipakai resolver.
- Package graph belum menjadi input `internal/build.Resolve`.
- Package import `@scope/package/path` belum resolve ke source package graph.

Tasks:

- [ ] Tambahkan model dependency root di `project.Manifest`.
- [ ] Parse dependency/package section di `nova.toml`.
- [ ] Implement parser/loader `nova.lock`.
- [ ] Implement parser/loader `nova.package.toml`.
- [ ] Integrasikan `standard.OfficialPackages()` dan package manifests project/registry ke
      `packages.Resolve`.
- [ ] Jalankan `packages.Resolve` dari `internal/cli.runProjectPipeline` sebelum `build.Resolve`.
- [ ] Teruskan resolved package graph ke `internal/build`.
- [ ] Resolve package imports di module graph, bukan hanya local relative imports.
- [ ] Fail production build bila lockfile missing, hash mismatch, atau target adapter tidak dipin.
- [ ] Tambahkan inspect output untuk package graph, lock digest, adapter path, dan permission source.
- [ ] Tambahkan conformance fixture untuk package graph deterministic, version conflict, unsupported
      target, dan lock drift.

### ADR-003 + ADR-010: Capability/Module Graph Completeness

Status: `[~]`

Yang sudah ada:

- Local module graph dari entry source sudah ada.
- Capability manifest generation sudah ada.
- Acyclic graph validator ada di `internal/capability`.

Gap:

- Acyclic capability graph validator belum dipanggil di pipeline utama.
- Type/state/event graph lintas file masih terbatas.
- Package module graph belum menjadi satu graph dengan local modules.
- Diagnostics belum selalu menyertakan candidate files/reason rejected seperti ADR-010.

Tasks:

- [ ] Panggil `capability.ValidateAcyclicModuleGraph` dari build pipeline.
- [ ] Tambahkan diagnostics stabil untuk cycle module graph.
- [ ] Gabungkan local modules dan package modules dalam satu build graph.
- [ ] Validasi imported state/event lintas file dengan payload/type contract asal.
- [ ] Tambahkan candidate/context diagnostics untuk missing package/local module.
- [ ] Tambahkan conformance untuk import alias lintas file, module cycle, missing import, dan
      package import.

## P1: Runtime Semantics And External Operations

### ADR-002 + ADR-005: Production Scheduler Lifecycle Parity

Status: `[~]`

Yang sudah ada:

- Go reference scheduler punya before/after/mount/dispose/error lifecycle, atomic commit,
  error routing, dan external operation request.
- Production JS/Java scheduler punya queue, sequence, nested drain protection, dan state commit.

Gap:

- Production JS/Java scheduler belum menjalankan lifecycle phases.
- Production JS/Java scheduler belum memiliki external operation completion path.
- Generated artifact belum membawa lifecycle handler IR yang executable.
- Error lifecycle belum wired ke production runtime.

Tasks:

- [ ] Tambahkan lifecycle metadata ke generated app model / SchedulerIR.
- [ ] Generate executable lifecycle handlers untuk web runtime.
- [ ] Generate executable lifecycle handlers untuk Android MainActivity/runtime.
- [ ] Implement phase order production:
      before lifecycle -> transition commit -> after lifecycle -> pending events.
- [ ] Implement mount/dispose lifecycle di web runtime.
- [ ] Implement mount/dispose lifecycle di Android runtime.
- [ ] Route transition/lifecycle/runtime error ke error lifecycle bila tersedia.
- [ ] Pastikan lifecycle output tidak enqueue ketika handler gagal.
- [ ] Tambahkan parity tests JS/Java terhadap Go scheduler trace.
- [ ] Tambahkan conformance fixture lifecycle before/after/error/mount/dispose.

### ADR-009: External Interop Runtime Bridge

Status: `[~]`

Yang sudah ada:

- Parser external import dan operation contract.
- Semantic validator menolak external call di pure zones.
- Build resolver memilih target implementation row dan validasi input/output contract.
- Go `internal/effect` punya external completion helper.

Gap:

- Web artifact belum memuat atau memanggil adapter JS project/package.
- Android artifact belum memanggil adapter Java project/package.
- `NovaExternalBindings.java` baru berisi list operation names.
- Completion event/failure event belum menjadi bagian runtime production.
- Runtime output validation belum dilakukan di JS/Java production runtime.

Tasks:

- [ ] Definisikan generated ExternalOperationTable dalam app IR.
- [ ] Web: copy/import selected `platform/web/*.web.js` ke artifact.
- [ ] Web: implement adapter invocation, Promise handling, output validation, failure mapping.
- [ ] Android: copy/generate selected `platform/android/*.android.java` ke project source.
- [ ] Android: generate binding method calls dari operation table.
- [ ] Android: run adapter work off main thread bila operation blocking.
- [ ] Support success/failure completion event mapping.
- [ ] Route adapter error ke failure event atau SchedulerError.
- [ ] Reject non-serializable/native output before enqueue.
- [ ] Tambahkan adapter contract tests untuk success, failure, invalid output, and permission denial.

### ADR-013: Runtime Security And Permission Enforcement

Status: `[~]`

Yang sudah ada:

- Build-time permission audit untuk external operations.
- Host event validation helper ada di `internal/security`.
- Serializable value validation tersedia di `internal/types`.

Gap:

- Host event validation belum wired ke production JS/Java runtime.
- Runtime permission denial belum observable sebagai event/error.
- Permission scopes dari manifest belum dipakai enforcement.
- Package permission source chain belum surfaced di build/inspect.
- AndroidManifest permission entries belum digenerate dari permission plan.

Tasks:

- [ ] Generate EventContract table untuk production runtime.
- [ ] Web: validate event name, payload shape, and allowed emitter before enqueue.
- [ ] Android: validate event name, payload shape, and allowed emitter before dispatch.
- [ ] Enforce disabled/permission-denied controls before dispatch.
- [ ] Implement permission scoped metadata in `project.Manifest`.
- [ ] Add permission source chain to `permissions.json` and `nova inspect`.
- [ ] Web: map permissions to browser capability checks and denial diagnostics.
- [ ] Android: generate required manifest permission entries from build plan.
- [ ] Android: adapter-level permission prompt/denial route.
- [ ] Add security conformance for undeclared event, invalid payload, missing permission, package
      permission expansion, and forbidden native output.

## P1: Standard Surface And Renderer Completeness

### ADR-015: Standard Package Surface

Status: `[~]`

Yang sudah ada:

- `internal/standard` lists official packages and primitive metadata.
- Web/Android artifacts support basic `text`, `button`, `surface`, `row`, `column`, `stack`, `page`.
- Accessibility warning helper exists for interactive primitive labels.

Gap:

- `@nova/core` helper packages are metadata only, not resolved source packages.
- `@nova/forms` primitives are listed but not fully rendered.
- `list`, `item`, `image`, `scroll`, `spacer`, `radio_group`, `checkbox`, `field` are incomplete or
  fallback behavior.
- Common props/events like `role`, `enabled`, `visible`, `test_id`, `on_focus`, `on_blur`,
  `on_long_press` are incomplete.
- Accessibility metadata is not fully wired into artifact validation/conformance.

Tasks:

- [ ] Define source package content or built-in contracts for `@nova/core`.
- [ ] Validate primitive props/events against `standard.RendererPrimitives()`.
- [ ] Web: implement `image`, `scroll`, `list`, `item`, and `spacer`.
- [ ] Web: implement forms primitives and DOM events `input`, `change`, `submit`, focus, blur.
- [ ] Web: implement `enabled`/`disabled`, `visible`, `role`, `test_id`, and accessibility attrs.
- [ ] Android: implement native `ImageView`, `ScrollView`, list projection, spacer.
- [ ] Android: implement `EditText`, numeric input, Switch, SeekBar, select-like primitive,
      checkbox/radio alternatives.
- [ ] Android: map accessibility label/contentDescription, enabled, visible, role where possible.
- [ ] Add standard package conformance fixtures for primitive metadata, a11y warnings, event mapping,
      and target parity.

### ADR-006 + ADR-008: ViewIR And Lowering Completeness

Status: `[~]`

Yang sudah ada:

- ViewIR tree, props/events, binding dependency metadata, route/page metadata.
- Basic renderer invalidation via state dependency metadata.

Gap:

- List projection and item scope are not fully implemented.
- Conditional projection syntax/IR is not standardized or implemented.
- Capability component projection is incomplete.
- SourceSpan is not carried through all IR nodes.
- TargetIR stage is implicit in artifact generation, not a first-class contract.

Tasks:

- [ ] Implement list/item projection semantics and `item` scope.
- [ ] Add key stability diagnostics for list projection.
- [ ] Decide final conditional syntax in ADR/spec before implementation.
- [ ] Implement conditional ViewIR node and renderer behavior.
- [ ] Implement capability component projection or explicitly mark out of current production scope.
- [ ] Add SourceSpan to ViewIR nodes, bindings, and event routes.
- [ ] Emit source map entries per ViewIR node/event route.
- [ ] Introduce explicit target IR structs only if needed by artifact boundary.
- [ ] Add conformance for ViewIR shape, source spans, event routes, list keys, and conditional
      projection.

## P1: Target Runtime Gaps

### ADR-016: Web Target Runtime

Status: `[~]`

Yang sudah ada:

- CSR static web artifact.
- Browser JS scheduler library embedded into artifact.
- Basic DOM renderer and route/page History API integration.
- Style asset injection for global/scoped CSS.

Gap:

- External adapter runtime missing.
- Browser permission checks/denials missing.
- DOM event mapping is mostly `click -> on_press`.
- Diagnostic overlay, runtime trace panel, permission audit panel, and source-mapped runtime errors
  are missing.
- Hydration/SSR contract is not implemented.

Tasks:

- [ ] Implement web external adapter loader/invoker.
- [ ] Add runtime event payload validation before enqueue.
- [ ] Expand DOM event mapping for input/change/submit/focus/blur/keyboard where contracted.
- [ ] Prevent disabled controls from dispatching events.
- [ ] Implement renderer error routing to SchedulerError/error lifecycle.
- [ ] Add runtime trace capture in dev mode.
- [ ] Add optional diagnostic overlay in dev mode.
- [ ] Add permission audit panel or inspect endpoint in dev server.
- [ ] Add hydration manifest format before implementing SSR/SSG.
- [ ] Add web runtime tests for DOM event routes, permission denial, external completion, and route
      history behavior.

### ADR-017 + ADR-025: Android Target Runtime

Status: `[~]`

Yang sudah ada:

- Java native View artifact generation.
- Gradle project generation.
- Android scheduler Java library embedded as module.
- Basic MainActivity renderer and route back stack.
- No heavy UI framework dependency is introduced by generated app code.

Gap:

- Activity/process lifecycle mapping to `@nova/app` events is incomplete.
- Android permission manifest/runtime prompt behavior is incomplete.
- External adapter bridge is incomplete.
- Deep link intent handling is incomplete.
- Snapshot restoration/configuration change behavior is incomplete.
- Renderer primitive coverage is incomplete.
- Instrumented renderer/permission tests are missing.

Tasks:

- [ ] Generate Activity lifecycle callbacks for `@app_started`, `@app_resumed`, `@app_paused`,
      `@app_stopped`.
- [ ] Implement root mount/dispose lifecycle behavior.
- [ ] Generate AndroidManifest permission entries from build permissions.
- [ ] Implement runtime permission request/denial route in adapters.
- [ ] Generate and invoke Java external adapter bindings.
- [ ] Add deep link intent parsing to route data.
- [ ] Add saved-state snapshot serialization/restoration path.
- [ ] Preserve scheduler state across configuration change where possible.
- [ ] Complete native View mapping for `image`, `scroll`, `list`, and forms primitives.
- [ ] Add JVM and instrumented tests for renderer, permission bridge, lifecycle, and restoration.

## P2: Persistence, App Lifecycle, And Navigation

### ADR-018: App Lifecycle And Navigation Model

Status: `[~]`

Yang sudah ada:

- Route state and page projection work for basic web/android cases.
- Internal app package defines lifecycle/navigation events and pure navigation stack helpers.

Gap:

- Production runtimes do not consistently emit `@nova/app` lifecycle events.
- Programmatic navigation framework capability is incomplete.
- Deep link handling is incomplete.
- Route validation/event contracts are not fully enforced by production runtime.

Tasks:

- [ ] Add activation/import contract for `@nova/app` lifecycle events.
- [ ] Web: map page load, visibilitychange, pagehide/pageshow to scheduler events.
- [ ] Android: map Activity/process lifecycle to scheduler events.
- [ ] Implement programmatic navigation event/action flow.
- [ ] Validate route payload shape before commit.
- [ ] Add conformance for browser back, Android back, deep link, and route params.

### ADR-019: State Persistence And Hydration

Status: `[~]`

Yang sudah ada:

- Go `internal/persistence` has snapshot capture/restore contract and validation.

Gap:

- Persistence policy is not parsed from `nova.toml`.
- Runtime web/android do not capture/restore HydrationSnapshot.
- `@app_restored` is not emitted by production runtimes.
- Web hydration/SSR not implemented.
- Android saved instance/process death restoration not implemented.

Tasks:

- [ ] Add `[state]` and `[state.persist]` manifest model and parser.
- [ ] Generate persistence metadata into artifact.
- [ ] Web: capture and restore snapshot from configured storage/session source.
- [ ] Android: capture compact snapshot metadata in saved instance state.
- [ ] Android: restore scheduler state after process recreation when snapshot is valid.
- [ ] Enqueue `@app_restored(snapshot)` when applicable.
- [ ] Add redaction metadata support for future sensitive annotations.
- [ ] Add persistence conformance for invalid snapshot, version mismatch, missing state, and sequence
      continuation.

## P2: Tooling And Conformance

### ADR-020: Testing And Conformance Strategy

Status: `[~]`

Yang sudah ada:

- Fixture runner exists.
- Scheduler trace comparison exists for Go reference.
- Current fixture set passes.

Gap:

- Fixture layout is narrow compared to ADR target.
- Web target harness for DOM/event/external/permission/hydration is missing.
- Android instrumented harness is missing.
- `nova test` does not run unit tests, target smoke tests, adapter contract tests, or runtime target
  tests.
- Compatibility matrix is not emitted per release/artifact.

Tasks:

- [ ] Expand `tests/conformance/` into language, scheduler, view, target, security, external, artifacts.
- [ ] Add fixture schema fields for external stubs, diagnostics spans, state snapshots, and target
      smoke metadata.
- [ ] Add web runtime harness for DOM event route and external completion.
- [ ] Add Android JVM/instrumented harness for renderer and permission bridge.
- [ ] Make `nova test` select and run target-specific fixture subsets.
- [ ] Add expected trace generation/check mode.
- [ ] Emit compatibility matrix metadata in artifacts or release docs.

### ADR-021: Developer Tooling And Hot Reload

Status: `[~]`

Yang sudah ada:

- `nova init/check/build/dev/test/inspect/fmt/lsp` commands exist.
- Web dev server polls files and triggers browser full reload.
- Android dev mode builds, installs, and launches debug APK.
- Hot reload decision helper exists in `internal/tooling`.

Gap:

- Hot reload ABI diff planner is not wired into dev mode.
- Web dev lacks overlay, runtime trace panel, event/state inspector, and permission audit panel.
- Android dev lacks runtime diagnostic bridge and source-mapped errors.
- `nova inspect` is missing full capability manifest, event route table, state cells, and package graph.

Tasks:

- [ ] Wire ABI diff detection into dev cycle.
- [ ] Use `internal/tooling.PlanHotReload` to decide patch/reload/rebuild.
- [ ] Add state-preserving reload path for compatible web patches.
- [ ] Add dev diagnostic overlay for web.
- [ ] Add event/state inspector and trace stream in web dev mode.
- [ ] Add permission audit panel or endpoint in web dev mode.
- [ ] Add Android runtime diagnostic bridge or documented fallback.
- [ ] Expand `nova inspect` with capability manifests, ViewIR metadata, event routes, state cells,
      package graph, lock digest, and permission source chain.

## P2: Documentation Status Cleanup

### ADR Status Drift

Status: `[~]`

Observation:

- Some ADRs are marked `Implemented / Accepted` while major runtime/package portions are still partial.
- ADR-024 and ADR-025 are marked `Accepted`, but their implementation is already substantially present.

Tasks:

- [ ] Split ADR index status into precise values:
      `Implemented`, `Partial`, `Accepted Design`, `Out of Production v1 Scope`.
- [ ] Mark ADR-027 as `Accepted Design / Not Implemented`.
- [ ] Mark ADR-022 and ADR-026 as `Partial` until CLI/build package integration lands.
- [ ] Mark ADR-009, ADR-013, ADR-016, ADR-017, ADR-018, ADR-019, ADR-020, ADR-021 as `Partial`
      until runtime/conformance gaps are closed.
- [ ] Consider marking ADR-024 and ADR-025 as `Implemented / Accepted` after a focused verification pass.
- [ ] Keep `docs/adr/README.md` aligned with this task file after implementation milestones.
