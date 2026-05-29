# Nova Design Philosophy

Catatan arah desain Nova — di luar detail bahasa dan tooling. Isinya dasar observasi dari
pengalaman membangun dan merawat aplikasi web + mobile, plus trade-off yang kita pilih.

Spesifikasi penulisan `.nova`: [language-design.md](language-design.md).

---

## Mekanisme aplikasi interaktif (yang sebenarnya terjadi)

Di permukaan grafis, aplikasi interaktif pada dasarnya menjalankan satu loop:

```txt
State (snapshot keadaan)
  -> render -> Screen
  -> listen (tap, ketik, gesture, lifecycle host)
  -> event
  -> ubah state (+ efek eksternal bila perlu)
  -> render lagi
```

**State** adalah penanda akibat: apa yang sudah terjadi setelah interaksi (dan respons sistem).
**Event** adalah sinyal bahwa sesuatu terjadi — bukan lapisan arsitektur, melainkan pemicu
antara listen dan perubahan state.

Pemisahan `ui / feature / domain / data` di banyak codebase adalah **konvensi organisasi
programmer**, bukan cara mesin layar bekerja. Nova tidak mendefinisikan ulang frontend sebagai
tumpukan lapisan itu; Nova mendefinisikan **kontrak** untuk loop di atas dan **zona penulisan**
di source agar loop itu tetap dapat diaudit saat aplikasi membesar.

---

## Dari pengalaman, apa yang sering pecah

**1. Efek tidak punya alamat tetap.**  
Di codebase besar, side effect (fetch, analytics, write storage, navigasi) menyebar di handler,
hook, dan helper. Saat insiden production, waktu investigasi banyak habis untuk menemukan *baris
mana* yang menyentuh resource — bukan untuk memperbaiki logika bisnis. Code review pun sulit
menjawab “apakah file ini pure?” tanpa membaca seluruh call tree.

**2. UI jadi tempat logika menumpuk.**  
Komponen layar yang seharusnya hanya render sering membawa validasi, caching, dan orchestration
karena itu jalur tercepat saat deadline. Akibatnya: layar sulit di-test tanpa environment UI,
dan perubahan desain visual memicu regresi bisnis.

Dalam jangka panjang pola ini cenderung ke dua ujung: kode **imperative** berantai di satu file
layar (tanpa modularitas yang bisa di-review), atau sebaliknya **modularitas dangkal** — banyak
file kecil tanpa batas tanggung jawab yang jelas, sehingga alur sulit dilacak dan
maintainability tidak pernah sengaja dirancang, hanya terakumulasi.

**3. Web dan mobile drift.**  
Tim yang menjaga dua stack (misalnya SPA + app native) biasanya berbagi spesifikasi produk,
tapi implementasi diverge: edge case beda, permission beda, urutan navigasi beda. Sinkronisasi
manual lewat QA dan dokumen; biayanya naik linear dengan jumlah fitur, bukan dengan jumlah
engineer.

**4. Kesalahan struktural ketahuan terlambat; waktu startup habis untuk kerja yang bisa lebih awal.**  
Validasi yang hanya di runtime di device berarti bug layout, routing, atau kontrak data sering
sampai ke user atau store review. Build yang lebih berat di CI biasanya lebih murah daripada
hotfix mendadak.

Di sisi lain, profil waktu di perangkat sering didominasi **inisialisasi**: resolve struktur
layar, bangun graph navigasi, parse konfigurasi, atau menyiapkan data yang sebenarnya sudah
bisa ditetapkan saat build. User menunggu di cold start atau frame pertama; tim mengira masalah
performa di “optimasi runtime”, padahal sebagian besar adalah pekerjaan yang belum dipindah ke
fase compile.

Data kuantitatif bervariasi per tim; pola di atas konsisten di proyek menengah–besar dengan
dua surface UI atau lebih.

---

## Apa yang kita optimalkan

Nova mengoptimalkan **kejelasan batas** dan **sama artinya lintas target**, dengan biaya:

- Disiplin struktur di source (pemisahan state / tampilan / efek harus eksplisit).
- Investasi pipeline build dan kontrak artefak (sampai IR dan metadata Nova-specific).
- Kurva belajar vocabulary baru — sengaja kecil, tapi tetap ada.

Nova **tidak** mengoptimalkan: skrip sekali jalan, compute berat di device, atau fleksibilitas
tanpa batas seperti bahasa general-purpose.

Secara praktik global, banyak solusi modern gagal bukan karena arah ini salah, tetapi karena
constraint dipasang terlalu kaku. Karena itu Nova memilih batas yang **progresif**: cukup ketat
untuk mencegah chaos, cukup longgar agar delivery tetap cepat.

---

## Arsitektur Nova: tiga lapis

