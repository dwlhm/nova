# ADR-024: Style Asset Injection

## Status

Accepted

## Scope

ADR ini mengikat style asset injection level production v1. Implementasi memakai stylesheet global
yang didaftarkan dari manifest target web. CSS module, style import per capability, scoping otomatis,
ordering antar capability, conflict handling, dan typed style token sengaja belum diputuskan di sini.
Area tersebut wajib dibahas dalam ADR lanjutan sebelum diimplementasikan.

## Context

Web target membutuhkan CSS, sementara Android target membutuhkan theme dan renderer styling.
Sebelumnya runtime CSS membawa class milik example seperti `counter-*` dan `multipage-*`.
Itu membuat artifact builder terlihat benar untuk example tertentu, tetapi melanggar batas renderer:
class milik user source tidak boleh menjadi aturan khusus di runtime umum.

## Decision

Style dibagi menjadi dua lapisan untuk production v1:

```txt
runtime base style   -> generic primitive defaults milik target runtime
global style assets  -> stylesheet global dari project manifest target
```

Runtime base style hanya boleh menargetkan primitive renderer-neutral atau marker generic target,
misalnya `data-nova-kind="row"`, `data-nova-kind="surface"`, atau Android theme yang dikonfigurasi
user. Runtime base style tidak boleh menargetkan class name yang berasal dari example atau aplikasi.

Untuk production v1, user style injection memakai stylesheet global per target:

```toml
[targets.web]
renderer = "@nova/web"
styles = ["src/theme.css", "src/counter.css"]

[targets.android]
renderer = "@nova/android"
styles = ["src/theme.css", "src/counter.css"]
```

Stylesheet web disalin ke artifact web dan di-link setelah runtime base CSS. Target Android memakai
subset class CSS statis dari stylesheet target untuk menghasilkan native View styling dasar
(`color`, `font-size`, `font-weight`, `text-transform`, `text-align`, `line-height`, `padding`,
`min-height`, `background`, `border`, dan `border-radius`). Class dari template tetap pass-through:

```nova
<surface class <- "dashboard-shell">
```

Target adapter boleh meneruskan class/style hook ke platform, tetapi tidak boleh menafsirkan nama
class tertentu tanpa kontrak renderer package atau asset user yang eksplisit.

Capability-local import, CSS module semantics, scoping, conflict handling, dan typed style token
tidak diputuskan di ADR ini. Area tersebut harus dibuat dalam ADR lanjutan sebelum diimplementasikan.

## Rules

1. Runtime target tidak boleh hardcode class milik example.
2. `class` adalah style hook user-side, bukan semantic branch compiler.
3. Target web boleh menyertakan base CSS generic untuk primitive defaults.
4. User CSS production v1 masuk melalui `styles` target, bukan disisipkan ke runtime umum.
5. Android theme, dependency styling, label, dan stylesheet harus berasal dari konfigurasi target project.
6. Runtime tetap harus generic dan tidak boleh mengompensasi dengan style khusus example.
7. Capability-local style import dan CSS module harus menunggu ADR lanjutan.

## Consequences

Keuntungan:

```txt
runtime tidak drift mengikuti example
template tetap portable lintas target
style user dapat diaudit sebagai asset project
Production v1 tidak menambah grammar import baru
```

Trade-off:

```txt
stylesheet global belum menyelesaikan scoping antar capability
target perlu metadata asset lebih kaya untuk CSS module lanjutan
```
