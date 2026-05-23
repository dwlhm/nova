"use strict";

const { describe, it } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

function loadScheduler() {
  const source = fs.readFileSync(path.join(__dirname, "..", "src", "nova-scheduler.js"), "utf8");
  const context = { window: {}, console };
  vm.createContext(context);
  vm.runInContext(source, context);
  return context.window.NovaScheduler;
}

function createHost(initial) {
  let state = { ...initial };
  const host = {
    lastInvalidations: null,
    app: {
      model: {
        states: [
          {
            name: "count",
            transitions: [{ event: "@increment", params: [], expression: "count + 1" }]
          }
        ]
      }
    },
    state: () => state,
    cloneState: (value) => ({ ...value }),
    commitState: (next) => { state = next; },
    update: (invalidations) => { host.lastInvalidations = invalidations; },
    evaluate: (expression, current, payload) => {
      if (expression === "count + 1") return current.count + 1;
      throw new Error("unsupported expression: " + expression);
    },
    hasRouteState: () => false,
    routeKey: () => "",
    routeObject: (route) => route,
    routeValueForShape: (_route, shape) => shape,
    stateInvalidations: (before, after) => {
      const changed = new Set();
      for (const key of Object.keys(after)) {
        if (before[key] !== after[key]) changed.add(key);
      }
      return changed;
    },
    reconcileNavigation: () => {},
    pick: (cell, ...keys) => keys.map((key) => cell[key]).find((value) => value) || ""
  };
  return host;
}

const NovaScheduler = loadScheduler();

describe("NovaScheduler", () => {
  it("exports a CommonJS module for Node tests", () => {
    delete require.cache[require.resolve("../src/nova-scheduler.js")];
    const exported = require("../src/nova-scheduler.js");
    assert.equal(typeof exported.create, "function");
  });

  it("rejects events without @ prefix", () => {
    const scheduler = NovaScheduler.create(createHost({ count: 0 }));
    scheduler.dispatch("increment", []);
    assert.equal(scheduler.queue.length, 0);
    assert.equal(scheduler.errors.length, 1);
  });

  it("applies matching transition on dispatch", () => {
    const scheduler = NovaScheduler.create(createHost({ count: 1 }));
    const runtimeHost = createHost({ count: 1 });
    const runtimeScheduler = NovaScheduler.create(runtimeHost);
    runtimeScheduler.dispatch("@increment", []);
    assert.equal(runtimeHost.state().count, 2);
    assert.ok(runtimeHost.lastInvalidations.has("count"));
  });

  it("drains nested dispatches synchronously", () => {
    const runtimeHost = createHost({ count: 0 });
    const scheduler = NovaScheduler.create(runtimeHost);
    scheduler.enqueue("test", "@increment", []);
    scheduler.enqueue("test", "@increment", []);
    scheduler.drain();
    assert.equal(runtimeHost.state().count, 2);
    assert.equal(scheduler.queue.length, 0);
  });
});
