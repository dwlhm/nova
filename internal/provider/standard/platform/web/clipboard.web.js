export function register(NovaExternal) {
  NovaExternal.define("@env/clipboard", {
    async read() {
      if (!navigator.clipboard || typeof navigator.clipboard.readText !== "function") {
        throw new Error("clipboard.read is unavailable");
      }
      return navigator.clipboard.readText();
    },
    async write({ input }) {
      if (!navigator.clipboard || typeof navigator.clipboard.writeText !== "function") {
        throw new Error("clipboard.write is unavailable");
      }
      await navigator.clipboard.writeText(String(input.value ?? ""));
      return null;
    }
  });
}
