Berikut ADR yang fokus ke **security dan permission model** Nova.

---

# ADR-013: Security & Permission Model

## Status

Implemented / Accepted

## Context

Nova membuat boundary eksplisit antara pure core dan dirty lifecycle/external adapter.
Boundary tersebut juga harus menjadi dasar security.

Risiko utama:

```txt
external code membaca/menulis data tanpa deklarasi
platform API dipakai tanpa permission
event payload membawa data tidak valid
renderer/host adapter menyuntik event palsu
package dependency memperluas capability diam-diam
target-specific code berbeda dari contract
```

ADR ini mendefinisikan prinsip security dan permission lintas platform.

---

# Decision

Nova memakai model:

```txt
deny by default
capability-based permission
explicit external boundary
build-time permission audit
runtime permission enforcement
serializable data boundary
adapter accountability
```

Nova source tidak mendapat akses platform langsung.
Akses platform hanya lewat capability contract dan external adapter.

---

# Trust Boundaries

Trust boundary utama:

```txt
Nova source
package Nova source
external implementation
target adapter
host event input
renderer platform output
serialized persisted data
devtools/debug input
```

Pure Nova code lebih dipercaya secara semantic karena compiler dapat memvalidasi purity/type.
External dan adapter code tetap dianggap boundary yang harus diaudit.

---

# Deny by Default

Capability yang menyentuh platform membutuhkan deklarasi.

Default:

```txt
no storage
no network
no filesystem
no location
no camera
no microphone
no notification
no clipboard
no process execution
no arbitrary native API
```

Project manifest harus mengizinkan permission yang dibutuhkan.

---

# Permission Declaration

Recommended manifest:

```toml
[permissions]
storage.read = true
storage.write = true
network.request = true
notification.send = false
```

Permission dapat di-scope:

```toml
[permissions.storage]
read = ["settings", "profile"]
write = ["settings"]

[permissions.network]
allow = ["https://api.example.com"]
```

Exact manifest syntax dapat berubah, tetapi semantic permission harus dapat diaudit.

---

# Permission Attachment

External operation atau platform capability menyatakan permission.

```txt
@env/storage.get -> storage.read
@env/storage.set -> storage.write
@env/network.fetch -> network.request
@env/notify.send -> notification.send
```

Build checks:

```txt
1. operation dipakai oleh lifecycle
2. operation membutuhkan permission
3. project manifest mendeklarasikan permission
4. target adapter dapat memetakan permission ke platform
```

Jika tidak terpenuhi, build gagal.

---

# Runtime Enforcement

Build-time permission tidak menggantikan runtime permission platform.

Runtime adapter harus:

```txt
request/check platform permission when required
handle denial explicitly
validate operation input
validate operation output
route failure as event or SchedulerError
avoid leaking platform object to Nova state/event
```

Permission denial should be observable:

```txt
@permission_denied(permission: string)
```

or as `SchedulerError`.

---

# Event Security

Host/renderer event input must be validated.

Rule:

```txt
1. Event name must be declared or imported.
2. Payload must match event contract.
3. Payload must be serializable Nova data.
4. Adapter cannot enqueue event that current capability is not allowed to emit.
5. Invalid event is rejected before transition.
```

This prevents platform/renderer code from bypassing capability contracts.

---

# State Security

State may store only Nova data.

Forbidden in state:

```txt
secret native handle
DOM node
UIView
file descriptor
socket handle
promise/future/coroutine
function pointer
raw adapter object
```

Sensitive data guidance:

```txt
1. Prefer opaque identifiers over raw secrets.
2. Do not persist secrets in scheduler state unless explicitly intended.
3. External adapter should keep secure tokens in platform secure storage.
4. Devtools should redact fields marked sensitive when such annotation exists.
```

Sensitive field annotation is future work, but security model should allow it.

---

# External Code Audit

Build graph should record:

```txt
external import source
operation names
input/output types
required permissions
calling lifecycle
target implementation file
package provenance
```

This supports:

```txt
security review
permission prompt explanation
package audit
CI policy checks
generated platform entitlement review
```

---

# Package Security

Packages can introduce:

```txt
Nova capabilities
external operations
renderer primitives
target adapters
permissions
```

Rule:

```txt
1. Package permissions must be surfaced to consuming project.
2. Package cannot silently expand permission scope.
3. Lockfile should pin package versions.
4. Build diagnostics should report permission source chain.
5. Target implementation mismatch fails build.
```

---

# Renderer and Host Security

Renderer adapter must not:

```txt
mutate scheduler state directly
call lifecycle directly
emit undeclared event
change event payload type
store unvalidated platform object in Nova data
```

Host adapter must:

```txt
validate platform event payload
respect queue limits
preserve event route table
report route mismatch
```

---

# Network and Filesystem

Network and filesystem are high-risk capabilities.

Recommended permission detail:

```txt
network.request with allowlist
filesystem.read with path/scope
filesystem.write with path/scope
storage.read/write with namespace
```

Target adapter should map these into:

```txt
web CSP/fetch policy
android permissions/network security config
ios entitlements/App Transport Security
desktop sandbox profile
```

Exact platform mapping belongs to target adapter docs.

---

# Devtools

Devtools can inspect scheduler state, event queue, and diagnostics.

Rule:

```txt
1. Devtools write/injection must be explicit in development mode.
2. Production build should disable unsafe devtools injection by default.
3. Devtools event injection must pass event contract validation.
4. Sensitive data should be redacted when annotations exist.
```

---

# Alternatives Considered

## Trust All External Code

External operation can do anything once imported.

Rejected because:

```txt
Permission audit would be impossible.
Package dependencies could expand platform access silently.
```

## Runtime-Only Permissions

Skip build-time permission checks and rely on platform prompts.

Rejected because:

```txt
Some targets do not have equivalent prompts.
CI/security review needs static audit.
Build should fail before shipping missing entitlements.
```

## Platform-Specific Security Model Only

Let each target define security independently.

Rejected because:

```txt
Nova needs common guarantees across targets.
Permission graph must be language-level metadata.
```

---

# Consequences

## Positive

```txt
1. Dirty platform access is explicit and auditable.
2. Build can catch missing permissions early.
3. Runtime still respects platform permission prompts.
4. Event and state boundaries reject invalid data.
5. Package permission expansion is visible.
```

## Negative

```txt
1. Manifest permission model needs careful design.
2. Adapter authors must provide permission metadata.
3. More diagnostics are required for permission source chains.
4. Some platform capabilities need nuanced scopes.
```

---

# Final Position

Nova security model is:

```txt
deny by default
permission-declared
external-boundary audited
runtime-enforced
event-validated
state-serializable
package-transparent
```

Core rule:

```txt
Capability is permission. Dirty access must be declared before it can run.
```
