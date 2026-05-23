"use strict";

/**
 * Nova production scheduler for browser runtimes (ADR-002).
 * @module @nova/scheduler
 * @version 0.1.0
 */
var NovaScheduler = (() => {
  function create(host) {
    const scheduler = { queue: [], nextSequence: 1, draining: false, errors: [] };
    scheduler.dispatch = (event, args, options) => {
      const envelope = enqueue(scheduler, host, (options && options.source) || "renderer", event, args || [], options || {});
      if (!envelope) return;
      drain(scheduler, host);
    };
    scheduler.enqueue = (source, event, args, options) => enqueue(scheduler, host, source, event, args || [], options || {});
    scheduler.drain = () => drain(scheduler, host);
    return scheduler;
  }

  function payloadFor(app, event, args) {
    const transitions = ((app.model || {}).states || []).flatMap((state) => state.transitions || []);
    const transition = transitions.find((candidate) => candidate.event === event && (candidate.params || []).length === args.length);
    if (!transition) return {};
    return payloadForTransition(transition, args);
  }

  function payloadForTransition(transition, args) {
    const payload = {};
    (transition.params || []).forEach((name, index) => { payload[name] = args[index]; });
    return payload;
  }

  function enqueue(scheduler, host, source, event, args, options) {
    if (!isSchedulerEvent(event)) {
      recordSchedulerError(scheduler, "enqueue", event, "scheduler event name must start with @");
      return null;
    }
    const eventArgs = (args || []).slice();
    if (!isSerializableNovaData(eventArgs)) {
      recordSchedulerError(scheduler, "enqueue", event, "event payload must be a serializable Nova data value");
      return null;
    }
    const sequence = scheduler.nextSequence || 1;
    const envelope = {
      sequence,
      source,
      name: event,
      args: eventArgs,
      payload: payloadFor(host.app, event, eventArgs),
      options: options || {}
    };
    scheduler.nextSequence = sequence + 1;
    scheduler.queue.push(envelope);
    return envelope;
  }

  function drain(scheduler, host) {
    if (scheduler.draining) return;
    scheduler.draining = true;
    try {
      while (scheduler.queue.length) step(scheduler, host);
    } finally {
      scheduler.draining = false;
    }
  }

  function step(scheduler, host) {
    const envelope = scheduler.queue.shift();
    const beforeRoute = host.routeKey();
    const beforeState = host.cloneState(host.state());
    const commit = planStateCommit(host, envelope, beforeState);
    if (commit.errors.length) {
      scheduler.errors.push(...commit.errors);
      return;
    }
    host.commitState(commit.state);
    host.update(commit.invalidations);
    host.reconcileNavigation(beforeRoute, envelope.options || {});
  }

  function planStateCommit(host, envelope, beforeState) {
    const nextState = host.cloneState(beforeState);
    const candidates = [];
    const errors = [];
    for (const cell of (host.app.model || {}).states || []) {
      for (const transition of (cell.transitions || [])) {
        if (transition.event !== envelope.name) continue;
        try {
          candidates.push({
            name: cell.name,
            value: host.evaluate(transition.expression, beforeState, payloadForTransition(transition, envelope.args || []))
          });
        } catch (error) {
          errors.push({
            source: host.pick(cell, "owner", "Owner", ""),
            phase: "transition",
            event: envelope.name,
            state: cell.name,
            message: error && error.message ? error.message : String(error)
          });
        }
      }
    }
    if (errors.length) return { state: beforeState, invalidations: new Set(), errors };
    for (const candidate of candidates) nextState[candidate.name] = candidate.value;
    if (host.hasRouteState()) {
      nextState.route = host.routeValueForShape(host.routeObject(nextState.route), nextState.route);
    }
    return { state: nextState, invalidations: host.stateInvalidations(beforeState, nextState), errors };
  }

  function recordSchedulerError(scheduler, phase, event, message) {
    scheduler.errors.push({ phase, event, message });
  }

  function isSchedulerEvent(event) {
    return typeof event === "string" && event.startsWith("@") && event.length > 1;
  }

  function isSerializableNovaData(value, seen) {
    if (value === undefined || value === null) return true;
    const type = typeof value;
    if (type === "string" || type === "boolean") return true;
    if (type === "number") return Number.isFinite(value);
    if (type !== "object") return false;
    const visited = seen || new WeakSet();
    if (visited.has(value)) return false;
    visited.add(value);
    let ok = false;
    if (Array.isArray(value)) {
      ok = value.every((item) => isSerializableNovaData(item, visited));
    } else {
      ok = Object.keys(value).every((key) => isSerializableNovaData(value[key], visited));
    }
    visited.delete(value);
    return ok;
  }

  return { create };
})();

if (typeof window !== "undefined") {
  window.NovaScheduler = NovaScheduler;
}

if (typeof module !== "undefined" && module.exports) {
  module.exports = NovaScheduler;
}
