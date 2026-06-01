export function register(NovaExternal) {
  NovaExternal.define("@env/network", {
    async request({ input }) {
      const payload = input && input.input ? input.input : input || {};
      const url = String(payload.url || payload.href || "");
      if (!url) throw new Error("network.request requires url");
      const method = String(payload.method || "GET").toUpperCase();
      const headers = payload.headers && typeof payload.headers === "object" ? payload.headers : {};
      const body = payload.body;
      const init = { method, headers };
      if (body !== undefined && method !== "GET" && method !== "HEAD") {
        init.body = typeof body === "string" ? body : JSON.stringify(body);
      }
      const response = await fetch(url, init);
      const contentType = response.headers.get("content-type") || "";
      let data = null;
      if (contentType.includes("application/json")) {
        data = await response.json();
      } else {
        data = await response.text();
      }
      return {
        ok: response.ok,
        status: response.status,
        data
      };
    }
  });
}
