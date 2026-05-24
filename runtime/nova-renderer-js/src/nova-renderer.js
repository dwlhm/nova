"use strict";

(function initNovaRenderer(root) {
  const primitives = new Map();

  const NovaRenderer = {
    definePrimitive(kind, implementation) {
      if (!kind) return;
      primitives.set(kind, implementation || {});
    },
    primitive(kind) {
      return primitives.get(kind) || null;
    },
    clear() {
      primitives.clear();
    }
  };

  if (root) {
    root.NovaRenderer = root.NovaRenderer || NovaRenderer;
  }
  if (typeof module !== "undefined" && module.exports) {
    module.exports = root && root.NovaRenderer ? root.NovaRenderer : NovaRenderer;
  }
})(typeof window !== "undefined" ? window : globalThis);
