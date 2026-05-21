# Code Style

Panduan ini mengikat gaya harian repo Nova. ADR tetap menjadi sumber keputusan desain,
sedangkan dokumen ini menjawab "bagaimana menulis perubahan berikutnya".

## Formatting

- Go selalu memakai `gofmt`.
- File `.nova`, CSS, TOML, JSON, dan Markdown memakai indentasi 2 spasi.
- File Go memakai tab sesuai `gofmt`.
- Semua file teks memakai LF, UTF-8, final newline, dan tanpa trailing whitespace.
- Source `.nova` harus lolos `go run ./cmd/nova fmt --check`.

## Go Package Style

- Package name singkat, lowercase, dan sesuai direktori.
- API exported hanya ketika dipakai lintas package atau menjadi kontrak test/conformance.
- Struct dipakai sebagai data contract eksplisit; jangan menyembunyikan state penting di closure
  jika data itu perlu diuji atau diserialisasi.
- Fungsi pure lebih disukai di core compiler/runtime. Terima input sebagai value dan kembalikan
  value/diagnostic baru.
- Sortir output dari map, filesystem walk, dependency graph, dan diagnostic sebelum dikembalikan
  jika hasilnya terlihat oleh test, CLI, artifact, atau conformance.
- Hindari panic untuk input user. Pakai diagnostic atau error dengan kode stabil.
- Komentar hanya untuk keputusan yang tidak jelas dari kode. Jangan menulis komentar yang hanya
  mengulang nama fungsi atau assignment.

## Host IO Boundaries

Kode yang menyentuh filesystem, process, network, atau device host harus tetap di edge:

- `cmd/nova`
- `internal/cli`
- `internal/bundler`
- `internal/conformance`

Package lain menerima data, port, atau snapshot yang sudah dibaca oleh edge package. Ini menjaga
lexer, parser, validator, target resolution, runtime semantic, dan artifact generation tetap mudah
ditest.

## Diagnostics

- Diagnostic build mengikuti ADR-011.
- Kode diagnostic harus stabil dan memakai prefix domain yang tepat, misalnya `NVA-PARSE-*`,
  `NVA-TARGET-*`, `NVA-SEC-*`, atau `NVA-RENDER-*`.
- Message harus menyebut expected vs actual bila relevan.
- Diagnostic yang keluar dari map atau graph harus diurutkan deterministik.
- Jika diagnostic baru masuk conformance surface, update fixture expected.

## Nova Source Style

- Nama file capability/component memakai `PascalCase.nova`.
- State dan fungsi memakai `camelCase`.
- Contract type dan capability memakai `PascalCase`.
- Scheduler event memakai prefix `@` dengan `snake_case` atau pola `@verb_noun`.
- Satu blank line antar top-level block.
- Template dan block state memakai indentasi 2 spasi.
- Source project contoh harus bisa diformat oleh `nova fmt` tanpa perubahan tambahan.

## Tests

- Test berada dekat package yang diuji.
- Tambahkan unit test untuk parser, validator, build resolution, artifact, scheduler, atau package
  yang behavior-nya berubah.
- Tambahkan conformance fixture ketika perubahan menyentuh semantic lintas target, artifact metadata,
  ViewIR, permission, atau diagnostic yang menjadi kontrak.
- Hindari test yang bergantung pada wall-clock, random, network nyata, atau tool host yang tidak
  diinjeksi.
- Untuk command/process, pakai interface atau runner fake seperti pola `bundler.Runner`.

## Generated Output

- Output `build/` bukan source of truth.
- Jangan mengedit generated web/android artifact secara manual.
- Perubahan generator harus dilakukan di `internal/artifact`, `internal/bundler`, atau target
  contract terkait, lalu diverifikasi dengan build example.
