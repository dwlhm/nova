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
    scheduler.runLifecycle = (phase, event, args, options) => {
      const envelope = syntheticEnvelope(scheduler, phase, event, args || [], options || {});
      const snapshot = host.cloneState(host.state());
      const pending = runLifecyclePhase(host, phase, envelope, snapshot, null);
      enqueuePending(scheduler, host, pending);
      drain(scheduler, host);
    };
    scheduler.commitTransition = (event, args) => {
      if (!isSchedulerEvent(event)) return;
      const eventArgs = (args || []).slice();
      const envelope = {
        name: event,
        args: eventArgs,
        payload: payloadFor(host.app, event, eventArgs)
      };
      const commit = planStateCommit(host, envelope, host.cloneState(host.state()));
      if (commit.errors.length) {
        scheduler.errors.push(...commit.errors);
        return;
      }
      host.commitState(commit.state);
      host.update(commit.invalidations);
    };
    return scheduler;
  }

  function syntheticEnvelope(scheduler, phase, event, args, options) {
    return {
      sequence: scheduler.nextSequence || 1,
      source: (options && options.source) || "runtime",
      name: event || "",
      args: args.slice(),
      payload: {},
      options: options || {},
      phase
    };
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

  function eventContract(app, name) {
    return ((app.events || []).find((contract) => contract.name === name)) || null;
  }

  function validateRouteData(value) {
    if (value === undefined || value === null) {
      return "route payload required";
    }
    let rawPath;
    if (typeof value === "string") {
      rawPath = value.trim() || "/";
    } else if (typeof value === "object" && !Array.isArray(value)) {
      rawPath = value.path == null ? "/" : String(value.path);
    } else {
      return "route payload must be serializable Nova data";
    }
    if (!rawPath.startsWith("/")) {
      return "route path must start with /";
    }
    const route = normalizeRouteValue(value);
    if (!isSerializableNovaData(route)) {
      return "route payload must be serializable Nova data";
    }
    return null;
  }

  function normalizeRouteValue(value) {
    if (value === undefined || value === null) {
      return { path: "/" };
    }
    if (typeof value === "string") {
      const path = value.trim() || "/";
      return { path: path.startsWith("/") ? path : "/" + path };
    }
    if (typeof value === "object" && !Array.isArray(value)) {
      const path = value.path == null ? "/" : String(value.path);
      return { ...value, path: path.startsWith("/") ? path : "/" + path };
    }
    return { path: "/" };
  }

  function validateHostEvent(app, source, event, args) {
    const contracts = app.events || [];
    if (contracts.length === 0) return null;
    const contract = eventContract(app, event);
    if (!contract) return "undeclared scheduler event " + event;
    if (contract.emitters && contract.emitters.length && !contract.emitters.includes(source)) {
      return source + " cannot emit scheduler event " + event;
    }
    if (!isSerializableNovaData(args)) {
      return "event payload must be a serializable Nova data value";
    }
    if (event === "@route_changed" && args.length > 0) {
      const routeError = validateRouteData(args[0]);
      if (routeError) return routeError;
    }
    return null;
  }

  function enqueue(scheduler, host, source, event, args, options) {
    if (!isSchedulerEvent(event)) {
      recordSchedulerError(scheduler, "enqueue", event, "scheduler event name must start with @");
      return null;
    }
    const eventArgs = (args || []).slice();
    const validationError = validateHostEvent(host.app, source, event, eventArgs);
    if (validationError) {
      recordSchedulerError(scheduler, "enqueue", event, validationError);
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
    envelope.payload = payloadFor(host.app, envelope.name, envelope.args || []);
    const beforeRoute = host.routeKey();
    const beforeState = host.cloneState(host.state());
    let pending = runLifecyclePhase(host, "before", envelope, beforeState, null);
    const commit = planStateCommit(host, envelope, beforeState);
    if (commit.errors.length) {
      pending = pending.concat(runLifecyclePhase(host, "error", envelope, beforeState, commit.errors[0]));
      scheduler.errors.push(...commit.errors);
      enqueuePending(scheduler, host, pending);
      return;
    }
    host.commitState(commit.state);
    const afterState = host.cloneState(host.state());
    pending = pending.concat(runLifecyclePhase(host, "after", envelope, afterState, null));
    host.update(commit.invalidations);
    host.reconcileNavigation(beforeRoute, envelope.options || {});
    enqueuePending(scheduler, host, pending);
  }

  function runLifecyclePhase(host, phase, envelope, snapshot, routedError) {
    if (phase === "after" && typeof host.shouldSkipLifecycleAfter === "function" && host.shouldSkipLifecycleAfter(envelope.name)) {
      return [];
    }
    const handlers = matchingLifecycles(host.app, phase, envelope.name);
    const pending = [];
    for (const handler of handlers) {
      const output = runLifecycleHandler(host, handler, envelope, snapshot, routedError);
      if (output.failed) {
        const errorRecord = lifecycleErrorRecord(handler, envelope, output.failed);
        pending.push(...runLifecyclePhase(host, "error", envelope, snapshot, errorRecord));
        continue;
      }
      pending.push(...output.pending);
    }
    return pending;
  }

  function runLifecycleHandler(host, handler, envelope, snapshot, routedError) {
    const pending = [];
    let failed = null;
    for (const step of handler.steps || []) {
      try {
        if (step.emit) {
          const args = (step.emit.args || []).map((expr) => host.evaluate(expr, snapshot, envelope.payload || {}));
          pending.push({ source: handler.owner, name: step.emit.name, args });
          continue;
        }
        if (step.external) {
          const input = {};
          for (const [key, expr] of Object.entries(step.external.input || {})) {
            input[key] = host.evaluate(expr, snapshot, envelope.payload || {});
          }
          if (typeof host.invokeExternal === "function") {
            host.invokeExternal({
              owner: handler.owner,
              effectId: step.external.effectId,
              input,
              onSuccess: step.external.onSuccess || "",
              onFailure: step.external.onFailure || ""
            }, pending);
          }
        }
      } catch (error) {
        failed = error && error.message ? error.message : String(error);
        pending.length = 0;
        break;
      }
    }
    return { pending, failed };
  }

  function lifecycleErrorRecord(handler, envelope, message) {
    return {
      source: handler.owner,
      phase: "lifecycle-" + handler.phase,
      event: envelope.name,
      message
    };
  }

  function matchingLifecycles(app, phase, eventName) {
    return (app.lifecycles || []).filter((handler) => {
      if (handler.phase !== phase) return false;
      if (phase === "before" || phase === "after") return handler.event === eventName;
      if (phase === "error") return !handler.event || handler.event === eventName;
      return true;
    });
  }

  function enqueuePending(scheduler, host, pending) {
    for (const item of pending || []) {
      if (item.kind === "external") continue;
      const source = item.source || "lifecycle";
      const args = item.args || [];
      if (item.payload !== undefined) {
        enqueue(scheduler, host, source, item.name, Array.isArray(item.payload) ? item.payload : [item.payload], {});
        continue;
      }
      enqueue(scheduler, host, source, item.name, args, {});
    }
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
            source: cell.owner || "",
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
