# ADR-023: Page Projection Routing

## Status

Implemented / Accepted

## Context

ADR-018 menetapkan bahwa multi screen Nova memakai route state, conditional template
projection, dan capability composition. ADR-006 juga menyebut conditional view, tetapi syntax
conditional belum dikunci di grammar.

MVP web membutuhkan cara yang cukup eksplisit untuk membuat beberapa page tanpa menambah dirty
zone atau native router object ke bahasa.

## Decision

Nova memakai view primitive `page` sebagai projection renderer-neutral:

```nova
<page path <- "/settings">
  <text value <- "Settings" /|
/|
```

`page` bukan top-level language construct. Ia adalah node ViewIR biasa dengan semantic khusus:
renderer menampilkan children page ketika `path` cocok dengan route aktif.

Route aktif dibaca dari state serializable:

```txt
route: string
```

atau:

```txt
route: Route { path: string }
```

MVP memilih nama state `route` sebagai convention. Transition tetap event scheduler biasa:

```nova
<contract type Route>
  path: string;
/|

<contract state Router>
  route: Route <- { path <- "/" } {
    @route_changed(next: Route) -> next;
  };
/|
```

ViewIR metadata menambahkan `Pages` agar tooling, conformance, dan target adapter dapat membaca
page projection tanpa menjalankan source Nova.

Artifact metadata juga menambahkan route table target-neutral:

```txt
routes: [{ pattern, nodePath, params, score, fallback }]
```

Route table ini diturunkan dari `ViewIR.Metadata.Pages` dan dipakai tooling/runtime sebagai kontrak
deterministik, bukan sebagai native router object.

Target adapter awal:

```txt
web      -> DOM fragment projection
android  -> Compose projection dari route state hasil scheduler transition
```

## Rules

1. `page.path` harus pure binding dan dievaluasi sebagai string path.
2. `page` tidak menghasilkan DOM/native container sendiri; children diproyeksikan sebagai fragment.
3. Route state tetap source of truth; adapter tidak menulis state langsung.
4. Event dari tombol/link tetap event route scheduler, misalnya `@route_changed`.
5. Record route literal memakai data syntax `{ path <- "/settings"; }` dan harus serializable.
6. Android adapter tidak boleh mengubah route aktif langsung dari event handler; handler harus
   dispatch event route, transition model memperbarui state, lalu renderer memproyeksikan page aktif.
7. Browser History API dan Android back stack tetap adapter reconciliation dari ADR-018, bukan
   bagian wajib page projection MVP.
8. `page.path` boleh berupa exact path (`/settings`), dynamic segment (`/users/:id` atau
   `/users/{id}`), prefix wildcard (`/docs/*`), atau fallback (`*` dan `/*`).
9. Adapter memilih page paling spesifik di level sibling yang sama. Exact/static segment mengalahkan
   dynamic segment, dynamic mengalahkan prefix wildcard, dan fallback menjadi pilihan terakhir.
   Beberapa page dengan skor sama boleh diproyeksikan bersama.
10. Route matching menormalisasi leading slash, trailing slash, duplicate slash, segment `.` dan
    `..`, query string, fragment, serta percent-encoded path segment.
11. Ketika route state berbentuk record/object, adapter boleh menambahkan `params` hasil dynamic
    match, serta mempertahankan `query` dan `fragment`. Ketika route state berbentuk string, adapter
    mengembalikan string path agar shape state tetap stabil.

## Consequences

Keuntungan:

```txt
multi-page bisa dipakai sebelum conditional syntax final
tidak ada dirty operation di template
ViewIR tetap renderer-neutral
example web bisa memakai route state yang sama dengan navigation ADR
Android dapat memakai example yang sama tanpa template khusus
```

Trade-off:

```txt
nama state route masih convention
typed query params, modal route, dan native transition masih butuh contract tambahan
URL/back-stack reconciliation tetap target-specific meski memakai route model yang sama
```
