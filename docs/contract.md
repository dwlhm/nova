# Application contract (runtime bundle)

Dokumen ini mendefinisikan **satu kontrak** yang dimuat runtime produksi. Bukan spesifikasi
grammar `.nova` (lihat [language-design.md](language-design.md)) dan bukan dump compiler
internal (ViewIR mentah, capability manifest).

## Versi

| Field | Nilai |
| --- | --- |
| `v` | `1` — naikkan hanya saat breaking change pada bentuk JSON |

Versi toolchain (language, scheduler, runtime) dicatat di `build.manifest.json` (dev/CI), bukan
di bundle runtime.

## File keluaran build

| File | Dimuat runtime? | Isi |
| --- | --- | --- |
| `app.bundle.js` | Ya (web) | `window.__NOVA_APP__` = kontrak `App` |
| `app.contract.json` | Opsional | Salinan JSON sama (debug, conformance) |
| `build.manifest.json` | Tidak | Modul, template index, versi toolchain |
| `style-manifest.json` | Ya (web CSS) | Aset gaya |

Android tidak memuat JSON di APK untuk UI; kontrak diproyeksikan ke Java saat build. Folder
`nova-ir/` menyimpan `app.contract.json` + `build.manifest.json` untuk audit.

## Bentuk `App` (v1)

```json
{
  "v": 1,
  "target": "web",
  "entry": "src/App.nova",
  "permissions": ["storage.read"],
  "model": {
    "states": [
      {
        "owner": "Counter",
        "name": "count",
        "initial": "0",
        "transitions": [
          { "event": "@increment", "params": [], "expression": "count + 1" }
        ]
      }
    ]
  },
  "view": {
    "nodes": [
      {
        "kind": "button",
        "props": { "class": "\"primary\"" },
        "events": { "on_press": { "name": "@increment", "args": [] } },
        "children": []
      }
    ],
    "bindings": [
      { "at": [0], "prop": "class", "states": ["count"] }
    ]
  },
  "effects": [
    { "id": "@env/storage#set", "permissions": ["storage.write"] }
  ]
}
```

### `model`

State machine siap dieksekusi scheduler: `initial` dan `expression` adalah **ekspresi JavaScript**
yang dievaluasi runtime (`state.*`, `payload.*`).

### `view`

- `nodes`: tree UI platform-neutral; `props` / event `args` adalah string ekspresi (bukan token lexer).
- `bindings`: invalidasi incremental (`at` = path indeks node, `states` = nama state).
- Event arg khusus input: string `"$value"`.

Tidak ada: `capabilityManifests`, `modules`, `viewIR.Tokens`, `renderer` dictionary di bundle.

### `effects`

Daftar port eksternal yang dipakai aplikasi (`capability#operation`) + izin. Implementasi tetap
di provider / `@env/*`.

## Istilah (jangan dicampur)

| Istilah | Arti |
| --- | --- |
| **App contract** | JSON di atas (`App`, `v: 1`) |
| **ViewIR** | Representasi internal compiler; tidak diekspos ke bundle |
| **ABI / build manifest** | Metadata toolchain + graph; `build.manifest.json` |
| **Provider** | Kode yang render/listen/IO; di luar file contract |

## Perubahan kontrak

1. Update `internal/contract` dan generator di `internal/artifact`.
2. Bump `contract.Version` dan dokumentasi ini.
3. Update runtime `nova-scheduler-js` / Java jika field `model` berubah.
4. Tambah fixture conformance untuk `app.contract.json`.

Implementasi tipe Go: `internal/contract/app.go`.
