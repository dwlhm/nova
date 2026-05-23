Berikut ADR yang fokus ke **state persistence, restoration, dan hydration** Nova.

---

# ADR-019: State Persistence & Hydration Model

## Status

Implemented / Accepted

## Context

Nova memiliki scheduler state yang hanya boleh berisi Nova data.

Target awal membutuhkan restoration:

```txt
web:
  reload
  back-forward cache
  SSR/SSG hydration
  offline storage

android:
  configuration change
  process death restoration
  saved instance state
  DataStore persistence
```

ADR-009 sudah mengatur external storage sebagai dirty operation, tetapi framework masih membutuhkan
kontrak umum untuk snapshot scheduler, hydration, dan restoration.

---

# Decision

Nova memakai **serializable scheduler snapshot** sebagai kontrak persistence/hydration.

```txt
HydrationSnapshot {
  abiVersion: string
  target: TargetId
  appInstanceId: string
  sequence: LogicalSequence
  stateCells: StateCellSnapshot[]
  route?: Route
  metadata: SnapshotMetadata
}
```

Snapshot hanya boleh berisi:

```txt
Nova data value
opaque value with serializable representation
logical sequence metadata
contract version metadata
```

Snapshot tidak boleh berisi:

```txt
DOM node
Android View state
Android Context
Promise/Future/Coroutine
external adapter object
function/closure
file/socket/native handle
```

---

# Snapshot Ownership

Scheduler runtime adalah pemilik snapshot semantic.

Adapter boleh:

```txt
request snapshot
persist snapshot
restore snapshot candidate
report snapshot failure
```

Adapter tidak boleh:

```txt
mutate snapshot state cells directly
invent state cell not present in SchedulerIR
skip validation because snapshot came from local device
restore native object into Nova state
```

---

# Snapshot Shape

```txt
StateCellSnapshot {
  owner: CapabilityRef
  name: StateName
  type: TypeRef
  value: DataValue
  version?: string
}

SnapshotMetadata {
  createdAt?: string
  source: "runtime" | "ssr" | "storage" | "android_saved_state" | "devtools"
  languageVersion: string
  schedulerVersion: string
  viewIrVersion: string
  projectVersion?: string
}
```

`createdAt` jika ada adalah data observability. Ia tidak boleh dipakai menentukan event ordering.

---

# Restoration Flow

Canonical restoration:

```txt
load artifact
load snapshot candidate
validate ABI and semantic versions
validate state cell identity
validate state value type
create scheduler with restored state
enqueue @app_restored(snapshot) if @nova/app active
render restored view
run mount lifecycle according to target policy
```

Rule:

```txt
1. Invalid snapshot is rejected with diagnostic.
2. Partial restore is disabled by default.
3. Migration must be explicit.
4. Restore does not replay old events.
5. New events receive new LogicalSequence after restored sequence.
```

---

# State Persistence Policy

Persistence is explicit.

Recommended manifest:

```toml
[state]
persistence = "explicit"

[state.persist]
AudioLab.collection = "local"
Router.route = "session"
```

Exact syntax may change, but semantic policy must express:

```txt
which state cells can persist
where they persist
retention scope
migration policy
sensitivity/redaction policy
```

Default:

```txt
no state persistence except runtime restoration needed by target shell
```

---

# Web Hydration

Web SSR/SSG flow:

```txt
server/compiler evaluates pure template with initial state
emit HTML
emit hydration manifest
browser loads runtime
runtime validates manifest
attach EventRouteTable
restore snapshot
resume scheduler
```

Rule:

```txt
1. Server-rendered HTML is not semantic source of truth.
2. Hydration manifest must include ViewIR identity and source map metadata.
3. Hydration mismatch is diagnostic NVA-RENDER-*.
4. Dirty lifecycle mount policy must be explicit:
   - run_on_client
   - already_ran_on_server
   - skip_until_event
5. Event before hydration is either buffered or blocked by runtime config.
```

Production web may ship CSR only, but ABI must not block SSR/hydration later.

---

# Android Restoration

Android restoration flow:

```txt
configuration change
  -> retain scheduler instance when possible
  -> remount renderer only

process death
  -> load saved HydrationSnapshot
  -> validate snapshot
  -> create scheduler
  -> render restored state
  -> enqueue @app_restored
```

Rule:

```txt
1. Configuration change does not imply Nova state reset.
2. Saved instance state may store only compact snapshot metadata.
3. Larger durable state uses @env/storage/DataStore through explicit persistence policy.
4. Android Bundle object is not Nova data.
```

---

# Migration

State migration is explicit and versioned.

Future migration contract:

```txt
fromVersion
toVersion
stateCell
pure migration function
fallback policy
```

Production v1:

```txt
1. If state cell type changes incompatibly, snapshot restore fails.
2. If state cell disappears, snapshot entry is ignored only when policy permits.
3. If new state cell appears, initial value from contract state is used.
```

---

# Sensitive Data

Snapshot persistence must respect ADR-013.

Rule:

```txt
1. Sensitive data is not persisted by default.
2. Devtools snapshot should redact sensitive fields when annotation exists.
3. Secrets should live in secure platform adapter, not scheduler state.
4. Persisted snapshot should include permission/audit source metadata.
```

Sensitive annotation is future work, but snapshot contract must allow redaction metadata.

---

# Consequences

Keuntungan:

```txt
Web hydration dan Android restoration memakai contract yang sama
snapshot dapat diuji tanpa platform
persistence tetap eksplisit dan auditable
Nova data boundary tetap bersih
```

Trade-off:

```txt
state migration butuh desain lanjutan
partial restore sengaja dibatasi pada production v1
adapter perlu mengelola storage platform tanpa membocorkan object native
```
