Berikut ADR yang fokus ke **type system** Nova.

---

# ADR-004: Type System

## Status

Implemented / Accepted

## Context

Nova membutuhkan type system yang mendukung:

```txt
pure state transition
serializable scheduler payload
template data binding
external operation contract
multiplatform target adapter
diagnostics yang presisi
```

Type system harus cukup kuat untuk mencegah dirty value dan platform object masuk ke semantic core.
Namun Nova bukan general-purpose language, sehingga type system tidak boleh menjadi terlalu luas.

---

# Decision

Nova menggunakan **strict data-oriented type system**.

Nilai yang boleh mengalir di state, event, props, function, template, dan external contract adalah:

```txt
serializable Nova data value
```

Core type:

```txt
string
number
boolean
unknown
void
null
literal type
array
record
union
opaque nominal type
```

Nova tidak melakukan implicit coercion.

```txt
"1" is not number
1 is not string
null is not void
```

Secara arsitektur, type system adalah bagian dari **functional core**.
Type checker, compatibility check, dan schema derivation harus dapat dijalankan sebagai fungsi murni
dari AST + capability manifest ke diagnostics + typed semantic model.

Dalam model **hexagonal**, type system juga menjadi bahasa contract untuk semua port:

```txt
host event port
scheduler state port
renderer view port
external operation port
package/module port
hydration/devtools port
```

Adapter boleh memakai type native apa pun di sisi platform, tetapi nilai yang melewati port Nova
harus sudah dimarshalling menjadi Nova data value dan lolos validasi contract.

---

# Primitive Types

```nova
name: string;
gain: number;
enabled: boolean;
payload: unknown;
```

Makna:

```txt
string   text Unicode
number   numeric value semantic Nova
boolean  true atau false
unknown  value yang perlu narrowing/validation sebelum dipakai secara spesifik
```

`number` tidak menjanjikan representasi platform tertentu.
Target adapter bertanggung jawab menjaga hasil yang kompatibel dengan conformance test.

---

# Void and Null

`void` dipakai untuk event atau operation tanpa payload.

```nova
@increment: void;
```

Emit event tanpa payload:

```nova
void -> @increment;
```

`null` adalah nilai data.

```nova
selected: User | null <- null;
```

Rule:

```txt
void bukan nilai data yang dapat disimpan sebagai state.
null hanya valid jika type mengizinkan null.
```

---

# Literal Types

Nova mendukung literal type untuk finite state sederhana.

```nova
<contract type Status>
  "idle" | "loading" | "ready" | "error";
/|
```

Literal yang didukung pada production v1:

```txt
string literal
number literal
boolean literal
null literal
```

Literal type berguna untuk:

```txt
status state
mode UI
event discriminator
target-independent enum sederhana
```

---

# Record Types

Record type memakai `:`.

```nova
<contract type User>
  id: UserId;
  name: string;
  email?: string;
/|
```

Rule:

```txt
1. Field wajib harus ada.
2. Field optional boleh tidak ada.
3. Field optional yang ada harus sesuai type.
4. Extra field boleh disimpan pada value internal, tetapi tidak boleh diandalkan kecuali type-nya mencakup field tersebut.
5. Type checker harus memberi diagnostic saat binding mengakses field yang tidak ada.
```

Nova memakai structural compatibility untuk record non-opaque.

---

# Opaque Nominal Types

Opaque type dideklarasikan tanpa body.

```nova
<contract type UserId /|
<contract type ProductId /|
```

Rule:

```txt
1. Opaque type berbeda berdasarkan nama dan capability asal.
2. UserId tidak kompatibel dengan ProductId walaupun representasi runtime sama.
3. Pembuatan opaque value harus melalui capability atau external adapter yang berwenang.
4. Opaque type tetap harus serializable jika masuk state/event.
```

Opaque type dipakai untuk identitas domain, bukan untuk menyimpan platform object.

---

# Functional Type Semantics

Type relation Nova harus deterministic dan bebas side effect.

Input:

```txt
TypeEnvironment
Expression
ExpectedType?
```

Output:

```txt
TypedExpression
Diagnostics[]
```

Rule:

```txt
1. Compatibility adalah pure relation antara source type dan target type.
2. Type checking tidak boleh membaca target platform, filesystem, network, atau runtime object.
3. Type checker tidak boleh menjalankan external operation untuk mengetahui type.
4. Opaque identity berasal dari deklarasi + capability asal, bukan dari representasi runtime.
5. Narrowing menghasilkan type environment baru tanpa mengubah value.
```

Implikasi:

```txt
type inference boleh ditambahkan selama tetap deterministic
diagnostic dapat direproduksi dari source yang sama
build cache aman karena type checking tidak bergantung dunia luar
```

---

# Hexagonal Type Boundaries

Nova membedakan domain type dan adapter representation.

```txt
Domain core:
  TypeRef
  TypedExpression
  DataValue
  EventEnvelope
  ViewIR

Ports:
  HostEventPort
  RendererPort
  ExternalOperationPort
  StateHydrationPort
  PackageBoundaryPort

Adapters:
  Web DOM adapter
  Android adapter
  storage/network/device adapter
  devtools adapter
```

Rule:

```txt
1. Port menerima atau mengirim Nova data value, bukan object platform.
2. Adapter bertanggung jawab marshal native value ke Nova data value.
3. Adapter bertanggung jawab validate output sebelum enqueue event atau commit.
4. Core tidak mengetahui DOM node, UIView, Promise, coroutine, file handle, atau socket.
5. Boundary failure menjadi diagnostic atau SchedulerError, bukan dirty value di core.
```

