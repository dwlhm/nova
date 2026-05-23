Berikut ADR yang fokus ke **renderer dan lowering pipeline** Nova.

---

# ADR-008: Renderer & Lowering Pipeline

## Status

Implemented / Accepted

## Context

Nova template harus dapat dirender ke berbagai target tanpa membuat semantic core bergantung
pada DOM, Android View, UIKit, SwiftUI, atau toolkit desktop.

Nova juga perlu menjaga agar diagnostic, source map, invalidation, dan event routing tetap konsisten
sepanjang pipeline compiler ke runtime.

---

# Decision

Nova memakai pipeline lowering bertahap:

```txt
source .nova
  -> tokens
  -> AST
  -> semantic graph
  -> typed IR
  -> scheduler IR
  -> view IR
  -> target IR
  -> platform artifact
```

Renderer tidak membaca source `.nova` langsung.
Renderer menerima output pipeline yang sudah tervalidasi.

---

# Pipeline Stages

## 1. Lexing

Input:

```txt
.nova source
```

Output:

```txt
Token[]
```

Lexing hanya mengenali token dan lokasi.
Lexing tidak melakukan semantic validation.

## 2. Parsing

Input:

```txt
Token[]
```

Output:

```txt
AST
```

Parser mengenali:

```txt
import
external import
contract type
contract state
contract capability
func
template
lifecycle
```

Parser boleh menyimpan body expression/template sebagai token stream sampai expression grammar matang.

## 3. Semantic Graph

Compiler membangun:

```txt
module graph
symbol table
type graph
state graph
event graph
lifecycle graph
external operation graph
template graph
```

Diagnostic semantic mulai muncul di tahap ini.

## 4. Typed IR

Typed IR menempelkan type ke:

```txt
state cell
transition expression
function parameter/return
event payload
template binding
external input/output
capability props/emits
```

Typed IR adalah input untuk scheduler dan renderer lowering.

## 5. Scheduler IR

Scheduler IR berisi:

```txt
StateCell definitions
TransitionPlan
EventContract
LifecycleHandler metadata
ExternalOperationRef
ErrorRoute
```

Scheduler IR tidak berisi platform renderer detail.

## 6. View IR

View IR berisi:

```txt
ViewNode tree
BindingExpr
EventRoute
ComponentRef
TargetConstraint
SourceSpan
```

View IR target-neutral.

## 7. Target IR

Target IR adalah format khusus adapter.

Contoh:

```txt
web target IR
android target IR
ios target IR
desktop target IR
```

Target IR boleh memakai konsep target, tetapi tetap harus mempertahankan source map dan event route
ke scheduler.

---

# Renderer Contract

Renderer adapter menerima:

```txt
initial view IR atau target IR
state snapshot
state invalidation set
event route table
capability manifest
```

Renderer adapter menghasilkan:

```txt
platform UI update
host event mapping
render diagnostics
```

Renderer adapter tidak boleh:

```txt
menulis scheduler state
memanggil lifecycle userland langsung
mengubah event ordering scheduler
mengubah type payload event
```

---

# Invalidation Model

Scheduler commit menghasilkan:

```txt
StateCommit {
  sequence
  event
  changes
  invalidations
}
```

Renderer memakai invalidation untuk memilih evaluasi ulang.

Rule:

```txt
1. Renderer menerima commit dalam sequence order.
2. Renderer tidak boleh melewati commit.
3. Renderer boleh batch platform updates jika logical order tetap sama.
4. Renderer cache tidak menjadi source of truth.
```

---

# View Evaluation

Template dievaluasi sebagai projection:

```txt
StateSnapshot + Props -> ViewIR
```

Projection harus pure.

Jika evaluasi binding gagal:

```txt
1. Renderer melaporkan renderer evaluation error.
2. Runtime mengarahkan error ke error lifecycle.
3. Scheduler state tidak diubah oleh renderer failure.
```

---

# Event Route Table

Event route table menghubungkan platform event ke scheduler event.

```txt
ViewNodeId.on_press -> @increment(void)
ViewNodeId.on_select -> @select(item)
ChildCapability.@pressed -> @save(payload)
```

Rule:

```txt
1. Event route dibuat saat lowering.
2. Payload expression dievaluasi terhadap snapshot/render scope.
3. Event dikirim sebagai EventEnvelope melalui scheduler enqueue.
4. Renderer tidak memanggil transition langsung.
```

---

# Source Maps

Setiap IR node harus membawa `SourceSpan` jika berasal dari source.

Dipakai untuk:

```txt
diagnostics
runtime error overlay
devtools selection
generated artifact debugging
test snapshot
```

Target artifacts harus mempertahankan mapping semampunya.

---

# Target Artifact

Artifact build dapat berupa:

```txt
web bundle
android source/generated module
ios source/generated module
desktop executable resources
intermediate package artifact
```

ADR ini tidak mengunci format artifact.

Yang dikunci:

```txt
semantic event ordering
state ownership
view IR contract
source mapping
adapter boundary
```

---

# Optimization

Optimisasi boleh dilakukan jika tidak mengubah semantic.

Allowed:

```txt
constant folding pure expression
dead template branch elimination untuk target tertentu
view subtree memoization
state invalidation pruning
event route table compaction
renderer-native batching
```

Not allowed:

```txt
reorder scheduler events
skip lifecycle
merge commits yang berbeda sequence
move dirty operation ke template/transition
drop runtime validation at external boundary
```

---

# Alternatives Considered

## Direct Source-to-Platform Compilation

Setiap target compiler membaca source Nova dan menghasilkan platform code sendiri.

Rejected because:

```txt
Semantic mudah drift antar target.
Diagnostics sulit seragam.
Conformance testing harus menguji terlalu banyak compiler path.
```

## Runtime Interpret Everything

Runtime menyimpan source/template mentah dan mengevaluasi semuanya saat berjalan.

Rejected because:

```txt
Target optimization sulit.
Build-time diagnostics berkurang.
Security audit external capability lebih lemah.
```

## Renderer Owns Scheduler

Renderer target sekaligus memproses event dan state.

Rejected because:

```txt
State commit harus scheduler-owned.
Renderer hanya adapter visual dan host event.
```

---

# Consequences

## Positive

```txt
1. Pipeline dapat dites per tahap.
2. Renderer target tidak perlu memahami seluruh source language.
3. Source map mendukung diagnostics dan devtools.
4. Optimisasi dapat dilakukan tanpa mengubah semantic core.
5. Conformance dapat berfokus pada IR behavior.
```

## Negative

```txt
1. Ada beberapa IR yang harus dirawat.
2. Debugging generated target code butuh source map.
3. Adapter harus menjaga event route table dengan benar.
4. Build pipeline lebih panjang dibanding interpreter sederhana.
```

---

# Final Position

Nova renderer pipeline adalah:

```txt
source to typed graph
typed graph to scheduler/view IR
view IR to target IR
target IR to platform artifact
```

Core rule:

```txt
Renderer lowers pure view data. Scheduler remains the owner of state and event semantics.
```
