# ADR-025: Android APK Size And Renderer Strategy

## Status

Accepted

## Context

Fase eksperimen awal pernah memakai framework UI AndroidX berat; production baseline (ADR-000,
ADR-017, ADR-025) beralih ke Java native View untuk mengecilkan APK dan memangkas lapisan UI.

Pembengkakan APK utama berasal dari dependency UI AndroidX, resource, dan DEX eksternal — bukan dari
jumlah file generated Nova.

Nova membutuhkan strategi jangka panjang agar aplikasi sederhana yang hanya memakai primitive dasar
tidak perlu membawa dependency UI besar.

## Decision

Production Android memakai renderer native View milik Nova: `@nova/android`.

Target arah:

```txt
ViewIR primitive -> Nova Android native View renderer -> Android View/ViewGroup/custom View ringan
```

Renderer native View harus dapat berjalan tanpa:

```txt
dependency UI berat yang tidak dipakai oleh primitive app
```

Untuk mode APK kecil, renderer native View menghasilkan source Java di `app/src/main/java`.
Renderer tidak bergantung pada framework UI eksternal supaya baseline ukuran APK benar-benar
mengukur renderer Nova dan Android framework platform.

Implementasi production (lihat `internal/artifact/android.go`):

```txt
1. `@nova/android` menghasilkan Java di app/src/main/java.
2. Primitive ViewIR diturunkan ke Android View/ViewGroup + invalidation granular.
3. Dependency UI eksternal tidak masuk dependency graph Android production.
4. Conformance + scheduler trace menjaga parity semantic dengan target web.
```

## Non Goals

```txt
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
Beberapa primitive kompleks mungkin tetap lebih murah memakai dependency eksternal
```
