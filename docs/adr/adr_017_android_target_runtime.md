Berikut ADR yang fokus ke **runtime target Android** Nova.

---

# ADR-017: Android Target Runtime

## Status

Implemented / Accepted

## Context

Nova menargetkan Android sebagai target awal bersama Web.

ADR-002 menetapkan scheduler semantics harus platform-neutral.
ADR-006 dan ADR-008 menetapkan template diturunkan ke ViewIR/TargetIR.
ADR-014 menetapkan runtime Android harus native JVM/Android framework. Production memakai Java
source.

Android memiliki:

```txt
Activity lifecycle
process lifecycle
main thread / Looper
coroutines
permission runtime
configuration changes
back navigation
resource packaging
```

Detail tersebut tidak boleh bocor ke core Nova.

---

# Decision

Target Android Nova menggunakan runtime native **Java** di atas Android SDK.

Komponen target Android:

```txt
nova-android-runtime          (Java: NovaRuntime, scheduler bridge)
nova-android-view-renderer    (Java: View/ViewGroup primitive lowering)
nova-android-host-adapter     (Activity, back stack, lifecycle)
nova-android-env-adapters     (@env/* -> platform API)
nova-android-artifact-builder (Gradle project generation)
```

Renderer production Android: **`@nova/android`** — native View renderer berbasis Java.

Nova core tetap tidak mengenal `android.view.View` atau object platform secara langsung; hanya ViewIR
dan ABI contract.

---

# Android Runtime Units

## Scheduler Runtime

Scheduler Android mengimplementasikan ADR-002.

Rule:

```txt
1. Scheduler punya satu logical queue per app instance.
2. Commit state berjalan pada scheduler dispatcher yang konsisten.
3. Renderer update yang menyentuh UI dijadwalkan ke Main dispatcher.
4. Coroutine completion masuk sebagai scheduler event baru.
5. Android lifecycle callback tidak boleh menjalankan nested state commit.
```

## Host Adapter

Host adapter memetakan:

```txt
Activity lifecycle
ProcessLifecycleOwner event
back navigation
View/button UI event
permission result
intent/deep link
external operation completion
```

menjadi:

```txt
EventEnvelope
SchedulerError
```

## Native View Renderer (production)

Renderer `@nova/android` menerima:

```txt
ViewIR
StateCommit + Invalidations
DependencyMetadata
EventRouteTable
```

dan menghasilkan:

```txt
Java MainActivity + NovaRuntime
Android View/ViewGroup tree
granular binding update (applyBindings)
event callback bridge (dispatch)
page visibility dari route state
```

Platform View object tidak pernah menjadi Nova data.

# Artifact Model

Build Android production menghasilkan:

```txt
generated Gradle module (Java source)
app project integration
```

Recommended output (`@nova/android`):

```txt
build/android/
  nova-ir/
    app.nova-ir.json
    app.source-map.json
    permissions.json
    target-manifest.json
  app/src/main/java/<namespace>/
    MainActivity.java
    NovaRuntime.java
  generated/
    NovaApp.java
    NovaRoutes.java
    NovaExternalBindings.java
  build.gradle.kts
```

Final app packaging dapat menghasilkan:

```txt
APK
AAB
Android library/AAR
```

tergantung mode project.

---

# Android App Identity

APK identity harus berasal dari konfigurasi target di project manifest, bukan dari contoh atau runtime package.

Konfigurasi wajib target Android (`@nova/android`):

```txt
application_id
namespace
compile_sdk
min_sdk
target_sdk
version_code
gradle_plugin
theme
theme_parent
java_version
label or project.name
version_name or project.version (optional)
```

`namespace` dan package generated code boleh tetap stabil untuk kebutuhan compiler/runtime, tetapi
Android install identity adalah `applicationId`. Dua project Nova berbeda harus menghasilkan
`applicationId` berbeda agar Android tidak menganggap APK sebagai update dari app lain.

Rule:

```txt
1. Artifact builder tidak boleh menurunkan applicationId dari nama example.
2. Versi Gradle, SDK, namespace, theme, Java version, dan label harus berasal dari manifest project.
3. Jika konfigurasi wajib kosong, build gagal dengan diagnostic NVA-ANDROID-001.
4. Production Android tidak memuat dependency UI tambahan di luar primitive Nova.
```

---

# Rendering Strategy

Primitive mapping production (`@nova/android`):

