Berikut ADR yang fokus ke **standard package surface** Nova.

---

# ADR-015: Standard Package Surface

## Status

Implemented / Accepted

## Context

ADR-003 menetapkan module/capability system.
ADR-007 menetapkan platform capability.
ADR-012 menetapkan namespace reserved:

```txt
@nova/*
@env/*
```

Namun Nova sebagai framework penuh membutuhkan standard package surface yang jelas agar aplikasi
yang sama dapat dibangun untuk target awal:

```txt
web
android
```

Standard package tidak boleh membocorkan DOM, Android View, Compose object, atau platform handle
ke core Nova.

---

# Decision

Nova membagi package resmi menjadi dua keluarga:

```txt
@nova/*  framework package target-neutral atau renderer-neutral
@env/*   environment capability yang dipenuhi target adapter
```

MVP standard package:

```txt
@nova/core
@nova/ui
@nova/forms
@nova/navigation
@nova/app
@env/storage
@env/network
@env/clipboard
@env/notify
@env/device
```

Rule:

```txt
1. @nova/* berisi contract, primitive, helper pure, dan capability target-neutral.
2. @env/* berisi dirty external capability contract.
3. @env/* hanya callable dari lifecycle.
4. Package official harus punya target manifest untuk web dan android.
5. Package official harus punya conformance fixture lintas target.
```

---

# Package Classes

## Pure Package

Pure package hanya berisi:

```txt
contract type
func
constant data contract
```

Contoh:

```txt
@nova/core
```

Pure package tidak membutuhkan permission.

## Renderer Package

Renderer package menyediakan primitive UI:

```txt
text
button
input
list
image
surface
scroll
stack
row
column
```

Contoh:

```txt
@nova/ui
@nova/forms
```

Renderer package tidak menulis state langsung. Event primitive selalu kembali melalui event route.

## Framework Capability Package

Framework capability package menyediakan event dan state contract target-neutral.

Contoh:

```txt
@nova/navigation
@nova/app
```

Ia dapat memakai host adapter, tetapi surface Nova tetap berupa data dan event.

## Environment Package

Environment package adalah dirty boundary.

Contoh:

```txt
@env/storage
@env/network
@env/clipboard
@env/notify
@env/device
```

Setiap operation wajib menyatakan permission dan failure mode.

---

# @nova/core

`@nova/core` menyediakan contract dan helper pure minimal.

Isi MVP:

```txt
Result<T, E>
Option<T>
JsonValue
MapEntry<K, V>
List helpers
String helpers
Number helpers
Record helpers
```

Rule:

```txt
1. Semua helper harus pure.
2. Helper tidak boleh membaca locale, clock, random, atau platform.
3. Helper yang membutuhkan locale masuk future package khusus.
4. JsonValue harus kompatibel dengan Nova data value.
```

---

# @nova/ui

`@nova/ui` adalah primitive renderer-neutral.

Primitive MVP:

```txt
text
button
image
list
item
surface
scroll
row
column
stack
spacer
```

Common props:

```txt
key?: string
role?: string
label?: string
enabled?: boolean
visible?: boolean
style?: StyleToken | StyleRecord
test_id?: string
```

Common events:

```txt
on_press -> @event
on_long_press -> @event
on_focus -> @event
on_blur -> @event
```

Rule:

```txt
1. Primitive harus bisa diturunkan ke DOM dan Compose.
2. Primitive tidak menjanjikan pixel-identical output.
3. Semantic role, label, enabled, dan focus harus dipertahankan.
4. List item dinamis harus memakai key stabil atau menghasilkan warning.
5. Style MVP adalah token/record data, bukan CSS string bebas atau Compose Modifier object.
```

---

# @nova/forms

`@nova/forms` menyediakan primitive input target-neutral.

Primitive MVP:

```txt
text_input
number_input
toggle
slider
select
radio_group
checkbox
form
field
```

Input event:

```txt
on_change -> @event(value)
on_submit -> @event(payload)
on_validate -> @event(payload)
```

