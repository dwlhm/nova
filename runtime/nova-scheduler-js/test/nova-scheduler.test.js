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

  it("runs before and after lifecycle around transition commit", () => {
    const runtimeHost = createHost({ count: 0 });
    runtimeHost.app.lifecycles = [
      {
        owner: "counter",
        phase: "before",
        event: "@increment",
        steps: [{ emit: { name: "@seen_before", args: ["count"] } }]
      },
      {
        owner: "counter",
        phase: "after",
        event: "@increment",
        steps: [{ emit: { name: "@seen_after", args: ["count"] } }]
      }
    ];
    runtimeHost.app.events = [
      { name: "@increment", emitters: ["renderer", "lifecycle"] },
      { name: "@seen_before", emitters: ["counter"] },
      { name: "@seen_after", emitters: ["counter"] }
    ];
    runtimeHost.evaluate = (expression, current) => {
      if (expression === "count + 1") return current.count + 1;
      if (expression === "count + 10") return current.count + 10;
      if (expression === "count") return current.count;
      throw new Error("unsupported expression: " + expression);
    };
    const scheduler = NovaScheduler.create(runtimeHost);
    scheduler.dispatch("@increment", []);
    assert.equal(runtimeHost.state().count, 1);
    assert.equal(scheduler.queue.length, 0);
  });

  it("runs mount lifecycle via runLifecycle", () => {
    const runtimeHost = createHost({ count: 0 });
    runtimeHost.app.lifecycles = [{
      owner: "boot",
      phase: "mount",
      steps: [{ emit: { name: "@increment", args: [] } }]
    }];
    runtimeHost.app.events = [{ name: "@increment", emitters: ["boot", "lifecycle"] }];
    const scheduler = NovaScheduler.create(runtimeHost);
    scheduler.runLifecycle("mount", "", []);
    assert.equal(runtimeHost.state().count, 1);
  });
});
