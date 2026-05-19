Berikut ADR yang fokus ke **effect dan lifecycle model** Nova.

---

# ADR-005: Effect & Lifecycle Model

## Status

Implemented / Accepted

## Context

ADR-001 menetapkan:

```txt
functional core
lifecycle-bounded dirty effects
external script selalu dirty
```

ADR-002 menetapkan scheduler sebagai pemilik event ordering, transition planning, dan commit.

ADR ini memperjelas:

```txt
apa itu effect
dimana effect boleh terjadi
bagaimana lifecycle dijalankan
bagaimana async operation masuk lagi ke scheduler
bagaimana error lifecycle ditangani
```

---

# Decision

Nova hanya memiliki satu dirty zone userland:

```nova
<lifecycle ...>
  ...
/|
```

Construct berikut harus pure:

```txt
contract type
contract state
func
template
```

Construct berikut dapat melakukan dirty operation:

```txt
lifecycle
external adapter
host adapter
renderer adapter
runtime scheduler internals
```

Userland dirty operation hanya valid di lifecycle.

Secara arsitektur, scheduler, transition, type checking, dan template evaluation berada di
**functional core**. Lifecycle berada di application shell: ia menerima snapshot immutable dari core,
memanggil port yang dideklarasikan, lalu mengembalikan event baru ke scheduler.

Dalam model **hexagonal**, effect tidak dipanggil sebagai global API. Effect selalu melewati port:

```txt
inbound port:
  host event
  renderer event
  async completion event

outbound port:
  external operation
  renderer invalidation
  notification/navigation/device/storage/network
```

Adapter per target mengimplementasikan port tersebut. Core hanya melihat data contract, event,
snapshot, dan diagnostic.

---

# Effect Definition

Effect adalah operasi yang dapat:

```txt
membaca atau menulis dunia luar
mengakses waktu/random
berkomunikasi dengan network/storage/device
memanggil platform API
mengubah renderer/platform state
menghasilkan hasil yang tidak hanya bergantung pada input data
```

Contoh dirty operation:

```txt
storage
network
filesystem
audio
notification
navigation
clipboard
camera
location
time
random
native process
analytics
```

Semua external operation dianggap dirty, walaupun implementasinya tampak pure.

---

# Functional Core and Effect Shell

Nova memakai pemisahan berikut:

```txt
functional core:
  parse
  validate
  type check
  plan transition
  commit state
  derive invalidation
  evaluate template to view IR

effect shell:
  run lifecycle phase
  call external operation port
  call renderer adapter
  receive host/platform event
  route adapter failure
```

Rule:

```txt
1. Core function menerima data dan mengembalikan data/diagnostics.
2. Core function tidak memanggil clock, random, IO, platform API, atau mutable singleton.
3. Lifecycle melihat snapshot, bukan state mutable.
4. Lifecycle tidak commit state, hanya emit event atau memanggil port.
5. Adapter boleh dirty, tetapi hasilnya harus kembali sebagai event atau error terstruktur.
```

Pipeline canonical:

```txt
host/renderer adapter
  -> enqueue event
  -> pure transition planning
  -> atomic scheduler commit
  -> invalidation metadata
  -> lifecycle effect shell
  -> external/renderer adapter
  -> completion event or error
```

---

# Hexagonal Ports

Port minimal yang relevan untuk effect model:

```txt
HostEventPort
  platform event -> EventEnvelope

ExternalOperationPort
  operation input data -> operation output data or adapter error

RendererPort
  StateCommit + ViewIR metadata -> platform update

LifecyclePort
  LifecycleContext -> LifecycleOutput

ErrorPort
  SchedulerError -> lifecycle error handling / host reporting
```

Rule:

```txt
1. Lifecycle hanya boleh memakai port yang tersedia melalui import atau runtime contract.
2. Port input dan output harus menggunakan Nova data value.
3. Adapter adalah satu-satunya tempat native handle boleh hidup.
4. Permission check ditempelkan pada port, bukan pada pure transition.
5. Port failure tidak boleh dilempar sebagai value normal ke transition.
```

Dengan model ini, target Web, Android, iOS, dan Desktop dapat mengganti adapter tanpa mengubah
semantic scheduler.

---

# Lifecycle Phases

Lifecycle phase minimal:

```txt
mount
dispose
before @event
after @event
error
```

Makna:

```txt
mount          capability instance aktif
dispose        capability instance akan dilepas
before @event  sebelum transition untuk event dievaluasi
after @event   setelah commit dan invalidation untuk event
error          saat scheduler/runtime mengarahkan error
```

Contoh:

```nova
<lifecycle mount>
  void -> @sync;
/|

<lifecycle after @sync>
  "items" |> storage.load key <- "items";
/|
```

---

# Lifecycle Input

Runtime memberi lifecycle context:

```txt
LifecycleContext {
  snapshot: StateSnapshot
  event?: EventEnvelope
  error?: SchedulerError
}
```

Rule:

```txt
1. Lifecycle membaca state melalui snapshot.
2. Lifecycle tidak menulis state langsung.
3. Lifecycle dapat memanggil external operation.
4. Lifecycle dapat emit scheduler event yang tersedia.
5. Lifecycle output event selalu masuk queue, tidak dieksekusi reentrant.
```

---

# Before Lifecycle

`before @event` berjalan sebelum transition.

Rule MVP:

