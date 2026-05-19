Berikut ADR yang fokus ke **runtime target Android** Nova.

---

# ADR-017: Android Target Runtime

## Status

Implemented / Accepted

## Context

Nova menargetkan Android sebagai target awal bersama Web.

ADR-002 menetapkan scheduler semantics harus platform-neutral.
ADR-006 dan ADR-008 menetapkan template diturunkan ke ViewIR/TargetIR.
ADR-014 menetapkan runtime Android harus native Kotlin/JVM Android.

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

Target Android Nova menggunakan runtime native Kotlin.

Komponen target Android:

```txt
nova-android-runtime
nova-android-compose-renderer
nova-android-host-adapter
nova-android-external-adapter
nova-android-gradle-plugin
nova-android-artifact-builder
```

Renderer Android MVP memakai **Jetpack Compose adapter**.

Reason:

```txt
Compose bersifat deklaratif
ViewIR tree dapat dipetakan ke composable tree
state invalidation dapat dihubungkan ke snapshot state adapter
accessibility semantics dapat dipetakan dengan jelas
```

Nova core tetap tidak mengenal Compose object.

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
Compose UI event
permission result
intent/deep link
external operation completion
```

menjadi:

```txt
EventEnvelope
SchedulerError
```

## Compose Renderer

Compose renderer menerima:

```txt
ViewIR/AndroidTargetIR
StateCommit
DependencyMetadata
EventRouteTable
```

dan menghasilkan:

```txt
Composable tree
state-backed recomposition trigger
event callback bridge
accessibility semantics
render diagnostics
```

Compose state dan Modifier tidak pernah menjadi Nova data.

---

# Artifact Model

Build Android menghasilkan salah satu bentuk:

```txt
generated Gradle module
Android library module
app project integration
```

Recommended output:

```txt
build/android/
  nova-ir/
    app.nova-ir.json
    app.source-map.json
    permissions.json
    target-manifest.json
  generated/
    NovaApp.kt
    NovaRoutes.kt
    NovaExternalBindings.kt
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

Konfigurasi wajib target Android:

```txt
application_id
namespace
compile_sdk
min_sdk
target_sdk
version_code
version_name or project.version
gradle_plugin
kotlin_plugin
compose_compiler_plugin
compose_bom
activity_compose
material3
theme
theme_parent
java_version
label or project.name
```

`namespace` dan package generated code boleh tetap stabil untuk kebutuhan compiler/runtime, tetapi
Android install identity adalah `applicationId`. Dua project Nova berbeda harus menghasilkan
`applicationId` berbeda agar Android tidak menganggap APK sebagai update dari app lain.

Rule:

```txt
1. Artifact builder tidak boleh menurunkan applicationId dari nama example.
2. Versi Gradle, SDK, dependency Compose, namespace, theme, Java version, dan label harus berasal dari manifest project.
3. Jika konfigurasi wajib kosong, build gagal dengan diagnostic.
4. Nilai fallback hanya boleh berasal dari field user-side lain, misalnya versionName dari project.version.
```

---

# Rendering Strategy

Primitive mapping awal:

```txt
text          -> Text
button        -> Button
image         -> Image or AsyncImage adapter if package enabled
list/item     -> LazyColumn/LazyRow with stable key
surface       -> Surface/Box
row           -> Row
column        -> Column
stack         -> Box
scroll        -> scrollable container
text_input    -> TextField
toggle        -> Switch
slider        -> Slider
```

Rule:

```txt
1. Compose renderer memakai stable key dari ViewIR untuk list.
2. Recomposition adalah optimisasi renderer, bukan semantic scheduler.
3. Event callback hanya enqueue event melalui host adapter.
4. Renderer tidak boleh membaca atau menulis state cell langsung.
5. Renderer diagnostics memakai source map dari ViewIR.
```

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
project-local Kotlin
package adapter
@env/* Android adapter
```

Naming convention:

```txt
name.android.kt
platform/android/name.android.kt
```

Operation bridge:

```txt
Nova data input
  -> Kotlin adapter input DTO
  -> suspend or immediate operation
  -> Nova data output validation
  -> scheduler completion event
```

Forbidden output:

```txt
Context
Activity
View
Composable lambda
CoroutineScope
Job
Flow
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
Android target terasa native
Compose cocok dengan declarative ViewIR
threading Android tetap di adapter
state restoration punya kontrak yang sama dengan Web hydration
```

Trade-off:

```txt
Compose menjadi pilihan renderer MVP Android
Gradle plugin dan generated Kotlin harus dijaga kompatibilitasnya
Android lifecycle lebih kompleks dari Web dan membutuhkan conformance fixture khusus
```
