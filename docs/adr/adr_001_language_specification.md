Berikut ADR yang fokus ke **language specification** Nova.

---

# ADR-001: Language Specification

## Status

Implemented / Accepted

## Context

Nova ditujukan sebagai **multiplatform framework language**, bukan general-purpose programming language. Nova berfokus pada deklarasi aplikasi lintas platform melalui:

```txt
state topology
scheduler event
pure state transition
pure data transformer
template declaration
lifecycle-bound dirty operation
capability file boundary
```

Target utama Nova adalah menjaga **functional core** dan membatasi seluruh dirty operation ke boundary eksplisit.

Nova harus mendukung:

```txt
Web
Android
iOS
Desktop / future target
```

Namun platform-specific detail tidak boleh bocor ke core language.

---

# Decision

Nova menggunakan model bahasa dengan prinsip berikut:

```txt
1. File .nova adalah capability boundary.
2. External script selalu dirty.
3. Dirty operation hanya boleh dilakukan di <lifecycle>.
4. State transition hanya boleh didefinisikan di <contract state>.
5. Function harus pure dan dieksekusi sebagai data pipeline.
6. Template adalah deklarasi view murni dan hanya boleh emit scheduler event.
7. Event scheduler wajib ditandai dengan @.
8. Event tanpa payload memakai void.
9. Contract capability hanya mendefinisikan props dan emitted events.
10. Lifecycle dapat listen/emit event capability lain jika event tersebut di-import.
```

---

# Core Constructs

Nova hanya memiliki construct utama berikut:

```txt
<import>
<contract type>
<contract state>
<contract capability>
<func>
<template>
<lifecycle>
```

Nova tidak memiliki construct berikut:

```txt
<state>
<action>
<effect>
<dirty>
<use>
<bind>
```

Alasannya:

```txt
state       masuk ke <contract state>
action      direpresentasikan sebagai scheduler event + transition
effect      direpresentasikan sebagai lifecycle dirty operation
dirty       tidak perlu construct sendiri karena lifecycle adalah dirty zone
use/bind    dependency dan event/state access cukup lewat import
```

---

# Capability Model

## File as Capability

Setiap file `.nova` adalah capability boundary.

Contoh:

```txt
Counter.nova
Button.nova
CounterStorage.nova
Math.nova
```

Capability dapat berisi:

```txt
import
contract type
contract state
contract capability
func
template
lifecycle
external import
```

Capability tidak harus memiliki semua construct.

---

## Import

`<import>` adalah deklarasi dependency. Import tidak mengeksekusi side effect.

```nova
<import Counter from "./Counter.nova" /|
<import button from "@nova/ui" /|
<import state count from "./Counter.nova" /|
<import event @increment from "./Counter.nova" /|
```

Makna:

```txt
import file       -> memasukkan capability ke dependency graph
import symbol     -> memakai symbol dari capability lain
import state      -> memberi akses baca state dari capability lain
import event      -> memberi akses listen/emit event dari capability lain
```

Import tidak menjalankan lifecycle secara langsung. Scheduler membaca lifecycle dari dependency graph pada runtime/build graph.

---

# Purity Model

Nova menggunakan model:

```txt
functional core + lifecycle-bounded effects
```

## Pure Zone

Construct berikut harus pure:

```txt
<contract type>
<contract state>
<func>
<template>
```

## Effect Zone

Construct berikut dapat melakukan dirty operation:

```txt
<lifecycle>
runtime scheduler commit
renderer/platform backend
```

Userland dirty operation hanya boleh berada di:

```nova
<lifecycle ...>
  ...
/|
```

---

# External Script Rule

External script dari bahasa lain selalu dirty.

Contoh external:

```txt
JavaScript
Kotlin
Swift
Rust
C
Java
```

Nova boleh consume external script hanya sebagai dirty dependency.

```nova
<import external storage from "./storage.web.js">
  operation set {
    input {
      key: string;
      value: unknown;
    }

    output void;
  }

  operation remove {
    input {
      key: string;
    }

    output void;
  }
 /|
```