Rule:

```txt
1. Input value selalu Nova data.
2. Native IME/detail platform tidak masuk event payload kecuali dikontrak sebagai data.
3. Android back/focus dan Web keyboard/submit dipetakan ke event yang sama jika semantic sama.
4. Validation pure dilakukan di func/transition, bukan di adapter.
```

---

# @nova/navigation

`@nova/navigation` menyediakan model routing target-neutral.

Core types:

```txt
Route {
  path: string;
  params?: record;
  query?: record;
  fragment?: string;
}

NavigationAction {
  kind: "push" | "replace" | "back";
  route?: Route;
}
```

Core events:

```txt
@route_changed(route: Route)
@navigation_failed(message: string)
```

Target mapping:

```txt
web      -> URL, History API
android  -> back stack, intent/deep link bridge
```

Rule:

```txt
1. Route adalah data state, bukan platform object.
2. Browser back dan Android back menghasilkan scheduler event.
3. Programmatic navigation adalah lifecycle dirty operation atau framework capability event.
4. Route resolver harus deterministic dan source-mapped.
```

---

# @nova/app

`@nova/app` menyediakan event app lifecycle target-neutral.

Events MVP:

```txt
@app_started(void)
@app_resumed(void)
@app_paused(void)
@app_stopped(void)
@app_restored(snapshot: unknown)
```

Target mapping:

```txt
web      -> page load, visibilitychange, pagehide, pageshow
android  -> Activity/Process lifecycle callbacks
```

Rule:

```txt
1. Events app lifecycle masuk scheduler queue seperti event lain.
2. Events ini tidak menambah dirty zone baru.
3. Root mount/dispose tetap mengikuti ADR-005.
4. Target adapter boleh mengirim event ini hanya jika capability mengimpor atau mengaktifkannya.
```

---

# @env/storage

Operations MVP:

```txt
get(key: string) -> unknown
set(key: string, value: unknown) -> void
remove(key: string) -> void
clear(scope?: string) -> void
```

Permissions:

```txt
storage.read
storage.write
```

Target mapping:

```txt
web      -> localStorage or IndexedDB adapter
android  -> DataStore adapter
```

Rule:

```txt
1. Storage value must be Nova data.
2. Adapter validates serialized payload on read.
3. Key scope should be project/package scoped.
4. Secure secrets require future secure storage capability, not generic storage.
```

---

# @env/network

Operations MVP:

```txt
request(input: RequestData) -> ResponseData
```

Core types:

```txt
RequestData {
  method: string;
  url: string;
  headers?: record;
  body?: unknown;
}

ResponseData {
  status: number;
  headers?: record;
  body?: unknown;
}
```

Permissions:

```txt
network.request
```

Rule:

```txt
1. URL allowlist dapat dideklarasikan di manifest.
2. Cookies/session native tidak otomatis terekspos sebagai Nova data.
3. Timeout dan cancellation adalah adapter concern dengan error terstruktur.
4. Response body unknown harus divalidasi sebelum menjadi type spesifik.
```

---

# Accessibility Requirement

Standard UI package wajib membawa metadata aksesibilitas.

Rule:

```txt
1. Interactive primitive harus punya accessible label dari text child, label prop, atau diagnostic.
2. Role semantic harus dipertahankan ke DOM/Compose.
3. Disabled state harus memengaruhi event route.
4. Focus order harus mengikuti tree order kecuali ada contract eksplisit.
5. Target adapter wajib memiliki snapshot accessibility minimal di conformance.
```

---

# Consequences

Keuntungan:

```txt
aplikasi tidak mengandalkan primitive platform langsung
web dan android punya surface yang sama sejak awal
permission package dapat diaudit
future target dapat mengimplementasikan @nova/* dan @env/* yang sama
```

Trade-off:

```txt
standard package harus konservatif
fitur platform unik perlu masuk lewat capability yang eksplisit
style dan input MVP tidak mengejar seluruh kemampuan CSS atau Android native
```