```txt
text          -> TextView
button        -> Button
image         -> ImageView (atau adapter @env jika perlu)
list/item     -> repeated child views dengan stable key metadata
surface       -> LinearLayout container
row           -> LinearLayout HORIZONTAL
column        -> LinearLayout VERTICAL
stack         -> FrameLayout
scroll        -> ScrollView
text_input    -> EditText (future primitive)
toggle        -> Switch (future primitive)
slider        -> SeekBar (future primitive)
```

Rule:

```txt
1. Native View renderer memakai invalidation granular dari StateCommit.
2. Rebuild View tree penuh hanya pada mount awal; update state memakai applyBindings.
3. Event callback hanya enqueue event melalui dispatch() di MainActivity.
4. Renderer tidak boleh membaca atau menulis state cell di luar scheduler contract.
5. Renderer diagnostics memakai source map dari ViewIR.
```

APK size strategy dan opsi renderer Android native View dibahas terpisah di ADR-025.

---

# Threading and Coroutines

Android runtime memakai coroutine sebagai implementation detail.

Rule:

```txt
1. Lifecycle dirty operation boleh suspend di adapter.
2. Suspension tidak terlihat di source language.
3. Completion selalu menjadi event baru atau SchedulerError.
4. State commit tidak boleh terjadi dari coroutine callback di luar scheduler loop.
5. Adapter harus membatalkan operation saat app instance dispose jika operation terikat lifecycle.
```

Dispatcher:

```txt
scheduler dispatcher  -> logical queue execution
main dispatcher       -> UI rendering and platform lifecycle callbacks
io dispatcher         -> storage/network adapter implementation
```

Urutan logical tetap ditentukan oleh scheduler sequence.

---

# Android Lifecycle Mapping

Mapping awal:

```txt
Activity.onCreate       -> root capability mount
Activity.onStart        -> @app_started or @app_resumed depending process state
Activity.onResume       -> @app_resumed
Activity.onPause        -> @app_paused
Activity.onStop         -> @app_stopped
Activity.onDestroy      -> dispose if finishing or app instance closed
configuration change    -> renderer remount with retained scheduler snapshot
process recreation      -> restore from HydrationSnapshot if available
```

Rule:

```txt
1. Root mount/dispose mengikuti capability lifetime, bukan setiap recomposition.
2. Configuration change tidak otomatis reset scheduler state.
3. Process death restoration memakai snapshot contract, bukan native object.
4. Lifecycle events masuk queue dan tetap tunduk pada event contract.
```

---

# Permissions

Android permission mapping awal:

```txt
storage.read/write      -> app-scoped storage/DataStore, no broad file permission by default
network.request         -> INTERNET manifest permission + allowlist policy
clipboard.read/write    -> ClipboardManager policy
notification.send       -> POST_NOTIFICATIONS on supported Android versions
device.info             -> safe Build/config fields only
```

Rule:

```txt
1. Build menghasilkan AndroidManifest entries yang diperlukan.
2. Runtime permission prompt dilakukan oleh adapter saat operation membutuhkan.
3. Permission denial menjadi failure event atau SchedulerError.
4. Nova tidak memberi akses arbitrary Intent atau Context ke userland.
```

---

# External Adapter

External Android implementation dapat berupa:

```txt
project-local Java
package adapter
@env/* Android adapter
```

Naming convention (production):

```txt
name.android.java
platform/android/name.android.java
```

Operation bridge:

```txt
Nova data input
  -> Java adapter (Map/POJO serializable)
  -> background thread jika perlu (di dalam adapter)
  -> Nova data output validation
  -> scheduler completion event
```

Forbidden output:

```txt
Context
Activity
View
platform callback handle
File descriptor
Socket
```

---

# Back Navigation

Android back navigation maps to `@nova/navigation`.

Rule:

```txt
1. Back press becomes scheduler event.
2. Navigation state decides whether back is consumed.
3. Adapter updates native back stack after scheduler commit.
4. Deep link intent becomes route data after validation.
```

---

# Consequences

Keuntungan:

```txt
Android target terasa native dengan APK lebih kecil
ViewIR tetap renderer-neutral
threading Android tetap di @env adapter
state restoration punya kontrak yang sama dengan Web hydration
```

Trade-off:

```txt
Native View renderer membutuhkan pemeliharaan primitive layout sendiri
Android lifecycle lebih kompleks dari Web dan membutuhkan conformance fixture khusus
```