External operation hanya boleh dipanggil di `<lifecycle>`.

Valid:

```nova
<lifecycle after @increment>
  count |> storage.set key <- "counter";
 /|
```

Invalid:

```nova
<func save value: number returns void>
  value |> storage.set key <- "counter"
/|
```

Invalid:

```nova
<contract state Counter>
  count: number <- 0 {
    @increment -> count |> storage.set key <- "counter";
  };
/|
```

Reason:

```txt
External selalu dirty.
Dirty hanya lifecycle.
```

---

# Type Contract

`<contract type>` dipakai untuk mendefinisikan:

```txt
opaque nominal type
type alias
record type
```

## Opaque Nominal Type

```nova
<contract type UserId /|
<contract type ProductId /|
```

Makna:

```txt
UserId dan ProductId adalah type berbeda.
Representasi runtime disembunyikan.
```

Ini bukan constant. Ini adalah kategori nilai.

Contoh:

```nova
<contract type User>
  id: UserId;
  name: string;
  email?: string;
/|
```

## Type Alias

```nova
<contract type Status>
  "idle" | "loading" | "success" | "error";
/|
```

## Record Type

```nova
<contract type User>
  id: UserId;
  name: string;
  email?: string;
/|
```

## Rule

Type declaration menggunakan `:`.

Value/data binding menggunakan `<-`.

```nova
name: string;      // type field
name <- "Dwi";     // value binding
```

---

# State Contract

`<contract state>` mendefinisikan state node dan pure transition rule.

```nova
<contract state Counter>
  count: number <- 0 {
    @increment -> count |> add 1;
    @decrement -> count |> sub 1;
    @reset -> 0;
  };
/|
```

Makna:

```txt
count adalah scheduler-managed state.
@increment menghasilkan next value untuk count.
Scheduler yang melakukan commit state.
```

Transition tidak melakukan mutation langsung.

Secara teori:

```txt
State × Event -> State
```

Contoh:

```txt
transition(count, @increment) = add(count, 1)
transition(count, @reset) = 0
```

## Multiple State

```nova
<contract state Login>
  status: Status <- "idle" {
    @submit -> "loading";
    @login_ok(user: User) -> "success";
    @login_failed(message: string) -> "error";
  };

  user: User | null <- null {
    @login_ok(user: User) -> user;
    @logout -> null;
  };

  error: string | null <- null {
    @submit -> null;
    @login_failed(message: string) -> message;
  };
/|
```

Satu event dapat memengaruhi beberapa state.

---

# Scheduler Event

Scheduler event wajib menggunakan prefix `@`.

```nova
@increment
@set_count
@login_ok
```

Event dengan payload:

```nova
@set(value: number)
@login_ok(user: User)
```

Event tanpa payload menggunakan `void`.

```txt
@increment: void
```

Emit tanpa payload menggunakan `void`.

```nova
void -> @increment;
```

Dalam template, shorthand ini valid:

```nova
<button on_press -> @increment>
  +
/|
```

Secara canonical:

```txt
on_press emits @increment(void)
```

---

# Function

`<func>` adalah pure synchronous transformer.

```nova
<func add value: number amount: number returns number>
  value + amount
/|
```

Func tidak boleh:

```txt
membaca state global
menulis state
dispatch event
memanggil external operation
mengakses time/random/storage/network/platform API
```

## Function Execution

Function execution menggunakan pipeline.

```nova
count |> add 1
```

Bukan:

```nova
add(count, 1)
```

Nova surface syntax harus flow-based.

Dengan named argument:

```nova
price |> formatCurrency currency <- "IDR"
```

Makna:

```txt
source |> transformer args...
```

Function adalah transformer dari data source.

---

# Template

`<template>` adalah pure view declaration.

Template boleh:

```txt
membaca state
menerima props
membentuk view declaration
emit scheduler event
```

Template tidak boleh:

```txt
memanggil external
melakukan dirty operation
menulis state langsung
```

Contoh:

```nova
<template>
  <button on_press -> @increment>
    +
  /|
/|
```

