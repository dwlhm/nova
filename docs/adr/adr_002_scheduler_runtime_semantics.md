Berikut ADR yang fokus ke **scheduler runtime semantics** Nova dengan desain multiplatform.

---

# ADR-002: Scheduler Runtime Semantics

## Status

Implemented / Accepted

## Context

ADR-001 menetapkan Nova sebagai:

```txt
scheduler-centered multiplatform framework language
functional core
lifecycle-bounded dirty effects
capability-driven rendering
```

Namun runtime scheduler belum didefinisikan secara eksplisit.

Scheduler perlu menjadi pusat koordinasi untuk:

```txt
scheduler event
pure state transition
atomic state commit
lifecycle dirty operation
render invalidation
external operation completion
error routing
```

Target platform production Nova mencakup:

```txt
Web
Android
```

Target future:

```txt
iOS
Desktop
future target
```

Masing-masing platform memiliki event loop, renderer, threading model, lifecycle aplikasi,
dan API native yang berbeda. Perbedaan tersebut tidak boleh bocor ke semantic core Nova.

---

# Decision

Nova menggunakan **single logical scheduler semantics** yang platform-neutral.

Implementasi scheduler untuk target aplikasi harus **native to the platform**:

```txt
web      -> scheduler runtime native JavaScript/TypeScript browser runtime
android  -> scheduler runtime native Java/Android runtime
```

Implementasi Go yang ada di repository adalah **reference/conformance prototype** untuk
memvalidasi semantic scheduler, test suite, dan tooling awal. Implementasi Go tidak menjadi
runtime target aplikasi untuk Web atau Android.

Implementasi runtime dipisahkan menjadi:

```txt
nova-scheduler-semantics
platform-native scheduler runtime
platform host adapter
platform renderer adapter
external operation adapter
```

Scheduler semantics bertanggung jawab mendefinisikan urutan event, evaluasi transition,
commit state, dan routing lifecycle. Runtime native per platform wajib mengikuti semantics
tersebut dan lolos conformance test yang sama.

Platform adapter bertanggung jawab atas integrasi native seperti:

```txt
DOM event loop
Android Looper / coroutine dispatcher
```

Future target adapter dapat menambahkan:

```txt
iOS MainActor / run loop
desktop UI loop
storage/network/audio/native API
```

Core Nova tidak mengenal:

```txt
DOM
Android View
UIView
SwiftUI
(Compose hanya pada @nova/android-compose deprecated)
thread native
platform clock
storage native
network native
```

Semua hal tersebut masuk melalui adapter.

---

# Runtime Units

## Event Envelope

Semua event yang masuk ke scheduler direpresentasikan sebagai envelope.

```txt
EventEnvelope {
  sequence: LogicalSequence
  source: CapabilityRef
  name: SchedulerEvent
  payload: DataValue | void
}
```

Rule:

```txt
1. sequence dibuat oleh scheduler, bukan platform.
2. source adalah capability yang mengirim event.
3. name selalu scheduler event dengan prefix @.
4. payload harus sesuai contract event.
5. payload harus berupa data serializable Nova.
```

Scheduler tidak memakai timestamp platform untuk menentukan urutan event.

Urutan event ditentukan oleh `LogicalSequence` agar behavior konsisten di semua target.

## State Cell

State dari `<contract state>` dikompilasi menjadi state cell.

```txt
StateCell {
  owner: CapabilityRef
  name: StateName
  type: TypeRef
  value: DataValue
  transitions: EventPattern -> PureExpr
}
```

State hanya berubah melalui scheduler commit.

Transition menghasilkan candidate next value, bukan mutation langsung.

## Lifecycle Handler

`<lifecycle>` dikompilasi menjadi lifecycle handler.

```txt
LifecycleHandler {
  owner: CapabilityRef
  phase: LifecyclePhase
  event?: SchedulerEvent
  body: DirtyPipeline
}
```

