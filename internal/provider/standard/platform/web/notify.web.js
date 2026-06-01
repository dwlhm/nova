export function register(NovaExternal) {
  NovaExternal.define("@env/notify", {
    async send({ input }) {
      const message = String(input.msg ?? input.message ?? "");
      if (!message) throw new Error("notify.send requires msg");
      if (typeof Notification === "undefined") {
        throw new Error("Notification API is unavailable");
      }
      if (Notification.permission === "default") {
        await Notification.requestPermission();
      }
      if (Notification.permission !== "granted") {
        throw new Error("notification permission denied");
      }
      new Notification(message, { body: String(input.type ?? "") });
      return null;
    }
  });
}
