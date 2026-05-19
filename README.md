# Nova

Nova adalah eksperimen bahasa dan runtime lintas target untuk membangun aplikasi dari satu source graph. Saat ini target MVP yang tersedia adalah `web` dan `android`, dengan pipeline build yang menurunkan source `.nova` menjadi artifact target, bundle manifest, dan APK debug untuk Android.

## Status Singkat

- Source Nova diparse, divalidasi, dan diturunkan menjadi IR.
- Target `web` menghasilkan aplikasi static browser.
- Target `android` menghasilkan project Gradle dan menjalankan `assembleDebug` lewat CLI.
- Example pertama tersedia di `examples/counter`.

## Struktur Penting

```txt
cmd/nova/                 CLI entrypoint
internal/artifact/        generator artifact web dan android
internal/bundler/         proses bundling target dan integrasi Gradle
internal/build/           target resolution
internal/parser/          parser Nova
internal/validator/       semantic validation
examples/counter/         example app counter
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