Nova dibagi menjadi tiga lapis dengan tanggung jawab berbeda. Jangan menyamakan ketiganya dengan
“lapisan UI” klasik — ini pemisahan **core kontrak**, **runtime semantic**, dan **hidupkan di
platform**.

```txt
┌─────────────────────────────────────────────────────────────┐
│ 1. Nova core (build)                                        │
│    bahasa .nova, validator, proyeksi ViewIR, kontrak artefak │
│    berhenti di keluaran Nova-specific (IR + metadata)       │
└───────────────────────────────┬─────────────────────────────┘
                                │ artifact
┌───────────────────────────────▼─────────────────────────────┐
│ 2. Nova runtime (per environment, tipis)                    │
│    muat kontrak; event → transisi pure → commit             │
│    jadwalkan effect; invalidate render dari snapshot        │
│    logic pure sederhana (guard, transform) — bounded        │
└───────────────────────────────┬─────────────────────────────┘
                                │ port (render, listen, IO)
┌───────────────────────────────▼─────────────────────────────┐
│ 3. Provider (per target & mode delivery)                    │
│    render surface, normalisasi input → event                │
│    strategi web: SPA, SSR, SSG, hybrid, …                   │
│    strategi mobile/desktop: shell native, bundling, …       │
│    implementasi effect port (network, storage, sensor, …)  │
└─────────────────────────────────────────────────────────────┘
```

| Lapis | Memutuskan | Tidak memutuskan |
| --- | --- | --- |
| **Core** | Makna state, event, transisi, ViewIR, effect intent, permission | SPA vs SSR; widget framework spesifik |
| **Nova runtime** | Urutan semantic, commit, orchestrasi loop | Cara menggambar pixel; routing host platform |
| **Provider** | Cara hidupkan graphical realm di environment | Mengubah makna transisi state tanpa kontrak |

**Core** tidak memilih model delivery web (SPA/SSR/SSG) — itu keputusan **provider**. **Core**
menetapkan kontrak yang harus dihormati provider mana pun.

Di device tetap ada **Nova runtime** untuk logic pure sederhana dan mesin state/event — bukan
parser source penuh, bukan framework UI. Provider memegang render, listen, dan IO platform;
runtime memegang kebenaran semantic aplikasi.

---

## Satu alur untuk keempat masalah

Keempat pola di atas punya akar yang sama: **batas tanggung jawab tidak jelas** dan
**validasi datang terlambat**. Solusi Nova bukan memaksa user menulis "tiga lapisan file",
melainkan menjaga satu alur kerja yang konsisten:

```txt
.nova source
  -> build (core): validasi + pra-komputasi + kontrak / IR
  -> bundle per target: Nova runtime + provider terikat
  -> execution: runtime apply kontrak; provider render + listen + ports
```

### Arah desain di source: Model · View · Effect (guardrail penulisan, bukan lapisan mesin)

Nova mendorong tiga **zona penulisan** — bukan tiga dunia runtime terpisah:

| Zona (nama konseptual) | Di source Nova | Isi | Bukan |
| --- | --- | --- | --- |
| **Model** | `<contract state>` | snapshot + aturan `@event → hasil` (pure) | “Domain layer” terpisah dari UI; ini mesin keadaan aplikasi |
| **View** | `<template>` | deskripsi tampilan + niat user (`@event`) | Tempat side effect atau orchestration berat |
| **Effect** | `<lifecycle>` | interaksi OS / jaringan / storage (deklaratif) | Logic yang mengubah state tanpa melalui transisi |

Mapping ke loop mekanik:

```txt
listen  -> @event dari template atau host
Model   -> handler state (akibat interaksi)
Effect  -> setelah commit, lewat port provider
View    -> render ulang dari snapshot state
```

Ini **bukan** berarti setiap capability harus selalu punya tiga blok lengkap. Capability boleh
minimal, tetapi ketika concern itu ada, batasnya harus jelas dan bisa diaudit.

**Menutup #1:** Effect tidak hidup di View atau Model. Side effect harus lewat jalur deklaratif
yang bisa dilacak oleh validator/build.

**Menutup #2:** View tidak menampung validasi bisnis atau orchestration. Aturan perubahan
keadaan tetap di transisi state (pure); View hanya membaca snapshot dan mengirim event.

Prinsip utama di level bahasa: **semantic constraint, structural freedom**. Bahasa mengunci
makna dan batas perilaku, bukan memaksa template struktur file yang sama untuk semua tim.

### Build (core): validasi + pra-komputasi → kontrak / IR

Build tidak hanya “compile”. Build:

1. **Menyambung** semua modul (graph dependency, izin, referensi lintas modul).
2. **Memvalidasi** struktur (route valid, view tree konsisten, tipe serializable, effect
   terdaftar vs permission, binding provider).
