Berikut ADR yang fokus ke **platform capability model** Nova.

---

# ADR-007: Platform Capability Model

## Status

Implemented / Accepted

## Context

Nova menargetkan:

```txt
web
android
ios
desktop
future target
```

Setiap target memiliki API, permission, renderer, event loop, dan lifecycle native yang berbeda.
Nova core tidak boleh mengenal detail tersebut secara langsung.

ADR ini mendefinisikan cara Nova menyatakan kebutuhan platform tanpa membocorkan object native
ke language core.

---

# Decision

Nova memakai model **platform capability**.

Platform capability adalah contract target-neutral yang dipenuhi oleh target adapter.

Kategori platform capability:

```txt
host capability
renderer capability
external capability
environment capability
permission capability
```

Capability platform diakses melalui:

```txt
@env/*
@nova/*
target manifest
external import contract
renderer primitive package
```

Core Nova hanya melihat contract, bukan implementasi native.

---

# Capability Categories

## Host Capability

Host capability menghubungkan event platform ke scheduler.

Contoh:

```txt
DOM click
Android onClick
iOS UIControl action
desktop command event
```

Semua diterjemahkan menjadi:

```txt
EventEnvelope
```

Host capability tidak boleh menulis state langsung.

## Renderer Capability

Renderer capability menyediakan primitive view dan update platform.

Contoh:

```txt
button
text
input
list
image
canvas
navigation surface
```

Renderer menerima:

```txt
view IR
state invalidation
event route metadata
```

Renderer mengirim event kembali ke scheduler.

## External Capability

External capability adalah dirty operation boundary.

Contoh:

```txt
@env/storage
@env/network
@env/audio
@env/os
@env/location
```

External capability hanya boleh dipanggil dari lifecycle.

---

# Platform Capability Contract

Contract platform harus menyatakan:

```txt
name
operations atau primitive nodes
input types
output types
required permissions
supported targets
failure modes
adapter implementation reference
```

Contoh external:

```nova
<import external storage from "@env/storage">
  operation get {
    input {
      key: string;
    }

    output unknown;
  }

  operation set {
    input {
      key: string;
      value: unknown;
    }

    output void;
  }
/|
```

Build target wajib menemukan adapter yang memenuhi contract tersebut.

---

# Target Support

Platform capability dapat mendukung target berbeda.

```txt
@env/storage:
  web      -> localStorage / IndexedDB adapter
  android  -> SharedPreferences / DataStore adapter
  ios      -> UserDefaults / file adapter
  desktop  -> filesystem adapter
```

Rule:

```txt
1. Contract Nova tetap sama.
2. Implementasi target boleh berbeda.
3. Output harus tetap valid Nova data.
4. Error harus masuk scheduler error/completion event.
5. Permission harus divalidasi per target.
```

---

# Target Capability Manifest

Target adapter menyediakan manifest:

```txt
TargetCapabilityManifest {
  target: TargetId
  capabilities: PlatformCapability[]
  rendererPrimitives: PrimitiveContract[]
  permissions: PermissionContract[]
  fileExtensions: TargetSourcePattern[]
}
```

Build memakai manifest untuk menjawab:

```txt
apakah @env/storage tersedia untuk web
apakah <button> tersedia untuk android
apakah permission notification sudah dideklarasikan
file external mana yang dipilih untuk target ios
```

---

# Capability Resolution

Resolution order untuk platform capability:

```txt
1. explicit target implementation
2. target family implementation
3. portable implementation
4. package default implementation
5. fail build
```

Contoh:

```txt
storage.web.js
storage.mobile.java
storage.common.js
```

Jika build target `web`, resolver memilih `storage.web.js`.
Jika tidak ada implementasi cocok, build gagal dengan diagnostic target resolution.

---

# Permission Attachment

Platform capability dapat mensyaratkan permission.

```txt
@env/location.read -> permission location.read
@env/notify.send   -> permission notification.send
@env/files.write   -> permission filesystem.write
```

Rule:

```txt
1. Permission dideklarasikan di project manifest.
2. Build gagal jika capability membutuhkan permission yang tidak dideklarasikan.
3. Runtime boleh tetap meminta izin user sesuai platform.
4. Permission denial masuk error lifecycle atau completion failure event.
```

Security detail ada di ADR-013.

---

# Feature Detection

Nova tidak menggunakan feature detection langsung di template atau transition.

Invalid direction:

```txt
if platform.hasCamera then ...
```

Preferred model:

```txt
target resolution memilih capability valid
capability contract menyatakan unsupported behavior
adapter mengirim event failure jika runtime permission/feature tidak tersedia
```

Reason:

```txt
Core language tetap platform-neutral.
Target branching terjadi di build graph atau adapter boundary.
```

---

# Degradation

Optional platform feature harus dimodelkan sebagai contract.

Contoh:

```txt
operation canSendNotification -> boolean
operation sendNotification -> void
```

Namun hasil operation tetap dirty dan hanya dapat dipakai melalui lifecycle/completion event.

Template tidak boleh langsung melakukan branching berdasarkan API platform.

---

# Alternatives Considered

## Platform Globals

Nova menyediakan global seperti `web`, `android`, `ios`, atau `platform`.

Rejected because:

```txt
Platform branch akan bocor ke pure core.
Template dan transition menjadi target-dependent.
Conformance lintas target melemah.
```

## Native Object in State

State boleh menyimpan object platform seperti DOM node atau native handle.

Rejected because:

```txt
State harus serializable Nova data.
Scheduler core tidak boleh memegang object target.
```

## Runtime Best-Effort Capability

Jika capability tidak ada, runtime diam-diam no-op.

Rejected because:

```txt
Behavior target menjadi tidak jelas.
Build harus gagal untuk dependency yang tidak terpenuhi.
```

---

# Consequences

## Positive

```txt
1. Core language tetap platform-neutral.
2. Target adapter memiliki contract eksplisit.
3. Build dapat gagal cepat jika capability tidak tersedia.
4. Permission dapat diaudit statis.
5. Renderer dan external API punya model yang seragam.
```

## Negative

```txt
1. Setiap target butuh manifest capability.
2. Package author harus menjaga contract lintas target.
3. Optional feature perlu dimodelkan eksplisit.
4. Adapter compatibility harus diuji dengan conformance suite.
```

---

# Final Position

Nova platform capability model adalah:

```txt
contract-first
target-resolved
permission-aware
adapter-implemented
core-neutral
```

Core rule:

```txt
Nova code depends on platform contracts, never on platform objects.
```