## Multiplatform Capability Resolution

Template dapat memiliki target:

```nova
<template target <- web>
  ...
/|
```

Atau tanpa target:

```nova
<template>
  ...
/|
```

Template tanpa target adalah target-polymorphic.

Ia valid untuk target build jika semua capability/form yang dipakai memiliki implementation untuk target tersebut.

Contoh:

```nova
<import button from "@nova/ui" /|

<template>
  <button on_press -> @increment>
    +
  /|
/|
```

Jika `button` memiliki implementation untuk `web` dan `android`, maka template valid untuk build target `web` dan `android`.

---

# Contract Capability

`<contract capability>` mendefinisikan public interface capability ketika capability digunakan sebagai component/form.

Fokus:

```txt
props / parameter
emitted events
```

Tidak dipakai untuk lifecycle dependency. Lifecycle dependency cukup melalui import.

## Example

```nova
<contract capability Button>
  props {
    label: string;
    disabled?: boolean;
  }

  emits {
    @pressed: void;
  }
/|
```

Internal template:

```nova
<template>
  <button disabled <- disabled on_press -> @pressed>
    <text value <- label /|
  /|
/|
```

Usage:

```nova
<Button label <- "Save" @pressed -> @save /|
```

## Event Surface Mapping

Capability can emit event:

```nova
@pressed
```

Caller maps emitted event:

```nova
@pressed -> @save
```

Payload is forwarded.

Example with payload:

```nova
<contract capability UserItem>
  props {
    user: User;
  }

  emits {
    @selected: UserId;
  }
/|

<template>
  <button on_press -> @selected(user.id)>
    <text value <- user.name /|
  /|
/|
```

Usage:

```nova
<UserItem user <- user @selected -> @open_user /|
```

---

# Lifecycle

`<lifecycle>` is the only userland dirty zone.

Lifecycle can:

```txt
listen scheduler event
read imported state
call external dirty operation
emit scheduler event
```

Lifecycle cannot:

```txt
write state directly
define state transition
be imported/exported as symbol
be called manually
```

## Listen Event

Lifecycle can listen event because the event is imported.

```nova
<import event @increment from "./Counter.nova" /|
<import state count from "./Counter.nova" /|

<lifecycle after @increment>
  count |> storage.set key <- "counter";
 /|
```

## Emit Event From Other Capability

Lifecycle may emit an event from another capability if that event is imported.

```nova
<import event @set from "./Counter.nova" /|

<lifecycle mount>
  10 -> @set;
 /|
```

No `<bind>` is needed.

## Lifecycle Phases

Minimal lifecycle phases:

```txt
mount
dispose
before @event
after @event
error
```

Example:

```nova
<lifecycle after @increment>
  count |> analytics.track name <- "counter_incremented";
 /|
```

---

# Event Dispatch Rules

## From Template

```nova
<button on_press -> @increment>
  +
/|
```

## With Payload

```nova
<button on_press -> @set_count(10)>
  Set 10
/|
```

## From Lifecycle

```nova
10 -> @set_count;
```

## Void Event From Lifecycle

```nova
void -> @storage_saved;
```

---

# Data Binding

Data binding uses:

```nova
receiver <- source
```

Example:

```nova
title <- "Dashboard"
value <- count
disabled <- isLoading
```

Data binding is not event dispatch.

Event dispatch uses:

```nova
event_slot -> @event
```

---

# Record and Array

Array:

```nova
[one, two, three]
```

Record value:

```nova
{
  id <- userId;
  name <- "Dwi";
}
```

Record type:

```nova
{
  id: UserId;
  name: string;
}
```

Value uses `<-`.

Type uses `:`.

---

# Minimal Grammar Sketch

## Top Level

```ebnf
nova_file =
  { top_level_item } ;

top_level_item =
    import_form
  | external_import_form
  | contract_type_form
  | contract_state_form
  | contract_capability_form
  | func_form
  | template_form
  | lifecycle_form ;
```

## Import

