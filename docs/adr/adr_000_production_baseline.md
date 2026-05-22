# ADR-000: Production Baseline

## Status

Accepted

## Context

Nova telah melampaui fase eksperimen MVP. Production baseline menetapkan batas resmi antara:

```txt
compiler/tooling (Go)
serializable ABI contracts
native target runtime (browser JS, Android Java)
platform env adapters (@env/*)
```

## Decision

### Production targets

```txt
web     -> static web artifact + browser JavaScript runtime
android -> Gradle Java project + Android framework View runtime
```

`@nova/android` adalah renderer production default. Renderer memakai source Java di
`app/src/main/java` tanpa Kotlin plugin, Compose, atau Material3.

`@nova/android-compose` tetap tersedia sebagai jalur compatibility yang deprecated untuk
project lama; tidak menjadi default baru.

### Layer trimming

```txt
.nova source
  -> Go compiler pipeline (parse, validate, build, ViewIR, artifact)
  -> ABI JSON (scheduler IR, ViewIR, permissions, routes)
  -> native runtime (web JS / Android Java)
  -> @env/* adapter (storage, network, device, ...)
  -> OS/platform API
```

Rule:

```txt
1. Go scheduler/app packages adalah conformance reference, bukan production runtime.
2. Production artifact tidak membaca .nova; hanya ABI + adapter binding.
3. @env/* adalah satu-satunya lapisan external capability ke platform asli.
4. Renderer tidak menambah framework UI berat di luar primitive Nova.
```

### ADR governance

ADR dengan status `Accepted` atau `Implemented / Accepted` mengikat implementasi production.
ADR baru harus memakai terminologi production v1 (lihat `docs/adr/README.md`), bukan "MVP" sebagai
label target akhir.

## Consequences

- APK Android lebih kecil karena menghindari Kotlin stdlib dan Compose stack.
- Dokumentasi dan example memakai konfigurasi Java-native minimal.
- Conformance wajib mencakup scheduler trace untuk perubahan semantic (lihat ADR-020).
