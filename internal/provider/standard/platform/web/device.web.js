export function register(NovaExternal) {
  NovaExternal.define("@env/device", {
    async info() {
      return {
        userAgent: navigator.userAgent,
        platform: navigator.platform,
        language: navigator.language,
        onLine: navigator.onLine
      };
    }
  });
}
