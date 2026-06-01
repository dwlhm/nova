export function register(NovaExternal) {
  NovaExternal.define("@env/storage", {
    async load({ input }) {
      const raw = localStorage.getItem(String(input.key ?? ""));
      if (raw == null) return null;
      try {
        return JSON.parse(raw);
      } catch {
        return raw;
      }
    },
    async set({ input }) {
      localStorage.setItem(String(input.key ?? ""), JSON.stringify(input.value));
      return null;
    },
    async remove({ input }) {
      localStorage.removeItem(String(input.key ?? ""));
      return null;
    },
    async clear() {
      localStorage.clear();
      return null;
    }
  });
}