Lifecycle adalah dirty zone userland.

Lifecycle boleh memanggil external operation dan emit scheduler event, tetapi tidak boleh
menulis state langsung.

---

# Scheduler Loop

Scheduler menjalankan satu logical event queue per app instance.

Flow canonical:

```txt
1. enqueue event
2. run before lifecycle for event
3. evaluate matching pure transitions
4. commit state batch atomically
5. notify renderer with invalidation set
6. run after lifecycle for event
7. enqueue events emitted by lifecycle
8. drain next event
```

Dalam bentuk singkat:

```txt
Event
  -> before lifecycle
  -> transition plan
  -> state commit
  -> render invalidation
  -> after lifecycle
  -> next Event
```

## Atomic Commit

Satu event menghasilkan satu commit batch.

Jika satu event memengaruhi beberapa state:

```nova
<contract state Login>
  status: Status <- "idle" {
    @login_ok(user: User) -> "success";
  };

  user: User | null <- null {
    @login_ok(user: User) -> user;
  };
/|
```

Maka scheduler mengevaluasi semua transition terlebih dahulu, lalu commit bersama.

Rule:

```txt
1. Transition membaca snapshot state sebelum event.
2. Transition tidak melihat hasil transition lain dalam batch yang sama.
3. Semua hasil valid dikomit bersama.
4. Jika transition gagal type check/runtime validation, batch untuk event tersebut tidak dikomit.
5. Error dikirim ke lifecycle error.
```

## Event Reentrancy

Event yang di-emit saat scheduler sedang memproses event tidak dieksekusi langsung.

Event tersebut masuk ke queue.

```txt
current event
  after lifecycle emits @next
  @next enqueued
  current event selesai
  scheduler drain @next
```

Rule:

```txt
No recursive dispatch.
No nested commit.
No platform-specific reentrancy.
```

Ini menjaga behavior sama di Web, Android, iOS, dan Desktop.

---

# Lifecycle Semantics

## Lifecycle Phases

Phase minimal tetap mengikuti ADR-001:

```txt
mount
dispose
before @event
after @event
error
```

Makna runtime:

```txt
mount          dipanggil setelah capability instance aktif.
dispose        dipanggil sebelum capability instance dilepas.
before @event  dipanggil sebelum transition event dievaluasi.
after @event   dipanggil setelah commit dan render invalidation event.
error          dipanggil ketika transition/lifecycle/external operation gagal.
```

## Before Lifecycle

`before @event` boleh melakukan dirty operation dan emit event, tetapi tidak boleh membatalkan
event utama pada production scheduler.

Reason:

```txt
Cancellation membuat urutan state platform-dependent jika before lifecycle berisi async operation.
Nova menjaga event -> transition -> commit tetap deterministic.
```

Jika validasi diperlukan, validasi sebaiknya berada di pure transition atau pure function.

## After Lifecycle

`after @event` berjalan setelah state commit.

Lifecycle ini cocok untuk:

```txt
persist state
analytics
network request
audio side effect
navigation side effect
emit follow-up event
```

Contoh:

```nova
<lifecycle after @increment>
  count |> storage.set key <- "counter";
  void -> @counter_saved;
/|
```

## Async External Operation

External operation tidak boleh membuat scheduler core menunggu platform-specific promise,
future, coroutine, atau callback.

Platform adapter menerjemahkan completion menjadi event baru.

```txt
external operation start
  -> platform async work
  -> completion callback
  -> adapter enqueue @operation_ok / @operation_failed
```

Nova code harus memodelkan hasil async sebagai scheduler event.

```nova
<lifecycle after @load_profile>
  userId |> api.fetchUser;
/|
```

Adapter dapat mengirim:

```txt
@fetch_user_ok(user: User)
@fetch_user_failed(message: string)
```

Nama event hasil async harus dideklarasikan sebagai bagian dari contract capability atau import event.

