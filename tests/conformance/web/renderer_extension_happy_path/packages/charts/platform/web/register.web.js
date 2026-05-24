export function register(NovaRenderer) {
  NovaRenderer.definePrimitive("sparkline", {
    mount() {
      return document.createElement("canvas");
    }
  });
}
