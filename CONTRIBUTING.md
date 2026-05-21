# Contributing

Dokumen ini adalah pintu masuk praktis untuk menjaga kode Nova tetap konsisten.
Keputusan desain yang lebih panjang tetap berada di `docs/adr/`.

## Before Changing Code

1. Baca [docs/architecture.md](docs/architecture.md) untuk batas package dan ownership.
2. Baca [docs/code-style.md](docs/code-style.md) untuk gaya Go, source `.nova`, diagnostic, dan test.
3. Jika perubahan mengubah kontrak bahasa, ABI, runtime target, permission, atau layout project, tambahkan atau update ADR.

## Local Checks

```bash
gofmt -w cmd internal
go run ./cmd/nova fmt --check
go test ./...
go run ./cmd/nova test
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