3. **Mengeluarkan kontrak** untuk versi aplikasi itu: state machine, ViewIR, route, metadata
   izin, tabel effect, **provider bindings** — bentuk siap pakai, bukan source mentah.

**Menutup #3:** Web dan mobile tidak memelihara dua implementasi **logika transisi** terpisah.
Yang dibagi sekali adalah kontrak semantic; tiap target memuat runtime + provider sendiri.

**Menutup #4:** Kesalahan struktural gagal di CI. Kerja yang dulu di startup — resolve layar,
bangun routing, validasi graph — selesai di build. Di device: **load kontrak + apply** oleh
Nova runtime; provider tidak mem-parse `.nova` dari nol.

Validasi dijalankan bertingkat:

- loop lokal cepat untuk feedback harian,
- validasi lebih dalam di CI untuk gate release.

### Runtime: Nova runtime + provider

Per environment:

```txt
Provider: input user/platform → Event normal
Nova runtime: Event → transisi pure → commit snapshot
Nova runtime: jadwalkan Effect → panggil port provider
Nova runtime: invalidate view dari snapshot
Provider: ViewIR + snapshot → render di surface native
```

Tidak ada parser aplikasi penuh di device untuk struktur yang sudah dipra-hitung di build.
Tidak ada dua jalur kode produk untuk **makna** fitur yang sama.

**Logic di Nova runtime** sengaja dibatasi: transisi, guard, transform pure ringan.
Compute berat, query data sensitif, dan strategi delivery tetap di provider atau backend.

"Sama artinya" berarti **semantic loop** (event intent, transisi, permission intent, error
class) — bukan API atau UX identik 1:1 di semua platform.

---

## Provider: bawaan, referensi, dan buatan user

Provider adalah plugin yang menghidupkan kontrak di graphical realm. Nova menyediakan provider
referensi; tim boleh mendaftarkan **provider sendiri** untuk capability tertentu.

```txt
Source memakai capability / fitur A
  -> build: resolve provider (user registry vs referensi bawaan)
  -> artifact mencatat binding + contract_version
  -> runtime: node IR fitur A → delegate ke provider terikat
  -> provider: render + listen + effect ports; emit completion @event
  -> Nova runtime: commit state seperti biasa
```

| Mode | Peran |
| --- | --- |
| **Reference IR + provider bawaan** | Bentuk ViewIR / effect shape canonical; default conformance |
| **Custom provider** | Implementasi user (mis. web + android) yang memenuhi kontrak fitur A |

Compiler mengarahkan ke provider terdaftar saat menemukan capability yang di-handle; jika tidak
ada binding custom, dipakai referensi Nova. Custom provider **mengimplement kontrak**, bukan
mengubah makna event/state secara diam-diam.

Yang wajib dijaga untuk ekosistem provider:

- **Capability contract** stabil (event yang boleh emit, field snapshot, effect ports, versi).
- **Permission** mengikuti manifest — provider tidak membuka resource di luar deklarasi.
- **Conformance** — trace event dapat dibandingkan dengan referensi untuk provider kritikal.
- **Isolation** — kegagalan provider tidak merusak state machine (batas modul/proses per platform).

Provider boleh memilih SPA, SSR, SSG, atau pola native; itu tidak mengubah kontrak transisi state
yang divalidasi di build.

---

### Peta masalah → mekanisme

| # | Masalah | Akar | Yang menutup |
| --- | --- | --- | --- |
| 1 | Efek tanpa alamat | Side effect di mana saja | Effect lewat jalur deklaratif + port provider |
| 2 | Logika menumpuk di UI | View jadi shortcut deadline | Transisi di state; View render + emit event |
| 3 | Web/mobile drift | Dua codebase logika | Kontrak + runtime semantic sekali; provider per target |
| 4 | Bug terlambat; init mahal | Validasi & struktur di runtime | Build → IR/kontrak; runtime apply + provider render |

Keempat baris ini kuat kalau guardrail di source, validasi build, dan conformance provider
berjalan bersama.

---

### Trade-off yang tetap ada

- Tim perlu disiplin menjaga zona penulisan — awalnya bisa terasa lebih lambat dari "logic di
  komponen".
- Tool build (watch, error jelas, diff artefak, provider resolution) wajib layak; tanpa itu
  guardrail terasa beban.
- UX yang memang beda per platform tetap butuh keputusan produk eksplisit; kontrak tidak
  menghapus perbedaan UX, hanya mencegah perbedaan **logika transisi** tanpa sadar.
- Provider parity (bawaan vs custom) tidak gratis: conformance, observability, dan maintenance
  naik dengan jumlah binding.
- IR terlalu gemuk menjadikan core seperti framework UI; terlalu tipis memicu duplikasi di tiap
  provider.

