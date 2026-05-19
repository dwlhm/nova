# Nova

Nova adalah eksperimen bahasa dan runtime lintas target untuk membangun aplikasi dari satu source graph. Saat ini target MVP yang tersedia adalah `web` dan `android`, dengan pipeline build yang menurunkan source `.nova` menjadi artifact target, bundle manifest, dan APK debug untuk Android.

## Status Singkat

- Source Nova diparse, divalidasi, dan diturunkan menjadi IR.
- Target `web` menghasilkan aplikasi static browser.
- Target `android` menghasilkan project Gradle Compose dan menjalankan `assembleDebug` lewat CLI.
- Example tersedia di `examples/counter` dan `examples/multipage`.

## Struktur Penting

```txt
cmd/nova/                 CLI entrypoint
internal/artifact/        generator artifact web dan android
internal/bundler/         proses bundling target dan integrasi Gradle
internal/build/           target resolution
internal/parser/          parser Nova
internal/validator/       semantic validation
examples/counter/         example app counter
examples/multipage/       example route state dan page projection
docs/adr/                 ADR desain Nova
```

## Example Counter

Source utama:

```txt
examples/counter/src/App.nova
```

Counter mendefinisikan state `count` dan event:

- `@increment`
- `@decrement`
- `@reset`

Target `web` dan `android` memakai template terpisah di source yang sama.

## Build Web

```bash
cd examples/counter
go run ../../cmd/nova build --target web
```

Output:

```txt
build/web/index.html
build/web/app.bundle.js
build/web/assets/nova-runtime.js
build/web/bundle-manifest.json
```

`index.html` bisa dibuka langsung di browser.

Untuk MVP, CSS project web didaftarkan sebagai stylesheet global dari manifest:

```toml
[targets.web]
renderer = "@nova/web"
styles = ["src/Counter.css"]
```

File CSS disalin ke artifact web dan di-link setelah runtime base CSS. Detail scoping,
CSS module, dan style per capability sengaja belum diputuskan di MVP; lihat
`docs/adr/adr_024_style_asset_injection.md`.

## Example Multi Page

Source utama:

```txt
examples/multipage/src/App.nova
```

Example ini memakai state `route: Route`, event `@route_changed`, dan node `<page path <- "...">`
untuk memilih page aktif tanpa dirty operation di template.

```bash
cd examples/multipage
go run ../../cmd/nova build --target web
```

Target Android memakai source yang sama:

```bash
cd examples/multipage
go run ../../cmd/nova build --target android
```

Jika hanya ingin memeriksa generated Android project tanpa menjalankan Gradle:

```bash
go run ../../cmd/nova build --target android --bundle=false
```

`[targets.android]` wajib membawa konfigurasi build dari sisi project, termasuk
`application_id`, `namespace`, SDK version, plugin/dependency version, `theme`,
`theme_parent`, `java_version`, dan `label`.
Lihat `examples/*/nova.toml` untuk contoh lengkap.

## Build Android

```bash
cd examples/counter
go run ../../cmd/nova build --target android
```

CLI akan:

1. Generate project Android di `build/android`.
2. Mencari Gradle dari `PATH`, `NOVA_GRADLE`, wrapper project, atau cache wrapper lokal.
3. Memakai JBR Android Studio bila `JAVA_HOME` saat ini tidak usable.
4. Menjalankan Gradle task default `assembleDebug`.
5. Menulis bundle manifest.

Output utama:

```txt
build/android/app/build/outputs/apk/debug/app-debug.apk
build/android/bundle-manifest.json
```

Jika perlu override Gradle:

```bash
go run ../../cmd/nova build --target android --gradle /path/to/gradle
```

Jika hanya ingin generate artifact tanpa bundling:

```bash
go run ../../cmd/nova build --target android --bundle=false
```

## Test

```bash
go test ./...
```

Test mencakup parser, build resolution, artifact generator, bundler, CLI, dan runtime core.
