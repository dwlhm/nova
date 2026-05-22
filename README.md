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
tests/conformance/        fixture conformance resmi
docs/adr/                 ADR desain Nova
```

## Dokumentasi Proyek

- [docs/architecture.md](docs/architecture.md) menjelaskan layer, ownership package, dan aturan dependency.
- [docs/code-style.md](docs/code-style.md) menjelaskan gaya Go, `.nova`, diagnostic, test, dan generated output.
- [CONTRIBUTING.md](CONTRIBUTING.md) merangkum alur kontribusi dan command verifikasi lokal.

## Command MVP

```bash
go run ./cmd/nova init --name demo
go run ./cmd/nova check --target web
go run ./cmd/nova build --target web
go run ./cmd/nova dev --target web
go run ./cmd/nova test
go run ./cmd/nova inspect --target web
go run ./cmd/nova fmt
```

`check` menjalankan pipeline validasi tanpa menulis artifact. `inspect` mencetak ringkasan
build plan JSON. `test` menjalankan fixture conformance dari `tests/conformance` secara default.
`fmt` menormalkan indentasi source `.nova`.

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

CSS project web bisa didaftarkan sebagai stylesheet global atau app-scoped dari manifest:

```toml
[targets.web]
renderer = "@nova/web"
styles = ["src/Counter.css"]
scoped_styles = ["src/App.css"]
```

File CSS disalin ke artifact web dan di-link setelah runtime base CSS. `scoped_styles`
diprefix ke root app web (`#nova-root[data-nova-style-scope~="app"]`) dan dicatat di
`build/web/style-manifest.json`. Detail CSS module dan style per capability masih area
lanjutan; lihat
`docs/adr/adr_024_style_asset_injection.md`.

## Dev Web

```bash
cd examples/counter
go run ../../cmd/nova dev --target web
```

CLI menjalankan proses development long-running: source project dipantau, artifact web
dibangun ulang lewat pipeline resmi, `build/web` disajikan lewat HTTP lokal, dan browser
yang terhubung menerima full reload setelah rebuild sukses.

## Example Multi Page

Source utama:

```txt
examples/multipage/src/App.nova
```

Example ini memakai state `route: Route`, event `@route_changed`, dan node `<page path <- "...">`
untuk memilih page aktif tanpa dirty operation di template.

`page.path` mendukung exact path (`/settings`), dynamic segment (`/users/:id` atau
`/users/{id}`), prefix wildcard (`/docs/*`), dan fallback (`*`). Runtime web dan Android
menormalisasi query/hash/trailing slash untuk matching, memilih page sibling paling spesifik,
dan mengisi `route.params` ketika state route berbentuk record/object.

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

## Dev Android

```bash
cd examples/counter
go run ../../cmd/nova dev --target android
```

CLI menjalankan loop seperti workflow minimum Android Studio: perubahan source memicu
generate project Android, Gradle debug build, `adb install --user 0 -r`, lalu app di-launch ulang.
Ini sengaja memakai install/update sync sebagai fallback utama, bukan runtime HMR, agar
developer cukup menjalankan `nova dev` sekali selama sesi kerja.

Berbeda dari `nova build`, dev Android default berjalan online supaya Gradle dapat
mengunduh dependency yang belum ada di cache pada run pertama. Setelah cache lengkap,
mode offline bisa dipakai dengan `--offline`.

Dev Android default memasang app hanya ke Android user `0`. Ini mencegah duplicate install
di device multi-user/profile seperti Samsung Dual Messenger atau Work Profile. Untuk target
profile lain, gunakan `--android-user <id>`.

Jika perlu override ADB:

```bash
go run ../../cmd/nova dev --target android --adb /path/to/adb
```

## Test

```bash
go test ./...
go run ./cmd/nova test
```

Test mencakup parser, build resolution, artifact generator, bundler, CLI, dan runtime core.
Conformance fixture membandingkan diagnostic code, metadata artifact, dan metadata ViewIR.