```txt
1. before lifecycle tidak dapat membatalkan event utama.
2. before lifecycle boleh emit event lanjutan.
3. event lanjutan diproses setelah event aktif selesai.
4. error pada before lifecycle diarahkan ke error lifecycle.
5. transition event utama tetap berjalan kecuali scheduler mengalami fatal error internal.
```

Reason:

```txt
Cancellation yang bergantung async effect mudah membuat behavior berbeda antar platform.
Validation sebaiknya berada di pure transition atau pure function.
```

---

# After Lifecycle

`after @event` berjalan setelah commit.

Use case:

```txt
persist state
analytics
network request
audio update
navigation
notification
emit follow-up event
```

Contoh:

```nova
<lifecycle after @increment>
  count |> storage.set key <- "counter";
  void -> @counter_saved;
/|
```

Rule:

```txt
1. after lifecycle melihat snapshot setelah commit.
2. error after lifecycle tidak rollback commit.
3. event yang di-emit after lifecycle masuk queue setelah event aktif.
```

---

# Mount and Dispose

`mount` dan `dispose` bukan scheduler event biasa, tetapi runtime lifecycle phase.

```nova
<lifecycle mount>
  void -> @load;
/|

<lifecycle dispose>
  subscription |> stream.close;
/|
```

Rule:

```txt
1. mount berjalan setelah capability instance terdaftar.
2. dispose berjalan sebelum capability instance dilepas.
3. mount/dispose boleh emit event.
4. mount/dispose tidak boleh menulis state langsung.
5. event dari dispose tetap melewati queue jika runtime masih menerima event.
```

Jika target tidak dapat menjamin event dari dispose akan sempat diproses, adapter harus memberi diagnostic
atau memakai cleanup native yang tidak bergantung state commit.

---

# Error Lifecycle

`error` menerima `SchedulerError`.

```nova
<lifecycle error>
  error.message |> notify.send msg <- error.message type <- "error";
/|
```

Rule:

```txt
1. Error lifecycle tidak boleh memicu infinite recursive error.
2. Runtime harus memiliki guard untuk error saat menjalankan error lifecycle.
3. Error lifecycle boleh emit event pemulihan.
4. Event pemulihan masuk queue normal.
5. Error lifecycle tidak rollback commit yang sudah terjadi.
```

Error dari transition membatalkan commit event aktif.
Error dari lifecycle terjadi di luar pure commit dan tidak rollback commit.

---

# Async Effect

Nova tidak mengekspos promise, future, coroutine, callback, atau thread primitive di surface language.

External async operation dipetakan menjadi event.

```txt
lifecycle starts external operation
adapter performs platform async work
adapter validates result
adapter enqueues completion event
```

Contoh pattern:

```nova
<lifecycle after @load_user>
  userId |> api.fetchUser id <- userId;
/|
```

Adapter dapat enqueue:

```txt
@fetch_user_ok(user: User)
@fetch_user_failed(message: string)
```

Completion event harus dideklarasikan atau di-import.

---

# Effect Capability Access

Lifecycle hanya boleh memanggil external operation yang di-import oleh file tersebut.

```nova
<import external storage from "@env/storage">
  operation set {
    input {
      key: string;
      value: unknown;
    }

    output void;
  }
/|
```

Rule:

```txt
1. External operation harus ada dalam import contract.
2. Input harus sesuai type.
3. Permission target harus mengizinkan operation.
4. Operation failure masuk error lifecycle atau completion failure event.
5. Operation dipanggil melalui ExternalOperationPort, bukan akses langsung ke platform global.
```

---

# Forbidden Effects

Invalid:

```nova
<func save value: number returns void>
  value |> storage.set key <- "counter"
/|
```

Invalid:

```nova
<contract state Counter>
  count: number <- 0 {
    @increment -> count |> storage.set key <- "counter";
  };
/|
```

Invalid:

```nova
<template>
  <button label <- time.now() /|
/|
```

Reason:

```txt
Func, transition, dan template harus pure.
```

---

# Alternatives Considered

## Effect Keyword

Menambahkan construct `<effect>` sebagai dirty block.

Rejected because:

```txt
Lifecycle sudah menjadi runtime boundary yang jelas.
Effect tanpa lifecycle phase membuat ordering kurang eksplisit.
```

## Async Transition

Transition boleh menunggu external operation.

Rejected because:

```txt
Commit timing menjadi platform-dependent.
Transition tidak lagi pure.
Error dan cancellation sulit distandarkan.
```

## Template Side Effects

Template boleh memanggil API platform saat render.

Rejected because:

```txt
Renderer menjadi pemilik effect.
Render ulang dapat menggandakan side effect.
Pure view model hilang.
```

---

# Consequences

## Positive

```txt
1. Semua dirty operation mudah ditemukan.
2. Pure transition tetap deterministic.
3. Runtime dapat menjaga event ordering lintas platform.
4. External async API dapat dipakai lewat completion event.
5. Security permission dapat ditempelkan ke lifecycle/external boundary.
```

## Negative

```txt
1. Workflow async harus dimodelkan sebagai event.
2. User tidak dapat memakai imperative call stack untuk effect.
3. Lifecycle graph perlu tooling/debugger.
4. Beberapa cleanup native membutuhkan adapter khusus.
```

---

# Final Position

Nova effect model adalah:

```txt
pure by default
dirty only in lifecycle
async by event completion
state mutation only by scheduler commit
permission checked at external boundary
functional core with effect shell
hexagonal ports with target adapters
```

Core rule:

```txt
If it touches the outside world, it belongs in lifecycle code through an explicit port or in adapter code.
```