---

# Multiplatform Boundary

## Platform-Native Core Runtime

Core runtime wajib sama secara semantic untuk semua platform.

Core runtime harus diimplementasikan native per target yang didukung.

Production implementation target:

```txt
web      -> JavaScript/TypeScript scheduler runtime
android  -> Java scheduler runtime (generated MainActivity + NovaRuntime)
```

Reference implementation:

```txt
go -> conformance prototype and tooling reference
```

Semua implementasi harus lolos conformance test yang sama.

Core runtime mengatur:

```txt
event queue ordering
transition matching
state snapshot
atomic commit
lifecycle routing
render invalidation metadata
error routing
```

## Host Adapter

Host adapter menghubungkan platform event ke scheduler event.

Contoh:

```txt
DOM click              -> @pressed
Android onClick        -> @pressed
```

Future target:

```txt
iOS button action      -> @pressed
desktop button command -> @pressed
```

Adapter hanya membuat event envelope. Adapter tidak boleh langsung mengubah state.

## Renderer Adapter

Renderer adapter menerima invalidation set dari scheduler.

```txt
StateCommit
  -> invalidation set
  -> renderer adapter
  -> platform render update
```

Renderer adapter boleh berbeda per target, tetapi harus memakai data hasil commit yang sama.

Core scheduler native tidak memanggil DOM, Android View, atau platform UI API secara
langsung. Integrasi platform tetap melalui adapter.

## External Adapter

External import selalu dirty dan diselesaikan oleh target adapter.

```nova
<import external storage from "./storage.web.js">
  operation set {
    input {
      key: string;
      value: unknown;
    }

    output void;
  }
/|
```

Untuk multiplatform, source external dapat dipetakan oleh build target:

```txt
storage.web.js
storage.android.kt
```

Target future dapat menambahkan:

```txt
storage.ios.swift
storage.desktop.rs
```

Contract operation harus sama di semua target yang didukung.

---

# Scheduling Rules

## Ordering

Ordering scheduler:

```txt
1. User/platform event yang masuk lebih dulu mendapat sequence lebih kecil.
2. Event yang di-emit lifecycle masuk setelah event aktif selesai.
3. Event hasil async external masuk saat adapter menerima completion.
4. Scheduler memproses queue secara FIFO berdasarkan LogicalSequence.
```

Tidak ada prioritas platform-specific pada semantic core.

Jika target butuh batching native, batching terjadi di adapter tanpa mengubah urutan logical event.

## Concurrency

Nova scheduler memakai single logical writer untuk state.

Rule:

```txt
1. State commit tidak paralel.
2. Pure transition boleh dievaluasi paralel jika hasilnya setara deterministic.
3. Lifecycle dirty operation boleh async, tetapi completion kembali sebagai event.
4. Renderer update harus menerima commit dalam urutan scheduler.
```

## Backpressure

Scheduler boleh menolak atau menunda event jika queue melewati batas runtime.

Namun semantic default adalah:

```txt
enqueue until handled
preserve FIFO order
surface overload through error lifecycle
```

Backpressure policy adalah runtime configuration, bukan syntax Nova.

---

# Error Model

Error sumber:

```txt
transition evaluation error
transition type validation error
lifecycle runtime error
external operation failure
renderer adapter failure
```

Routing:

```txt
1. Transition error membatalkan commit event aktif.
2. Lifecycle error tidak rollback commit yang sudah terjadi.
3. External failure harus masuk sebagai event atau error lifecycle.
4. Renderer failure masuk error lifecycle dan host adapter.
5. Error lifecycle tidak boleh dispatch recursive error tanpa guard.
```

Error envelope:

```txt
SchedulerError {
  source: CapabilityRef
  phase: SchedulerPhase
  event?: SchedulerEvent
  message: string
  cause?: unknown
}
```

---

# Build-Time Responsibilities

Compiler/build graph harus menghasilkan:

