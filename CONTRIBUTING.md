# Contributing

Dokumen ini adalah pintu masuk praktis untuk menjaga kode Nova tetap konsisten.
Keputusan desain yang lebih panjang tetap berada di `docs/adr/`.

## Before Changing Code

1. Baca [docs/language-design.md](docs/language-design.md) jika menyentuh source `.nova` atau semantic bahasa.
2. Baca [docs/architecture.md](docs/architecture.md) untuk batas package dan ownership.
3. Baca [docs/code-style.md](docs/code-style.md) untuk gaya Go, source `.nova`, diagnostic, dan test.
4. Jika perubahan mengubah kontrak bahasa, ABI, runtime target, permission, atau layout project, tambahkan atau update ADR (ikuti [docs/adr/TEMPLATE.md](docs/adr/TEMPLATE.md)).

## Local Checks

```bash
gofmt -w cmd internal
go run ./cmd/nova fmt --check
go test ./...
go run ./cmd/nova test
```

Scheduler library checks (when changing `runtime/nova-scheduler-*`):

```bash
cd runtime/nova-scheduler-js && npm test
cd runtime/nova-scheduler-java && gradle test
```

Untuk perubahan yang menyentuh generated artifact atau target runtime, jalankan juga build example
yang relevan:

```bash
cd examples/counter
go run ../../cmd/nova build --target web
go run ../../cmd/nova build --target android --bundle=false
```

## Pull Request Shape

- Jelaskan perubahan behavior, bukan hanya file yang berubah.
- Sebutkan target yang terkena dampak: language core, semantic validation, web, android,
  CLI, conformance, atau docs.
- Sertakan test atau alasan singkat kenapa test baru tidak diperlukan.
- Jangan commit output `build/`, cache lokal, APK/AAB, atau metadata editor.