Contoh:

```txt
DOM click event
  -> Web host adapter
  -> validate payload against @select(UserId)
  -> scheduler enqueue @select(payload)

storage native result
  -> storage adapter
  -> validate result against operation output
  -> enqueue completion event or route error
```

---

# Union Types

Union memakai `|`.

```nova
status: "idle" | "loading" | "ready" | "error";
selected: IEM | null;
payload: string | number;
```

Rule:

```txt
1. Value valid jika cocok dengan salah satu branch.
2. Union harus dinarrow sebelum dipakai sebagai type yang lebih spesifik.
3. Unknown harus divalidasi atau dinarrow sebelum akses field.
4. Union event payload harus tetap serializable.
```

Narrowing expression detail ditunda ke expression specification.
Production validator minimal harus mengecek assignment, event payload, dan function return.

---

# Array Types

Array memakai suffix `[]`.

```nova
items: IEM[];
matrix: number[][];
```

Rule:

```txt
1. Semua item array harus sesuai element type.
2. Array adalah data value, bukan mutable collection.
3. Transition menghasilkan array baru secara semantic.
4. Renderer dapat melakukan structural sharing sebagai optimisasi internal.
```

---

# Function Types

Function declaration wajib menulis type parameter dan return.

```nova
<func gainFor iem: IEM offset: number returns number>
  (iem.impedance / iem.sensitivity * 100) + offset
/|
```

Rule:

```txt
1. Function parameter selalu explicit.
2. Return type wajib untuk function public pada production v1.
3. Function body harus type-check terhadap return type.
4. Function tidak memiliki higher-order function type pada production v1.
5. Function value tidak dapat disimpan di state/event.
```

---

# Event Payload Types

Event payload didefinisikan melalui pattern atau capability contract.

```nova
@set(value: number)
@login_ok(user: User)
```

Canonical payload:

```txt
@set(value: number) -> record payload { value: number }
@increment          -> void payload
```

Rule:

```txt
1. Event name wajib diawali @.
2. Event tanpa payload memakai void.
3. Payload harus serializable.
4. Payload yang dikirim template/lifecycle harus sesuai contract.
5. Transition parameter mendapat binding dari payload event.
```

---

# External Operation Types

External operation memiliki input dan output contract.

```nova
operation load {
  input {
    key: string;
  }

  output unknown;
}
```

Rule:

```txt
1. Input harus serializable.
2. Output harus serializable.
3. Output tidak boleh berupa platform object.
4. Adapter wajib memvalidasi output sebelum masuk scheduler.
5. Kegagalan validasi menjadi scheduler/runtime error.
```

---

# Type Compatibility

Compatibility rule:

```txt
primitive      exact type
literal        value must equal literal
array          element type compatible
record         structural field compatibility
union          compatible with at least one branch
opaque         same declared opaque type identity
unknown        accepts any serializable value, requires narrowing for specific use
void           only void
null           only null or union containing null
```

Tidak ada implicit conversion.

Rejected examples:

```nova
count: number <- "1";
enabled: boolean <- "true";
selected: User <- null;
```

---

# Runtime Validation

Static type checking adalah source of truth.
Runtime validation tetap diperlukan di boundary:

```txt
host event adapter
external operation completion
package boundary
serialized state hydration
devtools injection
```

Runtime validation failure tidak boleh masuk ke transition sebagai value tidak valid.
Failure harus diarahkan ke error lifecycle atau build/runtime diagnostic.

Validation pada boundary adalah adapter concern, tetapi schema dan compatibility rule tetap dimiliki
functional core. Adapter tidak boleh memperluas semantic type tanpa deklarasi contract.

---

# Alternatives Considered

## Gradual Typing

Semua type optional dan error muncul saat runtime.

Rejected because:

```txt
Multiplatform conformance menjadi lemah.
External boundary sulit diamankan.
Diagnostics menjadi terlambat.
```

## Platform Native Types

Template dan external dapat membawa DOM node, UIView, coroutine, atau object native lain.

Rejected because:

```txt
Core scheduler harus platform-neutral.
State/event harus serializable.
Renderer adapter yang boleh memegang platform object.
```

## Full Generic Type System on production v1

Generic type dan higher-order function didukung dari awal.

Out of production v1 scope because:

```txt
Parser, checker, diagnostics, dan lowering menjadi jauh lebih kompleks.
Kebutuhan utama Nova masih dapat dipenuhi oleh data type dasar.
```

---

# Consequences

## Positive

```txt
1. State dan event aman untuk scheduler multiplatform.
2. External adapter memiliki contract validasi jelas.
3. Template binding dapat didiagnosis sebelum runtime.
4. Opaque type menjaga identitas domain.
5. Tidak ada coercion tersembunyi.
```

## Negative

```txt
1. User harus menulis type contract lebih eksplisit.
2. Generic dan higher-order pattern ditunda.
3. Boundary unknown membutuhkan narrowing/validation.
4. Runtime tetap harus melakukan validation pada data dari luar.
```

---

# Final Position

Nova type system adalah:

```txt
strict
data-oriented
serializable
mostly structural
nominal only for opaque type
boundary-validated
functional-core checked
hexagonal-port enforced
```

Core rule:

```txt
Anything crossing scheduler, renderer, host, package, or external boundary must be valid Nova data.
```