---

## Prinsip operasional di dunia nyata

1. **Semantic parity, bukan API parity.**  
   Event intent, state transition, permission intent, dan error class harus konsisten; cara
   render dan delivery boleh berbeda per provider.

2. **Progressive constraints.**  
   Guardrail dimulai dari batas minimal yang berdampak besar, lalu diketatkan saat skala tim,
   frekuensi insiden, atau risiko rilis naik.

3. **Explicit capability degradation.**  
   Untuk tiap capability, status di setiap environment harus eksplisit: `supported`,
   `degraded`, atau `unsupported`.

4. **Escape hatch dengan governance.**  
   Kasus platform-unik boleh lewat provider atau jalur khusus, terisolasi, terdokumentasi,
   dan direview — bukan bypass state/event di template.

5. **Conformance over assumption.**  
   Kesetaraan lintas provider dibuktikan dengan test kontrak dan observability, bukan asumsi
   dari desain.

---

## Rincian konsep (referensi)

**State / Model (zona penulisan):** perubahan keadaan hanya lewat `@event` → transisi pure;
snapshot adalah penanda akibat interaksi; mudah diuji tanpa DOM.

**View:** deskripsi tampilan (ViewIR); membaca snapshot, mengirim event; tidak mengubah aturan
transisi secara tersembunyi.

**Effect:** resource eksternal lewat lifecycle deklaratif; dieksekusi provider lewat port setelah
commit.

**Kontrak / IR:** keluaran build per target; Nova runtime memuat dan apply; provider mengonsumsi
ViewIR dan port effect.

**Provider:** implementasi render, listen, dan IO platform; dapat bawaan atau custom selama
memenuhi capability contract.

---

## Keamanan mengikuti deklarasi

Default tolak; izin mengikuti apa yang aplikasi deklarasikan. Data antar zona tetap serializable.
Provider terikat permission manifest yang sama dengan effect bawaan. Ini praktik umum di
mobile; kita bawa ke seluruh surface agar permission review satu pola.

---

## Cakupan sengaja sempit

Hanya aplikasi interaktif multi-surface. Vocabulary kecil, pola dalam (keadaan, aksi, layar,
efek). Fitur bahasa baru hanya jika pola itu terbukti tidak cukup di proyek nyata — bukan
karena satu kasus edge menarik. Provider menangani perluasan platform; core tidak menjadi
bahasa general-purpose.

---

## Kapan pendekatan ini kurang cocok

- Prototipe sekali pakai, prioritas kecepatan tulis di atas konsistensi lintas rilis.
- Tim satu surface saja, tanpa rencana multi-target — overhead kontrak + provider mungkin tidak
  terbayar.
- Aplikasi yang dominan grafis/real-time dengan sedikit state deklaratif — mesin keadaan bukan
  pusat masalah.
- Organisasi tanpa CI/build yang andal — validasi build-first akan friksi.
- Tim yang hanya butuh satu framework UI tanpa kebutuhan kontrak lintas target — Nova menambah
  lapisan (core + runtime + provider) yang mungkin tidak perlu.

Transparansi ini penting agar Nova dipakai di konteks yang ROI-nya masuk akal.

---

## Bagaimana mengukur apakah arah ini berhasil

Indikator operasional, bukan slogan:

| Indikator | Yang diharapkan |
| --- | --- |
| Waktu review modul baru | Reviewer menunjuk zona state / view / effect tanpa diagram |
| Insiden “efek tidak terduga” | Turun setelah adopt batas efek; tetap perlu observability runtime |
| Drift web vs mobile pada fitur sama | Turun pada transisi state; perbedaan UX eksplisit dan disetujui produk |
| Bug struktural di production | Lebih banyak tertangkap di CI daripada di store/user |
| Cold start / frame pertama | P95 tidak naik linear dengan jumlah layar; kerja init shift ke artefak build |
| Ukuran Nova runtime | Stabil atau terkontrol; provider boleh lebih besar tanpa membesarkan semantic core |
| Custom provider | Conformance trace lulus atau deviasi terdokumentasi |
| Architecture tax (lead time implementasi) | Tidak naik signifikan setelah guardrail baru diterapkan |
| False-positive validasi lokal | Rendah; dev loop tetap cepat dan bisa diprediksi |

Tanpa metrik ini, prinsip arsitektur mudah jadi dokumen yang tidak dipakai.

---

## Posisi dokumen

Ini preferensi desain dari pengalaman — bukan janji produk. Spesifikasi teknis di repo (ADR,
[language-design.md](language-design.md), [architecture.md](architecture.md)) adalah cara
menerjemahkannya; jika terjemahan melenceng, perbaiki spesifikasi agar tetap selaras dengan
trade-off di atas.
