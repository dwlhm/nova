"use strict";

var NovaExternalCore = (() => {
  function isSerializable(value, seen) {
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
      ok = value.every((item) => isSerializable(item, visited));
    } else {
      ok = Object.keys(value).every((key) => isSerializable(value[key], visited));
    }
    visited.delete(value);
    return ok;
  }

  function externalOperationSpec(app, effectId) {
    return ((app.externalOperations || []).find((entry) => entry.id === effectId)) || null;
  }

  function effectPermissions(app, effectId) {
    const spec = externalOperationSpec(app, effectId);
    if (spec && spec.permissions) return spec.permissions;
    const effect = ((app.effects || []).find((entry) => entry.id === effectId));
    return (effect && effect.permissions) || [];
  }

  function checkPermission(app, effectId) {
    for (const permission of effectPermissions(app, effectId)) {
      if (!((app.permissions || []).includes(permission))) {
        throw new Error("permission denied: " + permission);
      }
    }
  }

  function validateOutput(app, effectId, output) {
    const spec = externalOperationSpec(app, effectId);
    if (spec && spec.output === "void" && output != null) {
      throw new Error("external operation " + effectId + " expected void output");
    }
    if (spec && spec.output === "string" && output != null && typeof output !== "string") {
      throw new Error("external operation " + effectId + " expected string output");
    }
    return output;
  }

  function create() {
    const adapters = new Map();
    return {
      define(source, adapter) {
        adapters.set(source, adapter || {});
      },
      async invoke(effectId, input) {
        const separator = effectId.indexOf("#");
        if (separator <= 0) throw new Error("invalid effect id " + effectId);
        const source = effectId.slice(0, separator);
        const operation = effectId.slice(separator + 1);
        const adapter = adapters.get(source);
        if (!adapter) throw new Error("missing external adapter for " + source);
        const handler = adapter[operation];
        if (typeof handler !== "function") throw new Error(source + " has no operation " + operation);
        const output = await handler({ input: input || {} });
        if (!isSerializable(output)) {
          throw new Error("external operation " + effectId + " returned non-serializable output");
        }
        return output;
      },
      checkPermission,
      validateOutput,
      isSerializable
    };
  }

  return { create, checkPermission, validateOutput, isSerializable };
})();

if (typeof window !== "undefined") {
  window.NovaExternalCore = NovaExternalCore;
}

if (typeof module !== "undefined" && module.exports) {
  module.exports = NovaExternalCore;
}