```txt
state graph
event graph
lifecycle graph
capability dependency graph
target capability resolution
external operation table
renderer target table
```

Compiler harus memvalidasi:

```txt
1. Event yang di-listen lifecycle tersedia melalui local declaration atau import event.
2. Event yang di-emit template/lifecycle sesuai contract.
3. Payload event sesuai type.
4. State transition pure.
5. External operation hanya dipanggil di lifecycle.
6. Target build memiliki implementation untuk semua capability dan external import.
7. Capability implementation target memiliki props/emits yang kompatibel.
```

---

# Example

```nova
<import external storage from "./storage.web.js">
  operation set {
    input {
      key: string;
      value: unknown;
    }

    output void;
  }
/|

<contract state Counter>
  count: number <- 0 {
    @increment -> count |> add 1;
    @set(value: number) -> value;
  };
/|

<func add value: number amount: number returns number>
  value + amount
/|

<template>
  <button on_press -> @increment>
    +
  /|
/|

<lifecycle after @increment>
  count |> storage.set key <- "counter";
/|
```

Runtime flow:

```txt
button press
  -> host adapter enqueue @increment
  -> scheduler evaluates count transition
  -> scheduler commits count
  -> renderer receives count invalidation
  -> after @increment lifecycle persists count
```

Flow semantic tersebut sama untuk production Web dan Android.

Future target seperti iOS dan Desktop harus mengikuti flow semantic yang sama saat didukung.

Yang berbeda per target:

```txt
platform-native scheduler runtime
button native event
renderer primitive
storage implementation
async completion mechanism
```

---

# Alternatives Considered

## Platform-Idiomatic Scheduler Semantics per Target

Setiap target bebas mendefinisikan semantic scheduler sendiri mengikuti idiom platform.

Rejected because:

```txt
Semantic event ordering mudah berbeda antar platform.
State commit dapat mengikuti behavior event loop native yang tidak seragam.
Conformance language menjadi sulit.
```

Nova tetap memakai implementasi scheduler native per platform, tetapi semantic event ordering,
atomic commit, lifecycle routing, dan error routing harus mengikuti ADR ini.

## Renderer-Owned State

Renderer platform mengelola state dan Nova hanya menjadi template DSL.

Rejected because:

```txt
Nova kehilangan scheduler-owned state guarantee.
Dirty operation lebih mudah bocor ke view layer.
Pure transition tidak menjadi pusat model aplikasi.
```

## Async Transition

Transition boleh async dan menunggu external operation.

Rejected because:

```txt
Transition tidak lagi pure.
Commit timing menjadi platform-dependent.
Error dan cancellation menjadi sulit distandarkan.
```

---

# Consequences

## Positive

```txt
1. Semantic scheduler sama untuk semua target.
2. State commit tetap deterministic dan scheduler-owned.
3. Dirty operation tetap terisolasi di lifecycle/external adapter.
4. Renderer platform dapat berkembang tanpa mengubah core language.
5. Runtime conformance dapat dites lintas implementasi.
6. Async native API dapat dipakai tanpa mengotori transition.
```

## Negative

```txt
1. Runtime harus memiliki adapter layer per target.
2. Compiler perlu membangun event/lifecycle graph yang lengkap.
3. Async workflow harus dimodelkan sebagai event lanjutan.
4. Platform optimization harus menjaga urutan logical scheduler.
5. Debugging membutuhkan tooling untuk event queue dan commit batch.
```

---

# Final Position

Nova scheduler adalah:

```txt
single logical event queue
pure transition planner
atomic state committer
lifecycle dirty operation router
platform-neutral invalidation producer
```

Multiplatform guarantee:

```txt
Same Nova input.
Same logical event order.
Same transition result.
Same state commit.
Different host, renderer, and external adapters.
```

Core rule:

```txt
Platform adapts to scheduler.
Scheduler does not adapt to platform.
```
