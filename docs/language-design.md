# Nova Language Design

Panduan normatif untuk menulis `.nova`. Arah produk (di luar detail bahasa):
[design-philosophy.md](design-philosophy.md). Grammar lengkap:
`docs/adr/adr_001_language_specification.md`.

## Apa itu Nova

Nova adalah **bahasa framework multiplatform** untuk mendeklarasikan aplikasi: state di
`<contract state>`, UI di `<template>`, efek di `<lifecycle>`. Bukan bahasa general-purpose.

Pemisahan pure/effect **eksplisit di syntax**, supaya terbaca natural — state dan view seperti
kode aplikasi biasa; efek tidak “bocor” ke template atau func. Di belakang, runtime memakai
scheduler untuk menjaga urutan lintas target (detail di ADR-002); itu kontrak implementasi, bukan
hal yang harus kamu rancang manual tiap fitur.

## Enam prinsip (ingat ini dulu)

| # | Prinsip | Artinya singkat |
| --- | --- | --- |
| 1 | **Explicit zones** | State/UI di construct pure; efek hanya di lifecycle/external — batas terlihat, alur tetap natural. |
| 2 | **Pure by default** | `contract`, `func`, `template` tidak boleh side effect. |
| 3 | **Events, not hidden actions** | Perilaku aplikasi lewat `@event` + handler state; bukan API imperative tersebar. |
| 4 | **File = capability** | Satu `.nova` = satu unit kompilasi, dependency, dan audit keamanan. |
| 5 | **Import is inert** | Import hanya menambah edge di graph; tidak menjalankan kode. |
| 6 | **Serializable boundaries** | Data antar core, ABI, adapter, dan persistence harus serializable — tanpa handle platform di core. |

Prinsip 1–6 mengikat ADR-001–011. Urutan runtime lintas platform dijamin scheduler
(ADR-002) — jangan menambah construct baru hanya untuk “memanggil scheduler”; gunakan event,
state, dan lifecycle yang sudah ada.

## Model mental (yang terasa saat menulis)

```txt
Interaksi user / platform
  -> @event (dari template atau host)
  -> handler di <contract state> (ubah state, pure)
  -> bila perlu efek: <lifecycle> (storage, navigasi, external, …)
  -> template ter-render ulang dari snapshot state
```

Template **hanya** membaca state/props dan emit `@event` — tidak memanggil API platform.

### Di bawah hood (untuk runtime / conformance)

Setelah `@event`, runtime menjalankan transisi, **commit** atomik, lifecycle, lalu invalidasi
render sesuai ADR-002. Penulis `.nova` tidak perlu menyebut langkah commit/queue; cukup model di
atas.

## Tujuh construct — tidak ada yang kedelapan

Nova sengaja memakai **hanya** construct berikut. Semua pola lain dipetakan ke sini, bukan
ditambah sebagai syntax baru.

| Construct | Peran | Zona |
| --- | --- | --- |
| `<import>` / `<import external>` | Dependency dan kontrak operasi luar | Deklarasi (inert / contract) |
| `<contract type>` | Bentuk data | Pure |
| `<contract state>` | State + handler event (`@...`) | Pure (transition) |
| `<contract capability>` | Props + event yang di-emit UI | Pure (deklarasi) |
| `<func>` | Transform data | Pure |
| `<template>` | Deklarasi view + routing event UI | Pure |
| `<lifecycle>` | Efek samping userland | Effect |

### Yang sengaja tidak ada

| Bukan di Nova | Sebagai gantinya |
| --- | --- |
| `<state>` global | `<contract state>` per capability |
| `<action>` / `<effect>` | `@event` + transition; dirty di `<lifecycle>` |
| `<dirty>` | Lifecycle **adalah** dirty zone |
| `use` / `bind` hook | `import` symbol, state, atau event |
| Router / widget native di bahasa | State `route` + `<page>` di template (ADR-005); provider platform |

Menolak construct tambahan menjaga compiler, formatter, dan conformance tetap kecil.

## Tiga zona

```txt
┌─────────────────────────────────────────┐
│ Pure core                               │
│  contract type | state | capability     │
│  func | template                        │
└─────────────────┬───────────────────────┘
                  │ snapshot + @events
┌─────────────────▼───────────────────────┐
│ Lifecycle (satu dirty zone userland)    │
└─────────────────┬───────────────────────┘
                  │ ports
┌─────────────────▼───────────────────────┐
│ Adapters (scheduler, @env/*, renderer)  │
└─────────────────────────────────────────┘
```

## Aturan penulisan (checklist)

Saat menulis atau men-review `.nova`:

1. Satu concern utama per file capability.
2. Semua perubahan state lewat handler di `<contract state>`, bukan assignment di lifecycle.
3. Event publik memakai prefix `@`; tanpa payload → `void`.
4. Template: binding dan event saja — tidak ada pemanggilan external di template.
5. Storage, network, device, navigasi programmatic → lifecycle (atau framework package
   `@nova/*` yang sudah memodelkan event yang sama).
6. Platform API → `<import external>` / `@env/*`, bukan inline di func/template.
7. Cross-capability: import event/state yang diperlukan; jangan duplikasi state global.

## Konvensi nama (konsisten)

| Konsep | Konvensi | Contoh |
| --- | --- | --- |
| Scheduler event | `@snake_case` | `@increment`, `@route_changed` |
| State contract | `PascalCase` | `Counter`, `Router` |
| Type contract | `PascalCase` | `Route`, `Money` |
| Capability file | `PascalCase.nova` | `Counter.nova` |
| Route state (v1) | field `route` | ADR-005 |

## Kapan membaca ADR mana

| Pertanyaan | ADR |
| --- | --- |
| Grammar & construct | 001 |
| Runtime, lifecycle, route, persistence | 002 |
| Module, layout, package, provider binding | 003 |
| Tipe & serializable | 004 |
| ViewIR, routing, renderer | 005 |
| External & permission | 006 |
| Build & target | 007 |
| Diagnostics | 008 |
| Web & Android target | 009 |
| `@nova/*`, `@env/*` | 010 |
| CLI, test, conformance | 011 |

## Menambah fitur bahasa

Sebelum menambah syntax atau construct:

1. Buktikan tidak bisa diekspresikan dengan tujuh construct + package/adapter.
2. Update dokumen ini dan ADR-001; tambahkan conformance fixture.
3. Pertahankan pure/effect split; jangan buka dirty zone baru di userland tanpa ADR eksplisit.

Lihat juga [architecture.md](architecture.md) § Adding A Feature.
