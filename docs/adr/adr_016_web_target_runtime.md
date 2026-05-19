Berikut ADR yang fokus ke **runtime target Web** Nova.

---

# ADR-016: Web Target Runtime

## Status

Implemented / Accepted

## Context

ADR-014 menetapkan target awal Nova:

```txt
web
android
```

ADR-006 dan ADR-008 menetapkan template diturunkan ke ViewIR dan TargetIR, bukan langsung
bergantung pada DOM. Web tetap membutuhkan runtime nyata untuk browser, dev server, event DOM,
artifact bundling, permission, dan optional hydration.

---

# Decision

Target Web Nova menggunakan runtime native TypeScript/JavaScript yang berjalan di browser.

Komponen target Web:

```txt
nova-web-runtime
nova-web-dom-renderer
nova-web-host-adapter
nova-web-external-adapter
nova-web-dev-server
nova-web-artifact-builder
```

Output utama MVP:

```txt
CSR browser application
```

Output supported setelah MVP:

```txt
SSR HTML + hydration
SSG static HTML + hydration
PWA packaging
```

Runtime Web tidak menjadi semantic source of truth. Runtime menjalankan ABI yang dihasilkan
compiler:

```txt
SchedulerIR
ViewIR or WebTargetIR
EventRouteTable
ExternalOperationTable
PermissionTable
SourceMap
```

---

# Web Runtime Units

## Scheduler Runtime

Scheduler Web mengimplementasikan ADR-002.

Rule:

```txt
1. LogicalSequence dibuat oleh scheduler, bukan timestamp browser.
2. Browser event masuk sebagai EventEnvelope setelah validasi route.
3. Promise completion masuk sebagai event baru, tidak nested commit.
4. Scheduler commit tetap atomic per logical event.
5. Microtask/macrotask browser tidak boleh mengubah urutan logical queue.
```

## Host Adapter

Host adapter memetakan:

```txt
DOM event
History API event
visibility/page lifecycle event
network completion event
storage completion event
```

menjadi:

```txt
EventEnvelope
SchedulerError
```

Host adapter tidak boleh menulis state langsung.

## DOM Renderer

DOM renderer menerima:

```txt
ViewIR/WebTargetIR
StateCommit
DependencyMetadata
EventRouteTable
```

dan menghasilkan:

```txt
DOM mount/update/dispose
event listener registration
accessibility attributes
render diagnostics
```

DOM node tidak pernah menjadi Nova data.

---

# Artifact Model

Build Web menghasilkan:

```txt
build/web/
  index.html
  assets/
  nova-runtime.js
  app.bundle.js
  app.nova-ir.json
  app.source-map.json
  permissions.json
  target-manifest.json
```

MVP boleh menggabungkan file runtime dan app bundle, tetapi semantic metadata harus tetap
dapat diinspeksi oleh tooling.

Artifact metadata:

```txt
target: web
languageVersion
abiVersion
schedulerVersion
viewIrVersion
runtimeVersion
entryCapability
permissions
externalOperations
```

---

# Rendering Strategy

Web MVP memakai DOM renderer imperative yang dikendalikan oleh ViewIR.

Rule:

```txt
1. ViewIR node key dipakai untuk stabilitas list.
2. Renderer boleh diff DOM sebagai optimisasi.
3. Renderer tidak boleh mengubah event payload.
4. Renderer tidak boleh skip logical commit.
5. Renderer failure diarahkan ke error lifecycle atau host diagnostic.
```

Primitive mapping awal:

```txt
text        -> Text node or span
button      -> button
image       -> img
list/item   -> keyed container children
surface     -> div/section based on role
row/column  -> div with layout class
scroll      -> div with overflow semantics
input       -> input/textarea/select via @nova/forms
```

Mapping final boleh berubah selama semantic, accessibility, dan event route tetap sama.

---

# Event Mapping

DOM event mapping:

```txt
click       -> on_press
input       -> on_change
change      -> on_change
submit      -> on_submit
focus       -> on_focus
blur        -> on_blur
keydown     -> declared keyboard route if enabled
```

Rule:

```txt
1. Event listener hanya dipasang untuk route yang ada di EventRouteTable.
2. Payload disusun dari binding dan DOM value yang dikontrak.
3. Payload divalidasi sebelum enqueue.
4. Disabled control tidak boleh mengirim on_press/on_submit.
5. Undeclared DOM event diabaikan atau dilaporkan sebagai diagnostic dev.
```

---

# External Adapter

External Web implementation dapat berupa:

```txt
project-local JavaScript/TypeScript
package adapter
@env/* browser adapter
```

Naming convention:

```txt
name.web.js
name.web.ts
platform/web/name.web.js
platform/web/name.web.ts
```

Rule:

```txt
1. External operation selalu dipanggil dari lifecycle shell.
2. Promise result divalidasi terhadap output contract.
3. Rejection menjadi failure event atau SchedulerError.
4. Adapter tidak boleh mengembalikan DOM node, Promise, function, class instance, atau native handle.
5. Adapter code dan required permissions dicatat di audit manifest.
```

---

# Permissions

Web permission mapping awal:

```txt
storage.read/write     -> localStorage/IndexedDB policy
network.request        -> URL allowlist/CSP guidance
clipboard.read/write   -> Clipboard API permission/user gesture
notification.send      -> Notification permission
device.info            -> limited browser environment metadata
```

Rule:

```txt
1. Build-time permission audit wajib lulus sebelum artifact dibuat.
2. Runtime browser permission denial harus observable.
3. Browser capability unavailable menjadi SchedulerError atau failure event.
4. Generated artifact harus menyertakan permission summary untuk review.
```

---

# Hydration and SSR

CSR adalah output awal. SSR/SSG memakai kontrak yang sama:

```txt
server render ViewIR -> HTML + hydration manifest
browser load -> validate hydration manifest
attach event route -> resume scheduler snapshot
```

Rule:

```txt
1. HTML server-rendered bukan source of truth semantic.
2. Hydration mismatch menjadi diagnostic.
3. Event sebelum hydration selesai dapat di-buffer atau diblokir sesuai runtime mode.
4. Dirty lifecycle mount hanya berjalan sesuai policy hydration yang eksplisit.
```

Detail hydration lintas target didefinisikan di ADR-019.

---

# Dev Server

Web dev server wajib menyediakan:

```txt
incremental compile
diagnostic overlay
hot reload
state-preserving reload when compatible
source-mapped runtime errors
target manifest inspection
permission audit panel
```

Hot reload tidak boleh melewati validation:

```txt
source edit -> compile -> ABI compatibility check -> runtime patch or full reload
```

---

# Consequences

Keuntungan:

```txt
Web runtime native terhadap browser
DOM detail tetap di adapter
CSR MVP sederhana tetapi jalur SSR/hydration tersedia
diagnostics dan event ordering tetap sama dengan Android
```

Trade-off:

```txt
DOM renderer harus menjaga source map dan event route dengan disiplin
SSR/hydration menambah kontrak snapshot
browser permission model tidak selalu sama dengan manifest permission Nova
```
