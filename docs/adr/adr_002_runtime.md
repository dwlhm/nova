# ADR-002: Runtime (Scheduler, Effect, Navigation, Persistence)

## Status

Implemented / Accepted

## Depends on

ADR-001

## Scope

- **In scope:** urutan runtime semantic, lifecycle, route state, snapshot/hydration.
- **Out of scope:** implementasi DOM/View (ADR-009), ViewIR (ADR-005).

## Context

Semua target memakai loop yang sama: event → transisi pure → commit → lifecycle → invalidate
render. Platform hanya berbeda di provider (listen/render/IO).

## Decision

### Lapisan runtime

```txt
Nova runtime (per target): muat ABI, scheduler, transisi, lifecycle dispatch
Provider: render ViewIR, normalisasi input → event, effect ports
```

Go `internal/scheduler` / `internal/app` = referensi conformance, bukan runtime produksi.

### Urutan event (wajib)

```txt
1. Event masuk (envelope: sequence, source capability, @name, payload)
2. Evaluasi transisi pure (semua state terdampak)
3. Commit atomik snapshot
4. Lifecycle (before/after/mount/dispose/error) sesuai tabel effect
5. Invalidasi render dari snapshot
6. Provider render
```

Urutan event oleh `LogicalSequence` scheduler, bukan timestamp platform.

### Event envelope

```txt
EventEnvelope { sequence, source: CapabilityRef, name: @event, payload: DataValue | void }
```

Payload harus serializable dan sesuai kontrak event.

### Lifecycle

Satu dirty zone userland. Memanggil `<import external>` dan `@env/*` hanya dari lifecycle
(setelah commit kecuali fase `before` yang didefinisikan di implementasi scheduler).

Lifecycle tidak menulis state; emit event mengikuti import event.

### Navigation

Route = state serializable. Konvensi production v1: field `route` (string atau record `Route`).

```nova
<contract state Router>
  route: Route <- { path <- "/" } {
    @route_changed(next: Route) -> next;
  };
/|
```

Adapter tidak menulis `route` langsung; perubahan lewat `@route_changed` atau event setara.

### Persistence & hydration

Snapshot state serializable disimpan/dimuat lewat effect port (storage). Hydration: initial
snapshot dari artifact atau provider (SSR) → runtime apply → render.

Format snapshot dan versi schema didefinisikan di kontrak artifact; migrasi breaking = bump versi
artifact.

### Error

Error lifecycle atau effect → event/error class terdaftar; state tidak corrupt commit setengah.
Detail kode diagnostic: ADR-008.

## Consequences

- Perubahan urutan commit wajib update conformance trace (ADR-011).
- Runtime native: `runtime/nova-scheduler-js`, `runtime/nova-scheduler-java`.
