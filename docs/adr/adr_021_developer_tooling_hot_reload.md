Berikut ADR yang fokus ke **developer tooling, dev server, dan hot reload** Nova.

---

# ADR-021: Developer Tooling & Hot Reload

## Status

Implemented / Accepted

## Context

Nova sebagai framework penuh membutuhkan tooling yang membuat siklus kerja developer jelas:

```txt
create project
format/check source
build target
run dev server
run Android preview
inspect diagnostics
run conformance tests
package release
```

Tooling harus menjaga semantic yang sama dengan compiler/build resmi. Dev mode tidak boleh
menjalankan shortcut yang melanggar purity, permission, atau event ordering.

---

# Decision

Nova menyediakan CLI resmi:

```txt
nova
```

Production commands:

```txt
nova init
nova check
nova build
nova dev
nova test
nova inspect
nova fmt
```

Command target-aware:

```txt
nova build --target web
nova build --target android
nova dev --target web
nova dev --target android
```

Tooling membaca `nova.toml` sebagai manifest project.

---

# CLI Responsibilities

## nova check

Menjalankan:

```txt
layout validation
parse
semantic validation
type check
purity check
permission audit
target resolution if --target provided
```

Tidak menghasilkan artifact production.

## nova build

Menghasilkan artifact target:

```txt
web      -> build/web
android  -> build/android or Gradle integration
```

Build wajib fail pada diagnostic severity error.

## nova dev

Menjalankan loop development:

```txt
watch files
incremental compile
serve or deploy target
stream diagnostics
hot reload when compatible
fallback full reload when necessary
```

Production dev mode berjalan sebagai proses long-running. Developer menjalankan `nova dev`
sekali, lalu CLI melakukan rebuild/deploy setiap kali file project berubah. Ini menjadi
kontrak ergonomi lintas target: user tidak perlu menjalankan command build/dev ulang
untuk setiap perubahan source.

## nova test

Menjalankan:

```txt
unit tests
fixture tests
scheduler conformance
target smoke tests
adapter contract tests
```

Subset dipilih melalui flag target.

## nova inspect

Menampilkan:

```txt
module graph
capability manifest
permission audit
target resolution plan
event route table
state cells
artifact metadata
```

---

# Diagnostic Protocol

Tooling memakai diagnostic stream yang sama dengan ADR-011.

Transport dev mode dapat berupa:

```txt
stdout JSON lines
dev server websocket
IDE language server protocol adapter
runtime devtools bridge
```

Rule:

```txt
1. Diagnostic code stabil di semua transport.
2. Source span mengacu ke file .nova.
3. Runtime error memakai source map bila tersedia.
4. Target adapter diagnostic harus menyertakan target id.
```

---

# Hot Reload

Hot reload memakai ABI compatibility check.

Patch types:

```txt
template_patch
pure_func_patch
style_token_patch
target_adapter_patch
state_schema_patch
permission_patch
runtime_semantic_patch
```

Rule:

```txt
1. template_patch boleh state-preserving jika event/state contract tidak berubah.
2. pure_func_patch boleh state-preserving jika signature tidak berubah.
3. state_schema_patch membutuhkan migration atau full reload.
4. permission_patch membutuhkan full rebuild and permission audit.
5. runtime_semantic_patch membutuhkan full reload.
6. external adapter patch tidak boleh melewati permission audit.
```

Hot reload flow:

```txt
file change
  -> incremental compile
  -> diagnostics
  -> ABI diff
  -> patch runtime or full reload
  -> preserve snapshot if compatible
```

---

# Web Dev Mode

`nova dev --target web` menyediakan:

```txt
local dev server
incremental bundle
browser diagnostic overlay
runtime trace panel
permission audit panel
event/state inspector
hot reload
source-mapped errors
```

Production implementation:

```txt
poll watched project files
generate web artifact through official build pipeline
serve build/web over local HTTP
inject development-only EventSource client at response time
send browser full-reload event after successful rebuild
```

Rule:

```txt
1. Dev overlay tidak menjadi bagian production artifact.
2. Event injection devtools harus melewati event contract validation.
3. State editing devtools hanya tersedia dalam dev mode dan harus tercatat di trace.
4. Dev server external adapters tetap dianggap dirty.
5. Production boleh memakai full browser reload sampai ABI patching tersedia.
```

---

# Android Dev Mode

`nova dev --target android` menyediakan:

```txt
Gradle integration
generated Java update (@nova/android) atau Kotlin update (@nova/android-compose deprecated)
device/emulator deploy
runtime diagnostic bridge
hot restart when ABI incompatible
state-preserving renderer patch when compatible
source-mapped errors
```

Production implementation memilih install/update sync, bukan runtime HMR:

```txt
poll watched project files
generate Android project through official build pipeline
run Gradle debug task
install or update APK with adb install --user <id> -r
launch explicit activity component with adb shell am start --user <id> when enabled
repeat on every project file change
```

Pendekatan ini meniru workflow "Run" Android Studio pada level minimum yang dibutuhkan:
APK debug langsung dipasang/diperbarui ke device atau emulator dari proses dev yang
tetap berjalan. Runtime HMR Android ditunda karena biaya bridge, ABI diff, state
migration, dan granular hot patch belum sepadan untuk production v1.

Karena dev mode mengutamakan first-run ergonomics, Android dev bundling default tidak
memakai offline mode. Developer tetap dapat mengaktifkan `--offline` ketika Gradle cache
sudah lengkap.

Android dev deploy default memakai user `0` supaya device multi-user/profile tidak menerima
install ganda. CLI boleh mengekspos override user id untuk Work Profile atau profile lain.

Implementation may use:

```txt
Gradle plugin
ADB bridge
Android Studio integration
local websocket diagnostic bridge
```

Rule:

```txt
1. Android dev runtime must not require production app to expose debug bridge.
2. Generated code changes must be deterministic.
3. Runtime event trace must use the same format as conformance trace.
4. Production Android dev mode must not require repeating `nova dev` after every source change.
5. Android HMR is optional; install/update sync is the required fallback strategy.
6. Android install/update sync must target exactly one Android user/profile by default.
```

---

# Formatting

`nova fmt` formats `.nova` source.

Rule:

```txt
1. Formatter must preserve semantic.
2. Formatter should not reorder top-level declarations unless explicitly configured.
3. Comments should remain near original construct.
4. Formatter should be idempotent.
```

Production formatter may be conservative and only normalize whitespace around known constructs.

---

# Language Server

Nova tooling should expose language server capability:

```txt
syntax diagnostics
semantic diagnostics
go to definition
find references
rename symbol
hover contract
target availability hints
permission usage hints
```

Language server consumes compiler semantic model, not a separate parser.

---

# Consequences

Keuntungan:

```txt
developer workflow target web dan android jelas
hot reload tetap aman terhadap ABI dan permission
diagnostic source map dipakai seragam oleh CLI, browser, Android, dan IDE
```

Trade-off:

```txt
tooling menjadi bagian framework contract
dev bridge perlu disiplin agar tidak bocor ke production
state-preserving hot reload butuh ABI diff yang matang
```
