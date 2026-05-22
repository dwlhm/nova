# ADR-025: Android APK Size And Renderer Strategy

## Status

Accepted

## Context

Fase eksperimen awal pernah memakai Jetpack Compose + Kotlin; production baseline (ADR-000, ADR-017,
ADR-025) beralih ke Java native View untuk mengecilkan APK dan memangkas lapisan UI.

Pembengkakan APK utama berasal dari dependency UI AndroidX, Compose runtime, Material3, resource,
dan DEX eksternal — bukan dari jumlah file generated Nova. Memecah output menjadi
`MainActivity.kt`, `NovaRuntime.kt`, `NovaRoutes.kt`, dan file generated lain tetap masuk source set
Kotlin yang sama dan tidak menambah overhead runtime berarti.

Nova membutuhkan strategi jangka panjang agar aplikasi sederhana yang hanya memakai primitive dasar
tidak perlu membawa dependency UI besar.

## Decision

Production Android memakai renderer native View milik Nova sebagai default `@nova/android`.
Compose/Kotlin (`@nova/android-compose`) hanya compatibility path yang deprecated.

Target arah:

```txt
ViewIR primitive -> Nova Android native View renderer -> Android View/ViewGroup/custom View ringan
```

Renderer native View harus dapat berjalan tanpa:

```txt
androidx.compose.*
androidx.activity:activity-compose
androidx.compose.material3:material3
org.jetbrains.kotlin.android
Kotlin stdlib/runtime pada APK final
dependency UI berat yang tidak dipakai oleh primitive app
```

Untuk mode APK kecil, renderer native View menghasilkan source Java di `app/src/main/java`.
Compose renderer boleh tetap memakai Kotlin selama mode compatibility, tetapi native View renderer
tidak boleh bergantung pada Kotlin plugin atau stdlib supaya baseline ukuran APK benar-benar
mengukur renderer Nova dan Android framework platform.

Implementasi production (lihat `internal/artifact/android.go`):

```txt
1. `@nova/android` menghasilkan Java di app/src/main/java tanpa Kotlin plugin.
2. Primitive ViewIR diturunkan ke Android View/ViewGroup + invalidation granular.
3. Material3, Compose, activity-compose, dan Kotlin stdlib tidak masuk dependency graph default.
4. Conformance + scheduler trace menjaga parity semantic dengan target web.
5. `@nova/android-compose` tetap ada untuk migrasi project lama, bukan default baru.
```

## Non Goals

```txt
Menghapus jalur Compose sebelum migrasi project lama selesai
Membuat clone Material3 lengkap di Nova
Mengorbankan semantic ViewIR atau event scheduler demi ukuran APK
```

## Consequences

Keuntungan:

```txt
APK untuk app sederhana dapat turun tanpa dependency UI besar
Nova punya kontrol penuh atas primitive rendering Android
Artifact Android menjadi lebih selaras dengan prinsip renderer milik Nova
```

Trade-off:

```txt
Native View renderer membutuhkan implementasi layout, styling, accessibility, dan invalidation sendiri
Scheduler hanya mengetahui state/capability invalidation; pemetaan ke View native tetap milik renderer
Parity semantic dengan jalur `@nova/android-compose` (deprecated) dijaga lewat conformance
Beberapa primitive kompleks mungkin tetap lebih murah memakai dependency eksternal
```
