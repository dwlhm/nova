"use strict";

window.NovaAppLifecycle = (() => {
  const SOURCE = "@nova/app";
  const SNAPSHOT_KEY = "nova:app:snapshot";

  function activeEvents(app) {
    const values = app && app.appLifecycle ? app.appLifecycle : [];
    return new Set(values.map(String));
  }

  function cloneValue(value) {
    if (value === undefined || value === null) return value;
    if (typeof value !== "object") return value;
    try {
      return JSON.parse(JSON.stringify(value));
    } catch (_) {
      return value;
    }
  }

  function captureSnapshot(runtime) {
    if (!runtime || !runtime.state) return null;
    return {
      source: "session",
      state: cloneValue(runtime.state),
      savedAt: Date.now()
    };
  }

  function persistSnapshot(runtime) {
    const snapshot = captureSnapshot(runtime);
    if (!snapshot) return;
    try {
      sessionStorage.setItem(SNAPSHOT_KEY, JSON.stringify(snapshot));
    } catch (_) {}
  }

  function readSnapshot() {
    try {
      const raw = sessionStorage.getItem(SNAPSHOT_KEY);
      if (!raw) return null;
      return JSON.parse(raw);
    } catch (_) {
      return null;
    }
  }

  function clearSnapshot() {
    try {
      sessionStorage.removeItem(SNAPSHOT_KEY);
    } catch (_) {}
  }

  function restoreSnapshot(runtime, active) {
    if (!active.has("@app_restored")) return;
    const snapshot = readSnapshot();
    if (!snapshot || typeof snapshot !== "object") return;
    if (snapshot.state && typeof snapshot.state === "object") {
      runtime.state = { ...runtime.state, ...snapshot.state };
    }
    runtime.scheduler.enqueue(SOURCE, "@app_restored", [snapshot]);
    runtime.scheduler.drain();
    clearSnapshot();
  }

  function bind(runtime) {
    if (!runtime || !runtime.scheduler || !runtime.app) return;
    const active = activeEvents(runtime.app);
    const enqueue = (name, payload) => {
      if (!active.has(name)) return;
      const args = payload === undefined ? [] : [payload];
      runtime.scheduler.enqueue(SOURCE, name, args);
      runtime.scheduler.drain();
    };

    restoreSnapshot(runtime, active);
    enqueue("@app_started");
    document.addEventListener("visibilitychange", () => {
      if (document.visibilityState === "hidden") enqueue("@app_paused");
      else enqueue("@app_resumed");
    });
    window.addEventListener("pagehide", () => {
      persistSnapshot(runtime);
      enqueue("@app_stopped");
    });
    window.addEventListener("pageshow", () => enqueue("@app_resumed"));
  }

  return { bind, captureSnapshot, restoreSnapshot, persistSnapshot };
})();
