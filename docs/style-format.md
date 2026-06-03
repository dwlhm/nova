# Nova portable style (`.nova-style`)

Panduan penulisan untuk skin lintas **web** dan **Android**. Spesifikasi mengikat:
[adr/adr_012_style_format.md](adr/adr_012_style_format.md).

Style didaftarkan dengan `<import style>` / `<import stylesheet>` di file `.nova`, bukan dengan
`styles` / `scoped_styles` di `nova.toml` (kunci legacy → `NVA-PROJECT-005`).

## Mulai cepat

**Entry atau modul** (setelah migrasi):

```nova
<import style from "./theme.nova-style" /|
<import style from "./Counter.nova-style" /|
<import stylesheet from "./web-effects.css" /|

<template target <- web>
  <surface class <- "surface-card">
    <text class <- "counter-value" value <- "Count: " + count /|
    <button class <- "primary-btn" on_press -> @increment>
      <text value <- "+" /|
    /|
  /|
/|
```

**File** `Counter.nova-style` (satu file = satu scope):

```txt
scope: app

token color.bg.surface = #fbfcfe
token color.text.primary = #151923
token space.3 = 12
token radius.md = 8

class surface-card {
  padding: space.3
  background: color.bg.surface
  border: 1 #dfe5ef
  border-radius: radius.md
  min-height: 48
}

class counter-value {
  color: color.text.primary
  font-size: 34sp
  font-weight: 800
  text-align: center
}

class primary-btn {
  background: #1d4ed8
  color: #ffffff
  padding: 10 14
  border-radius: 6
}

state primary-btn active {
  background: #1e40af
}

state primary-btn focus {
  border: 2 #93c5fd
}

state primary-btn disabled {
  background: #94a3b8
  color: #e2e8f0
}
```

**Web-only CSS** `web-effects.css` (hanya dipakai build web; Android mengabaikan import ini):

```css
.primary-btn:hover {
  background: #2563eb;
}
```

## Import vs TOML

| Cara | Status |
| --- | --- |
| `<import style from "./x.nova-style">` | Target — portable web + Android |
| `<import stylesheet from "./x.css">` | Target — web only |
| `styles` / `scoped_styles` di `nova.toml` | **Dihapus** (setelah migrasi) |

Style mengikuti **module graph**: jika `App.nova` mengimpor `Store.nova` dan `Store` mengimpor
style, sheet ikut ke build `App`.

## Satu file, satu scope

Header wajib di setiap `.nova-style`:

```txt
scope: global
```

atau

```txt
scope: app
```

| Scope | Arti |
| --- | --- |
| `global` | Theme/token; CSS web tanpa prefix app root |
| `app` | Skin layar; web prefix `#nova-root[data-nova-style-scope~="app"]` |

Jangan campur scope dalam satu file. Import **tidak** membawa scope.

## Aturan penamaan (web dulu)

| Situasi | Pakai istilah |
| --- | --- |
| Ada di web & Android, sama niat | Nama **CSS** (`color`, `padding`, `active`) |
| Ada di web, tidak di Android | Tetap nama web (`hover`); Android: abaikan + warning |
| Beda nama, sama niat | Nama **web** (`active`, bukan `pressed`) |

## Author glossary

### Import

| Istilah | Arti |
| --- | --- |
| `<import style …>` | Sertakan `.nova-style` portable |
| `<import stylesheet …>` | Sertakan `.css` untuk web saja |

### Scope & struktur

| Istilah | Arti |
| --- | --- |
| `scope: global` | Satu scope per file — global |
| `scope: app` | Satu scope per file — app-scoped di web |
| `class foo` | Setara `.foo` |
| `state foo active` | Setara `.foo:active` |
| `token a.b = …` | Konstanta desain; resolve saat build |

### State interaksi

| Tulis di file | Web | Android |
| --- | --- | --- |
| `hover` | `:hover` | Diabaikan di phone (warning) |
| `focus` | `:focus` | Fokus |
| `active` | `:active` | Sentuh / tekan |
| `disabled` | `:disabled` | Nonaktif |
| `checked` | `:checked` | Toggle (jika didukung) |

Untuk sentuh, pakai **`state … active`**, bukan `hover`.

### Properti portable (v1)

`color`, `font-size`, `font-weight`, `text-align`, `text-transform`, `line-height`, `padding`,
`min-height`, `background` / `background-color`, `border` (+ color/width), `border-radius`,
`align-content` (terbatas di Android).

### Web-only CSS

Gunakan file `.css` + `<import stylesheet>`:

- `box-shadow`, `display`, `flex`, `grid`, `@media`, `::before`, `--*`
- Selector kompleks dan pseudo di CSS mentah

Tidak perlu flag `target <- web` pada import; Android otomatis skip `.css`.

## Checklist PR

- [ ] Setiap `.nova-style` punya tepat satu `scope:`
- [ ] Portable skin di `.nova-style`, bukan `.css`, bila target Android
- [ ] Style di-import dari modul yang dipakai (transitive OK)
- [ ] `class` di template = string literal untuk skin shared
- [ ] Sentuh pakai `state … active`
- [ ] Enhancement mouse/layout di `.css` + `import stylesheet`
- [ ] Tidak duplikat class dalam scope yang sama setelah merge graph

## Lihat juga

- [README.md](../README.md) — status migrasi TOML
- [contract.md](contract.md) — `style-manifest.json` (web)
- [adr/adr_005_view.md](adr/adr_005_view.md) — style di provider
