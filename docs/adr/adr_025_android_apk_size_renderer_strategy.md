# ADR-025: Android APK Size And Renderer Strategy

## Status

Proposed

## Context

ADR-017 memilih Jetpack Compose dan Material3 sebagai renderer MVP Android karena mempercepat
implementasi awal dan cocok dengan ViewIR deklaratif. Setelah APK debug dihasilkan, ukuran artifact
dapat membesar oleh dependency UI AndroidX, Compose runtime, Material3, resource, dan DEX eksternal.

Jumlah file Kotlin generated Nova bukan penyebab utama pembengkakan APK. Memecah output menjadi
`MainActivity.kt`, `NovaRuntime.kt`, `NovaRoutes.kt`, dan file generated lain tetap masuk source set
Kotlin yang sama dan tidak menambah overhead runtime berarti.

Nova membutuhkan strategi jangka panjang agar aplikasi sederhana yang hanya memakai primitive dasar
tidak perlu membawa dependency UI besar.

## Proposal

Tambahkan renderer Android native View milik Nova sebagai kandidat pengganti Compose renderer untuk
mode APK kecil.

Target arah:

```txt
ViewIR primitive -> Nova Android native View renderer -> Android View/ViewGroup/custom View ringan
```

Renderer native View harus dapat berjalan tanpa:

```txt
androidx.compose.*
androidx.activity:activity-compose
androidx.compose.material3:material3
dependency UI berat yang tidak dipakai oleh primitive app
```

Nova dapat tetap mempertahankan Compose renderer sebagai mode produktivitas atau compatibility selama
native View renderer belum mencapai parity.

## Tasks

```txt
1. Ukur baseline APK debug dan release untuk renderer Compose saat ini.
2. Tambahkan target/config renderer Android native View minimal.
3. Turunkan primitive dasar text, button, row, column, stack, page, dan surface ke View/ViewGroup.
4. Build primitive UI dasar Nova sendiri, termasuk styling minimal, state update, dan event bridge.
5. Hilangkan Material3, Compose UI, dan activity-compose saat renderer native View aktif.
6. Ukur ulang APK debug/release untuk native View renderer dan bandingkan dengan Compose renderer.
7. Evaluasi setiap dependency UI eksternal berdasarkan ukuran APK, effort implementasi, aksesibilitas,
   styling, maintainability, dan parity lintas target.
8. Tambahkan conformance atau smoke fixture Android untuk memastikan kedua renderer menjaga semantic
   ViewIR, event route, routing, dan state update yang sama.
```

## Decision Criteria

Renderer native View layak menjadi default Android bila:

```txt
APK release lebih kecil secara signifikan untuk app primitive dasar
event/state/routing semantics tetap setara dengan renderer Compose
aksesibilitas dasar tidak mundur
styling primitive tetap cukup konsisten dengan target web
biaya maintenance masih wajar untuk Nova
```

Dependency UI eksternal boleh tetap dipakai bila penghematan ukuran tidak sepadan dengan effort
implementasi atau risiko kualitas.

## Non Goals

```txt
Menghapus Compose renderer segera
Membuat clone Material3 lengkap di Nova
Mengorbankan semantic ViewIR atau event scheduler demi ukuran APK
Mengoptimalkan APK sebelum ada baseline debug/release yang terukur
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
Parity dengan Compose renderer harus dijaga lewat test dan conformance
Beberapa primitive kompleks mungkin tetap lebih murah memakai dependency eksternal
```