```ebnf
import_form =
  "<import", import_spec, "from", string_lit, "/|" ;

import_spec =
    import_items
  | "state", import_items
  | "event", event_import_items ;

import_items =
  import_item, { ",", import_item } ;

import_item =
  identifier, [ "as", identifier ] ;

event_import_items =
  event_import_item, { ",", event_import_item } ;

event_import_item =
  "@", identifier ;
```

## External Import

```ebnf
external_import_form =
  "<import", "external", identifier, "from", string_lit, ">",
  { external_operation },
  "/|" ;

external_operation =
  "operation", identifier, "{",
    input_section,
    output_section,
  "}" ;
```

External is always dirty.

## Contract Type

```ebnf
contract_type_form =
    contract_type_empty
  | contract_type_body ;

contract_type_empty =
  "<contract", "type", identifier, "/|" ;

contract_type_body =
  "<contract", "type", identifier, ">",
  contract_type_content,
  "/|" ;

contract_type_content =
    record_type_body
  | alias_type_body ;

record_type_body =
  type_field_decl, { type_field_decl } ;

type_field_decl =
  identifier, [ "?" ], ":", type_expr, ";" ;

alias_type_body =
  type_expr, ";" ;
```

## Contract State

```ebnf
contract_state_form =
  "<contract", "state", identifier, ">",
  { state_decl },
  "/|" ;

state_decl =
  identifier, ":", type_expr, "<-", expr,
  [ transition_block ],
  ";" ;

transition_block =
  "{", { transition_rule }, "}" ;

transition_rule =
  scheduler_event_pattern, "->", expr, ";" ;

scheduler_event_pattern =
    "@", identifier
  | "@", identifier, "(", event_param_list, ")" ;

event_param_list =
  event_param, { ",", event_param } ;

event_param =
  identifier, ":", type_expr ;
```

## Contract Capability

```ebnf
contract_capability_form =
  "<contract", "capability", identifier, ">",
  { capability_section },
  "/|" ;

capability_section =
    props_section
  | emits_section ;

props_section =
  "props", "{", { prop_decl }, "}" ;

prop_decl =
  identifier, [ "?" ], ":", type_expr, ";" ;

emits_section =
  "emits", "{", { emit_decl }, "}" ;

emit_decl =
  "@", identifier, ":", type_expr, ";" ;
```

## Func

```ebnf
func_form =
  "<func", identifier, { func_param }, [ "returns", type_expr ], ">",
  expr,
  "/|" ;

func_param =
  identifier, ":", type_expr ;
```

## Template

```ebnf
template_form =
  "<template", [ "target", "<-", identifier ], ">",
  template_body,
  "/|" ;
```

## Lifecycle

```ebnf
lifecycle_form =
  "<lifecycle", lifecycle_trigger, ">",
  { lifecycle_statement },
  "/|" ;

lifecycle_trigger =
    lifecycle_phase
  | lifecycle_phase, scheduler_event_ref ;

lifecycle_phase =
  "mount" | "dispose" | "before" | "after" | "error" ;

scheduler_event_ref =
  "@", identifier ;

lifecycle_statement =
  expr, ";" ;
```

---

# Consequences

## Positive

```txt
1. Nova keeps a pure functional core.
2. All dirty operations are explicit in lifecycle.
3. External interop is possible without polluting pure zones.
4. State changes are controlled by scheduler.
5. Multiplatform rendering is capability-driven.
6. Event dependencies are explicit through import.
7. Components/capabilities have clear props and emit contracts.
```

## Negative

```txt
1. More static analysis is required.
2. Lifecycle/event graph must be tracked by compiler/runtime.
3. External code cannot be reused in pure logic.
4. Some common imperative patterns are intentionally disallowed.
5. Capability authors must define clear props/emits.
```

---

# Final Position

Nova is:

```txt
a scheduler-centered multiplatform framework language
with a pure functional core
and lifecycle-bounded dirty effects.
```

Core guarantee:

```txt
All state transitions are pure.
All functions are pure.
All templates are declarative.
All external operations are dirty.
All dirty operations are isolated in lifecycle.
All state commits are scheduler-owned.
```
