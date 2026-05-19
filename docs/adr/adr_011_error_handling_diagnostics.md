Berikut ADR yang fokus ke **error handling dan diagnostics** Nova.

---

# ADR-011: Error Handling & Diagnostics

## Status

Implemented / Accepted

## Context

Nova harus memberi error yang jelas di dua fase:

```txt
build-time diagnostics
runtime scheduler errors
```

Karena Nova menjaga semantic lintas platform, error tidak boleh bergantung pada detail platform
yang tidak stabil. Error juga harus dapat dipetakan kembali ke source `.nova`.

---

# Decision

Nova membedakan:

```txt
diagnostic compile/build
runtime scheduler error
external/adapter error
renderer error
host error
```

Compile/build diagnostic tidak masuk scheduler.
Runtime error diarahkan melalui scheduler error model dan `error` lifecycle jika memungkinkan.

---

# Diagnostic Shape

Diagnostic build minimal:

```txt
Diagnostic {
  code: string
  severity: error | warning | info
  message: string
  file?: string
  span?: SourceSpan
  related?: RelatedSpan[]
  hint?: string
}
```

Diagnostic harus:

```txt
menunjuk source span jika tersedia
memakai kode stabil
menjelaskan expected vs actual
memberi related location untuk konflik/import
tidak bergantung pada urutan map/hash runtime
```

---

# Diagnostic Codes

Kode diagnostic memakai prefix:

```txt
NVA-LEX-*       lexer
NVA-PARSE-*     parser
NVA-TYPE-*      type checker
NVA-PURITY-*    purity/effect validation
NVA-EVENT-*     event contract
NVA-MODULE-*    module/capability graph
NVA-TARGET-*    target resolution
NVA-SEC-*       security/permission
NVA-RENDER-*    renderer lowering
NVA-RUNTIME-*   runtime validation
```

Contoh:

```txt
NVA-PURITY-001: func cannot call external operation storage.set
NVA-EVENT-003: event @loaded payload does not match declared type
NVA-TARGET-004: no implementation for @env/storage on ios
```

---

# Severity

Severity:

```txt
error    build cannot continue or runtime operation failed
warning  build can continue but behavior may be risky
info     tool/devtools information
```

Rule:

```txt
1. Type mismatch is error.
2. Dirty operation in pure zone is error.
3. Missing target capability is error.
4. Event cycle without visible guard may be warning.
5. List without stable key may be warning.
```

---

# Runtime Error Shape

Runtime scheduler error:

```txt
SchedulerError {
  source: CapabilityRef
  phase: SchedulerPhase
  event?: SchedulerEvent
  state?: StateKey
  message: string
  cause?: unknown
}
```

Phase:

```txt
enqueue
transition
lifecycle_mount
lifecycle_dispose
lifecycle_before
lifecycle_after
lifecycle_error
external
renderer
host
```

---

# Error Routing

Runtime error routing:

```txt
transition error -> cancel commit for active event -> error lifecycle
lifecycle error  -> no rollback -> error lifecycle
external error   -> failure event or error lifecycle
renderer error   -> error lifecycle and host adapter notification
host error       -> host diagnostic or error lifecycle if app context exists
```

Rule:

```txt
1. Transition error prevents state commit for that event.
2. Lifecycle error does not rollback already committed state.
3. Error lifecycle has recursion guard.
4. Error emitted recovery event goes through normal queue.
5. Fatal host/runtime error may stop app instance.
```

---

# Transition Error

Transition can fail due to:

```txt
runtime validation failure
missing state snapshot
pure expression evaluation error
type contract violation
```

If transition error occurs:

```txt
1. no candidate state is committed for active event
2. renderer invalidation is not produced for failed commit
3. error lifecycle receives SchedulerError
4. scheduler continues if runtime remains healthy
```

---

# Lifecycle Error

Lifecycle error can happen in:

```txt
mount
dispose
before @event
after @event
error
```

Rule:

```txt
1. Error in before lifecycle does not cancel event in MVP.
2. Error in after lifecycle does not rollback commit.
3. Error in error lifecycle is guarded and reported to host.
4. Lifecycle output events produced before failure are adapter/runtime-defined only if execution semantics allow it.
```

Preferred implementation:

```txt
Lifecycle output is committed after handler returns successfully.
```

This avoids partial lifecycle output.

---

# External Error

External error sources:

```txt
operation rejected
permission denied
adapter exception
output validation failed
timeout
target API unavailable
```

Expected domain failures should be typed events.

```txt
@load_failed(message: string)
@permission_denied(permission: string)
```

Unexpected failures may become `SchedulerError`.

---

# Renderer Error

Renderer error sources:

```txt
missing primitive
invalid target IR
binding evaluation failure
platform render failure
event route mismatch
```

Renderer error must include source span if possible.

Renderer error must not mutate scheduler state.

---

# Host Error

Host error happens before or around scheduler enqueue.

Example:

```txt
invalid platform payload
event route missing
app instance disposed
queue overload
```

If scheduler app context exists, host error should route to error lifecycle.
If not, host reports platform diagnostic.

---

# User-Facing Diagnostics

Diagnostic message style:

```txt
short main message
exact source location
expected vs actual
one concrete hint when possible
related spans for imported declarations
```

Example:

```txt
NVA-PURITY-001: func cannot call external operation storage.set
  src/Counter.nova:12:11
  external operations are only valid in lifecycle blocks
```

---

# Source Mapping

All diagnostic-capable stages should preserve:

```txt
file
line
column
token span
generated IR node id
target artifact span when available
```

Runtime errors in generated target code should map back to `.nova` source if possible.

---

# Alternatives Considered

## Exceptions in Nova Source

Expose try/catch or exception throwing in Nova.

Rejected for MVP because:

```txt
Scheduler event/error routing already defines failure flow.
Exceptions would complicate pure transition semantics.
Cross-target exception behavior differs.
```

## Fail-Silent Runtime

Runtime logs errors but keeps app going without routing.

Rejected because:

```txt
Apps need recovery lifecycle.
Silent failure hides target adapter bugs.
Diagnostics become weak.
```

## Platform-Native Error Types in State

State stores platform error objects directly.

Rejected because:

```txt
State must remain Nova data.
Platform error objects are not portable.
```

---

# Consequences

## Positive

```txt
1. Build errors are stable and tool-friendly.
2. Runtime errors flow through scheduler semantics.
3. Source maps can connect target failures to Nova source.
4. Transition failure preserves atomic commit guarantee.
5. Error lifecycle gives app-level recovery hook.
```

## Negative

```txt
1. Compiler and runtime must maintain source span metadata.
2. Error taxonomy needs ongoing discipline.
3. Adapter authors must map native failures into Nova errors.
4. Error lifecycle recursion guard adds runtime complexity.
```

---

# Final Position

Nova error model is:

```txt
diagnostic-coded at build time
scheduler-routed at runtime
source-mapped across lowering
atomic for transition failure
non-rollback for lifecycle failure
guarded for error lifecycle
```

Core rule:

```txt
Errors must be explicit, typed enough to act on, and traceable back to Nova source.
```
