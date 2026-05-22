Berikut ADR yang fokus ke **template dan view model** Nova.

---

# ADR-006: Template & View Model

## Status

Implemented / Accepted

## Context

Nova perlu menghasilkan UI lintas platform dari deklarasi yang sama.

ADR-001 menetapkan bahwa `<template>`:

```txt
pure
declarative
boleh membaca state/props
boleh emit scheduler event
tidak boleh dirty operation
```

ADR ini memperjelas model view, binding, event emission, component/capability usage,
dan hubungan template dengan renderer pipeline.

---

# Decision

Template Nova dikompilasi menjadi **renderer-neutral view IR** yang cukup kaya untuk
diturunkan langsung ke strategi render target.

Template tidak mengenal:

```txt
DOM
UIView
native view node (Compose hanya pada @nova/android-compose deprecated)
SwiftUI View
desktop widget object
platform handle
```

Template hanya menghasilkan deklarasi:

```txt
node tree
attributes
bindings
event routes
children
component usage
list projection
conditional projection
```

Renderer adapter menurunkan view IR ke platform target.

Secara arsitektur, template evaluation adalah bagian dari **functional core**:

```txt
StateSnapshot + Props + Template
  -> ViewIR + DependencyMetadata + Diagnostics
```

Renderer adalah **hexagonal outbound port**. Web, Android, iOS, dan Desktop adapter boleh memilih
strategi render masing-masing, tetapi tidak boleh mengubah semantic view IR, event route, atau state
dependency yang dihasilkan core.

Untuk target Web, lowering boleh menghasilkan:

```txt
SSR HTML
SSG HTML
CSR DOM construction/update code
event route bootstrap
static/dynamic slot metadata
```

Jadi Web dapat berakhir sebagai HTML/DOM, tetapi Nova tetap melewati view IR agar semantic,
diagnostics, event route, dan state dependency tetap sama dengan target lain.

Untuk Android production, lowering menghasilkan binding Java ke Android View framework. Bentuk final
Android production memakai View tree Java; Compose hanya pada renderer compatibility deprecated
adapter dan harus tetap mengikuti contract view IR.

---

# Template Purity

Template boleh:

```txt
membaca state lokal
membaca imported state
membaca props
memanggil pure func
membentuk view tree
emit scheduler event
memetakan emitted event capability anak
```

Template tidak boleh:

```txt
memanggil external operation
mengakses time/random/platform API
menulis state
menjalankan lifecycle
membaca platform object
melakukan imperative render command
```

Invalid:

```nova
<template>
  <button on_press -> storage.set>
    "Save"
  /|
/|
```

---

# Functional View Model

Template adalah pure projection.

Input:

```txt
Template
StateSnapshot
Props
ImportedStateSnapshot
PureFunctionEnvironment
```

Output:

```txt
ViewIR
DependencyMetadata
Diagnostics[]
```

Rule:

```txt
1. Evaluasi template tidak mengubah state.
2. Evaluasi template tidak menjalankan lifecycle.
3. Evaluasi template tidak memanggil external operation.
4. Evaluasi template tidak membaca object renderer atau platform.
5. Output template adalah data IR dan event route, bukan native view handle.
6. Pure function yang dipanggil template harus deterministic terhadap input data.
```

Konsekuensi:

```txt
template dapat dievaluasi ulang kapan saja
SSR/SSG/CSR/native lowering memakai semantic yang sama
diagnostic binding dapat dibuat sebelum runtime target berjalan
```

---

# Hexagonal Renderer Port

Renderer port menerima semantic view output dari core.

```txt
RendererPort {
  mount(view: ViewIR, metadata: DependencyMetadata)
  update(commit: StateCommit, metadata: DependencyMetadata)
  dispose()
}
```

Host/renderer event kembali ke core melalui inbound port:

```txt
platform event
  -> renderer/host adapter
  -> validate route payload
  -> scheduler enqueue event
```

Rule:

