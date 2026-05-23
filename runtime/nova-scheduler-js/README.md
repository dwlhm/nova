# @nova/scheduler

Production scheduler for Nova web targets. Implements [ADR-002](../../docs/adr/adr_002_scheduler_runtime_semantics.md) queue, envelope, and state commit semantics.

The host runtime (`nova-web-runtime`) supplies evaluation, route shaping, and view updates through a `host` object passed to `NovaScheduler.create(host)`.

## Install

```bash
npm install @nova/scheduler
```

For local development inside the Nova monorepo, reference this directory:

```bash
npm install file:../../runtime/nova-scheduler-js
```

## Usage

Load the script before the Nova web runtime:

```html
<script src="node_modules/@nova/scheduler/src/nova-scheduler.js"></script>
<script src="nova-runtime.js"></script>
```

```js
const scheduler = window.NovaScheduler.create(host);
scheduler.dispatch("@increment", []);
```

Node-based tests can import the same source:

```js
const NovaScheduler = require("@nova/scheduler");
```

## Host contract

| Method | Purpose |
| --- | --- |
| `state()` | Current scheduler-owned state map |
| `cloneState(state)` | Immutable copy for commit planning |
| `commitState(next)` | Apply committed state |
| `update(invalidations)` | Notify renderer after commit |
| `evaluate(expr, state, payload)` | Run transition expression |
| `hasRouteState()` | Whether `route` cell exists |
| `routeKey()` / `routeObject()` / `routeValueForShape()` | Route normalization |
| `stateInvalidations(before, after)` | Changed state names |
| `reconcileNavigation(beforeRoute, options)` | History/back stack |
| `app` | App IR (`model.states`, transitions) |

## Versioning

Package version tracks Nova `schedulerVersion` metadata (`0.1.0` in artifact `metadata.json`).

## Test

```bash
npm test
```
