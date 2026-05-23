Berikut ADR yang fokus ke **external interop model** Nova.

---

# ADR-009: External Interop Model

## Status

Implemented / Accepted

## Context

Nova harus berinteraksi dengan kode dan API di luar Nova:

```txt
JavaScript
Kotlin
Swift
Rust
C
Java
platform SDK
network/storage/audio/device API
```

ADR-001 menetapkan bahwa external script selalu dirty.
ADR-005 menetapkan dirty operation hanya boleh dipanggil dari lifecycle.

ADR ini mendefinisikan bentuk contract external dan cara hasilnya masuk kembali ke Nova.

---

# Decision

Nova memakai explicit external import:

```nova
<import external name from "source-or-capability">
  operation operationName {
    input {
      field: Type;
    }

    output Type;
  }
/|
```

External operation:

```txt
selalu dirty
hanya callable di lifecycle
memiliki input/output contract
diselesaikan oleh target adapter
tidak boleh membawa platform object ke Nova core
```

Production layering untuk `@env/*`:

```txt
lifecycle -> external operation table -> @env adapter -> platform API
```

Tidak ada framework tambahan di antara adapter dan OS. Adapter web
memakai modul JS platform; adapter Android memakai kelas Java platform (`*.android.java`).

---

# External Import Source

Source external dapat berupa:

```txt
relative file path
package capability
environment capability
target-resolved source pattern
```

Contoh:

```nova
<import external storage from "./storage.web.js">
  ...
/|

<import external storage from "@env/storage">
  ...
/|
```

Resolver menentukan implementasi aktual berdasarkan target build.

---

# Operation Contract

Operation memiliki:

```txt
name
input record
output type
failure mode
permission requirement
target implementation
```

Contoh:

```nova
operation send {
  input {
    msg: string;
    type?: string;
  }

  output void;
}
```

Rule:

```txt
1. Input harus serializable Nova data.
2. Output harus serializable Nova data.
3. Unknown boleh dipakai, tetapi harus divalidasi sebelum menjadi type spesifik.
4. Output contract adalah boundary validation contract.
5. Operation tidak boleh mengembalikan native handle ke state/event.
```

---

# Invocation Semantics

External operation invocation di lifecycle memulai request ke adapter.

```nova
<lifecycle after @sync>
  "iem_vault" |> storage.load key <- "iem_vault";
/|
```

Canonical runtime:

```txt
lifecycle statement
  -> external invocation request
  -> adapter executes target code
  -> adapter validates completion
  -> adapter enqueues completion event or routes error
```

Nova surface language tidak menunggu promise/future/callback.

---

# Completion Model

Hasil external operation masuk kembali sebagai:

```txt
completion scheduler event
atau SchedulerError
```

Preferred pattern:

```txt
operation success -> @operation_ok(payload)
operation failure -> @operation_failed(message)
```

Completion event harus:

```txt
dideklarasikan di contract capability
atau di-import sebagai event
atau disediakan oleh adapter binding target
```

Jika adapter tidak punya mapping event, failure masuk `error` lifecycle.

---

# Sync and Async

Nova tidak membedakan sync/async di source language.

Rule:

```txt
1. Adapter boleh menyelesaikan operation secara sinkron.
2. Adapter tetap mengembalikan completion melalui scheduler semantics.
3. Completion tidak boleh menjalankan nested commit.
4. Completion event mendapat LogicalSequence baru.
```

Reason:

```txt
Perbedaan sync/async platform tidak boleh mengubah urutan scheduler.
```

---

# Data Marshalling

External adapter bertanggung jawab melakukan marshalling:

```txt
Nova data -> platform call input
platform result -> Nova data
platform error -> SchedulerError atau failure event
```

Allowed Nova data:

```txt
string
number
boolean
null
array
record with string keys
opaque value with serializable representation
```

Not allowed:

```txt
DOM node
UIView
thread
file descriptor
socket handle
function pointer
promise/future/coroutine object
class instance without serializable representation
```

---

# External Purity Boundary

Even if an external function is mathematically pure, Nova treats it as dirty.

Reason:

```txt
Compiler cannot verify arbitrary external language semantics.
External language can read time, globals, IO, or platform state.
Target behavior can differ.
```

Pure reusable logic should be written as Nova `<func>` or as future verified pure package format,
not as normal external import.

---

# Target Implementations

Target-specific files may follow naming convention:

```txt
name.web.js
name.android.java
name.ios.swift
name.desktop.rs
name.common.js
```

Resolver selection is defined in ADR-010.

Rule:

```txt
1. All selected implementations must satisfy the same operation contract.
2. Missing operation fails build.
3. Mismatched input/output fails build.
4. Runtime validation still checks untrusted results.
```

---

# Error Handling

External failure can be represented as:

```txt
failure completion event
SchedulerError routed to error lifecycle
fatal adapter error reported to host
```

Rule:

```txt
1. Expected domain failure should use typed failure event.
2. Unexpected platform failure may use SchedulerError.
3. Permission denial should be explicit and inspectable.
4. External failure does not rollback commits that already happened.
5. Failure event follows scheduler queue ordering.
```

---

# Security

External code is a trust boundary.

Build/runtime must track:

```txt
which external operations are used
which lifecycle calls them
which permissions they require
which target implementation provides them
```

Security model detail ada di ADR-013.

---

# Alternatives Considered

## Direct Foreign Function Calls in Func

Nova `<func>` boleh memanggil external pure helper.

Rejected because:

```txt
Purity external language tidak dapat dijamin.
Target conformance menjadi sulit.
```

## Promise/Future in Nova Surface

Nova memiliki async/await atau coroutine syntax.

Out of production v1 scope because:

```txt
Scheduler event model sudah menyediakan async completion.
Async primitive akan memperbesar language core.
Perbedaan runtime target dapat bocor.
```

## Native Object Handles in State

External dapat mengembalikan native handle dan menyimpannya di state.

Rejected because:

```txt
State/event harus serializable dan platform-neutral.
Native handle harus tinggal di adapter.
```

---

# Consequences

## Positive

```txt
1. Interop tetap eksplisit dan auditable.
2. Dirty code tidak bocor ke pure zone.
3. Sync/async platform tidak mengubah scheduler semantics.
4. Target implementation dapat berbeda selama contract sama.
5. Security dan permission dapat dilacak dari graph.
```

## Negative

```txt
1. External result harus dimodelkan sebagai event/error.
2. Adapter perlu marshalling dan validation.
3. Pure helper external belum didukung pada production v1.
4. Package author harus menyediakan implementasi target yang konsisten.
```

---

# Final Position

Nova external interop adalah:

```txt
dirty by default
contract-based
lifecycle-only
adapter-executed
event-completed
serializable at boundary
```

Core rule:

```txt
External code may touch the world. Nova core only receives validated data or errors.
```