```txt
1. Renderer adapter memegang native node/handle, bukan template.
2. Template hanya menyimpan key, binding, event route, dan semantic props.
3. Event route adalah data contract, bukan closure imperative ke state.
4. Renderer cache boleh ada, tetapi hanya optimisasi adapter.
5. Adapter failure masuk error lifecycle atau host diagnostic.
```

---

# Target Annotation

Template dapat target-specific:

```nova
<template target <- web>
  ...
/|
```

Atau target-polymorphic:

```nova
<template>
  ...
/|
```

Rule:

```txt
1. Template tanpa target valid untuk target manapun jika semua node/capability tersedia.
2. Template dengan target hanya dipakai untuk target tersebut.
3. Jika ada template target-specific dan polymorphic, resolver memilih target-specific.
4. Jika target-specific tidak ada, resolver memakai polymorphic template.
5. Jika tidak ada template valid, build target gagal.
```

---

# Binding Syntax

Data binding memakai `<-`.

```nova
<text value <- user.name /|
<button disabled <- isLoading /|
```

Event route memakai `->`.

```nova
<button on_press -> @submit /|
```

Rule:

```txt
1. `<-` mengevaluasi pure expression.
2. `->` hanya valid untuk event slot atau emitted event mapping.
3. Event target harus scheduler event.
4. Event payload harus sesuai type.
```

---

# Event Emission

Event dari template selalu masuk scheduler queue melalui host/renderer adapter.

```nova
<button on_press -> @increment>
  "+"
/|
```

Dengan payload:

```nova
<button on_press -> @select(item)>
  <text value <- item.name /|
/|
```

Canonical meaning:

```txt
platform event -> renderer event adapter -> scheduler enqueue @select(payload)
```

Template tidak menjalankan transition langsung.

---

# Capability Usage

Capability dengan `<contract capability>` dapat dipakai sebagai component.

```nova
<contract capability Button>
  props {
    label: string;
    disabled?: boolean;
  }

  emits {
    @pressed: void;
  }
/|
```

Usage:

```nova
<Button label <- "Save" @pressed -> @save /|
```

Rule:

```txt
1. Props memakai data binding `<-`.
2. Emitted event mapping memakai `->`.
3. Payload emitted event diteruskan ke scheduler event tujuan.
4. Props dan emitted event harus sesuai contract capability.
5. Component usage tidak memberi akses ke lifecycle internal component.
```

---

# View Node Model

Renderer-neutral node:

```txt
ViewNode {
  kind: NodeKind | CapabilityRef
  target?: Target
  props: BindingMap
  events: EventRouteMap
  children: ViewNode[]
  key?: DataValue
}
```

Node kind dapat berupa:

```txt
primitive node dari renderer target
capability component
text node
list projection
conditional projection
slot/content projection
```

Primitive node harus didefinisikan oleh renderer capability package, misalnya `@nova/ui`.

---

# Lists

List projection harus pure.

Contoh:

```nova
<list items <- collection>
  <item key <- item.id>
    <text value <- item.brand + " " + item.model /|
  /|
/|
```

Rule:

```txt
1. `items` harus array.
2. `item` adalah binding lokal dalam body list.
3. `key` direkomendasikan untuk item yang dapat reorder.
4. Renderer boleh memberi warning jika list tidak memiliki key stabil.
5. List body tidak boleh dirty operation.
```

Jika key tidak diberikan, renderer boleh memakai index sebagai fallback tetapi harus dianggap kurang stabil.

---

# Conditional View

Conditional projection adalah pure branch.

Syntax final belum dikunci, tetapi semantic IR-nya:

```txt
if condition:
  render branch A
else:
  render branch B
```

Rule:

```txt
1. condition harus boolean.
2. kedua branch harus pure view declaration.
3. conditional tidak boleh menjalankan lifecycle.
4. renderer boleh preserve identity jika key sama.
```

Syntax conditional akan distandarkan di language grammar lanjutan.

---

# Text

Text adalah data binding ke string.

```nova
<text value <- "Total: " + count /|
```

Rule:

```txt
1. Text node value harus string.
2. Non-string harus diformat oleh pure func eksplisit.
3. Renderer tidak boleh melakukan locale formatting implisit.
```

Contoh:

```nova
<text value <- count |> formatNumber locale <- "id-ID" /|
```

---

# Renderer Invalidation

Template dievaluasi terhadap state snapshot.

Lowering template harus menghasilkan dependency metadata:

```txt
state -> template
state -> binding
binding -> node/slot
event route -> node
```

Scheduler commit menghasilkan invalidation set:

```txt
StateCommit.Invalidations
```

Renderer adapter tidak melakukan hydration stage atau generic tree diff untuk mencari apa yang
berubah. IR/target IR harus sudah cukup smart: dependency graph dan dynamic slot metadata memberi
tahu renderer binding/node/slot mana yang perlu di-update.

Rule:

```txt
1. Renderer tidak memiliki state sumber kebenaran sendiri.
2. Renderer cache hanya optimisasi.
3. Render ulang tidak boleh menghasilkan side effect userland.
4. Event handler di view hanya enqueue scheduler event.
5. Web SSR/SSG tidak memiliki hydration stage; bootstrap interaktivitas hanya memasang event route dari metadata.
6. CSR update memakai compiled dependency/slot metadata, bukan runtime diff umum.
7. Renderer update dipanggil melalui RendererPort dengan commit dan metadata dari core.
```

---

# Accessibility and Semantics

Nova template harus membawa semantic props ketika tersedia.

Contoh:

```nova
<button label <- "Save" disabled <- isSaving on_press -> @save /|
```

Rule:

```txt
1. Renderer package harus mendefinisikan props accessibility target-neutral.
2. Target adapter memetakan semantic props ke platform.
3. Compiler boleh memberi warning untuk control tanpa label semantic.
4. Accessibility bukan dekorasi renderer, melainkan bagian dari view contract.
```

---

# Alternatives Considered

## Direct Platform Template Without IR

Template web menjadi HTML/DOM dan Android menjadi Java View tree tanpa melewati view IR bersama (anti-pattern).

Rejected because:

```txt
Core template semantic akan berbeda per target.
Diagnostics dan type checking sulit disatukan.
Renderer-neutral IR dibutuhkan untuk conformance.
```

Clarification:

```txt
Web target tetap boleh menghasilkan SSR/SSG HTML atau CSR DOM code.
Yang ditolak adalah melewati IR bersama dan membiarkan tiap target mendefinisikan semantic template sendiri.
```

## Renderer-Owned Event Handling

Renderer langsung memanggil state update saat user berinteraksi.

Rejected because:

```txt
State commit harus scheduler-owned.
Template hanya boleh emit scheduler event.
```

## Template Allows Effects

Template boleh melakukan effect saat render.

Rejected because:

```txt
Render dapat diulang berkali-kali.
Side effect akan tidak deterministic.
Lifecycle sudah menyediakan dirty boundary.
```

---

# Consequences

## Positive

```txt
1. UI semantic tetap platform-neutral.
2. Renderer adapter dapat berkembang per target.
3. Template aman dievaluasi ulang.
4. Event flow tetap melewati scheduler.
5. Capability component memiliki props/emits contract yang jelas.
```

## Negative

```txt
1. Renderer package harus mendefinisikan primitive node contract.
2. Ada tahap view IR sebelum platform render.
3. Platform-specific UI kadang membutuhkan template target-specific.
4. Conditional syntax masih perlu dikunci dalam grammar.
5. Renderer/lowering harus menjaga dependency metadata agar tidak jatuh ke hydration/diffing umum.
```

---

# Final Position

Nova template adalah:

```txt
pure view declaration
renderer-neutral IR source
compiled dependency/slot metadata
state/props projection
scheduler event emitter
component contract consumer
functional view model
hexagonal renderer port input
```

Core rule:

```txt
Template describes what should be seen and what event should be emitted.
Renderer decides how to show it on the target without changing template semantics.
```
