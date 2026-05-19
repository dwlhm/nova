Berikut ADR yang fokus ke **app lifecycle dan navigation model** Nova.

---

# ADR-018: App Lifecycle & Navigation Model

## Status

Implemented / Accepted

## Context

ADR-005 mendefinisikan lifecycle capability:

```txt
mount
dispose
before @event
after @event
error
```

Itu cukup untuk dirty boundary per capability, tetapi framework penuh juga membutuhkan model
aplikasi:

```txt
startup
resume/pause
route/url/back stack
deep link
multiple screens
state restoration
```

Web dan Android memiliki konsep lifecycle dan navigasi berbeda:

```txt
web      -> URL, History API, page visibility
android  -> Activity lifecycle, back stack, intent/deep link
```

Nova membutuhkan semantic target-neutral tanpa menambah dirty zone baru.

---

# Decision

Nova memodelkan lifecycle aplikasi dan navigasi sebagai **framework capabilities**:

```txt
@nova/app
@nova/navigation
```

Lifecycle aplikasi dan navigasi tidak menjadi construct bahasa baru.
Keduanya masuk scheduler sebagai event dan state data biasa.

Rule:

```txt
1. Root capability tetap dimount melalui lifecycle mount.
2. Host lifecycle platform diterjemahkan menjadi scheduler event.
3. Navigation adalah serializable state + event route, bukan native router object.
4. Programmatic navigation adalah dirty operation framework yang melewati adapter.
5. Back/deep link/browser URL masuk scheduler sebelum mengubah semantic state final.
```

---

# App Instance

Runtime menjalankan satu atau lebih app instance.

```txt
AppInstance {
  id: AppInstanceId
  root: CapabilityRef
  target: TargetId
  scheduler: SchedulerRuntime
  renderer: RendererPort
  host: HostAdapter
  external: ExternalOperationPort
}
```

MVP:

```txt
1 app instance per browser tab
1 app instance per Android Activity host
```

Future:

```txt
multi-window desktop
Android multi-activity shell
iOS scene sessions
```

---

# Root Mount

Runtime boot flow:

```txt
load artifact
validate ABI/runtime version
create scheduler
create renderer
create host adapter
restore optional snapshot
mount root capability
enqueue @app_started if @nova/app active
render initial view
```

Rule:

```txt
1. mount lifecycle runs once per capability instance lifetime.
2. renderer remount due to platform recreation does not imply capability mount.
3. dispose lifecycle runs when capability instance is removed or app instance closes.
4. app lifecycle events do not bypass scheduler ordering.
```

---

# App Lifecycle Events

Standard events from `@nova/app`:

```txt
@app_started(void)
@app_resumed(void)
@app_paused(void)
@app_stopped(void)
@app_restored(snapshot: unknown)
```

Target mapping:

```txt
web:
  page load/pageshow       -> @app_started or @app_resumed
  visibility visible       -> @app_resumed
  visibility hidden        -> @app_paused
  pagehide                 -> @app_stopped

android:
  process/activity start   -> @app_started
  onResume                 -> @app_resumed
  onPause                  -> @app_paused
  onStop                   -> @app_stopped
  saved snapshot restore   -> @app_restored
```

Rule:

```txt
1. Target adapter may coalesce duplicate lifecycle events.
2. Coalescing must preserve logical order.
3. Lifecycle event payload must be Nova data.
4. Capability must import or opt into app lifecycle events before handling them.
```

---

# Navigation State

Navigation state is data.

```txt
Route {
  path: string;
  params?: record;
  query?: record;
  fragment?: string;
}

NavigationStack {
  current: Route;
  entries: Route[];
}
```

Navigation events:

```txt
@route_changed(route: Route)
@navigate(action: NavigationAction)
@navigation_failed(message: string)
```

Navigation action:

```txt
NavigationAction {
  kind: "push" | "replace" | "back";
  route?: Route;
}
```

---

# Navigation Flow

User navigation:

```txt
platform navigation input
  -> host adapter validates route
  -> enqueue @route_changed(route)
  -> transition updates route state
  -> renderer invalidation
  -> navigation adapter reconciles platform URL/back stack
```

Programmatic navigation:

```txt
lifecycle emits @navigate(action)
  -> transition plans navigation state
  -> commit
  -> navigation adapter updates platform URL/back stack
  -> completion or failure event
```

Rule:

```txt
1. Route state is source of truth after commit.
2. Adapter may update platform URL/back stack only after successful commit.
3. Back navigation can be rejected by transition guard.
4. Deep link payload must be parsed into Route data before enqueue.
5. Invalid route becomes diagnostic or @navigation_failed.
```

---

# Screen Composition

Nova does not add a separate screen construct.

Screen is represented by:

```txt
route state
conditional template projection
capability composition
target-specific template when needed
```

Example:

```nova
<contract state Router>
  route: Route <- { path: "/" } {
    @route_changed(next: Route) -> next;
  };
/|
```

Rule:

```txt
1. Screen transition is state transition.
2. Animation is renderer adapter concern unless exposed by @nova/ui contract.
3. Navigation side effects stay in lifecycle or adapter.
```

---

# Platform Reconciliation

Web:

```txt
Route.path/query/fragment <-> URL
NavigationStack action    <-> History API
browser back/forward      -> @route_changed(route)
```

Android:

```txt
Route                       <-> deep link compatible route data
NavigationStack action      <-> Android back stack adapter
system back                 -> @navigate({ kind: "back" })
intent/deep link            -> @route_changed(route)
```

Rule:

```txt
1. Platform native router state is mirror, not semantic source of truth.
2. Reconciliation failure is routed as @navigation_failed or SchedulerError.
3. URL/back stack updates must not create infinite event loops.
```

---

# Consequences

Keuntungan:

```txt
tidak perlu construct bahasa baru
Web URL dan Android back stack dapat berbagi route model
navigation bisa diuji lewat scheduler conformance
app lifecycle tetap berada di event queue yang sama
```

Trade-off:

```txt
navigation model awal sengaja sederhana
fitur nested router/modal/native transition butuh contract tambahan
adapter harus hati-hati mencegah loop URL/back-stack
```
