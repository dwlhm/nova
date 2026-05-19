package artifact

func webCSS() string {
	return `:root {
  color-scheme: light;
  font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}

* { box-sizing: border-box; }

body {
  margin: 0;
  min-height: 100vh;
  background: #f5f7fb;
  color: #18202f;
}

#nova-root {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 24px;
}

.counter-shell {
  width: min(420px, 100%);
  display: grid;
  gap: 18px;
  padding: 24px;
  border: 1px solid #d8dee9;
  border-radius: 8px;
  background: #ffffff;
  box-shadow: 0 18px 50px rgba(39, 52, 79, 0.12);
}

.counter-title {
  font-size: 20px;
  font-weight: 700;
}

.counter-value {
  font-size: 48px;
  font-weight: 800;
  text-align: center;
}

.counter-actions {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

button {
  min-height: 44px;
  border: 1px solid #c4cad4;
  border-radius: 6px;
  background: #ffffff;
  color: #18202f;
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}

button:hover { background: #eef3ff; }
`
}

func webRuntime() string {
	return `"use strict";

window.NovaRuntime = (() => {
  function pick(value, lower, upper, fallback) {
    if (!value) return fallback;
    if (Object.prototype.hasOwnProperty.call(value, lower)) return value[lower];
    if (Object.prototype.hasOwnProperty.call(value, upper)) return value[upper];
    return fallback;
  }

  function tokenExpression(tokens, stateNames) {
    return (tokens || []).map((token) => {
      const type = pick(token, "type", "Type", "");
      const literal = pick(token, "literal", "Literal", "");
      if (type === "STRING") return JSON.stringify(literal);
      if (type === "IDENT" && stateNames.has(literal)) return "state." + literal;
      if (type === "true" || type === "false" || type === "null") return type;
      if (type === "void") return "undefined";
      return literal;
    }).filter(Boolean).join(" ");
  }

  function evaluate(expression, state, payload) {
    if (!expression || !expression.trim()) return "";
    return Function("state", "payload", "\"use strict\"; return (" + expression + ");")(state, payload || {});
  }

  function bindingExpression(binding, app) {
    if (!binding) return "\"\"";
    const tokens = pick(binding, "tokens", "Tokens", []);
    if (tokens.length) return tokenExpression(tokens, stateNameSet(app));
    const text = pick(binding, "text", "Text", "");
    return JSON.stringify(text);
  }

  function stateNameSet(app) {
    return new Set(((app.model || {}).states || []).map((state) => state.name));
  }

  function initialState(app) {
    const state = {};
    for (const cell of (app.model || {}).states || []) {
      state[cell.name] = evaluate(cell.initial, state, {});
    }
    return state;
  }

  function payloadFor(app, event, args) {
    const transitions = ((app.model || {}).states || []).flatMap((state) => state.transitions || []);
    const transition = transitions.find((candidate) => candidate.event === event && (candidate.params || []).length === args.length);
    if (!transition) return {};
    const payload = {};
    (transition.params || []).forEach((name, index) => { payload[name] = args[index]; });
    return payload;
  }

  function dispatch(runtime, event, args) {
    const payload = payloadFor(runtime.app, event, args || []);
    for (const cell of (runtime.app.model || {}).states || []) {
      const transition = (cell.transitions || []).find((candidate) => candidate.event === event);
      if (!transition) continue;
      runtime.state[cell.name] = evaluate(transition.expression, runtime.state, payload);
    }
    render(runtime);
  }

  function render(runtime) {
    const root = runtime.root;
    root.replaceChildren(...renderNodes(pick(runtime.app.viewIR, "nodes", "Nodes", []), runtime));
  }

  function renderNodes(nodes, runtime) {
    return (nodes || []).map((node) => renderNode(node, runtime)).filter(Boolean);
  }

  function renderNode(node, runtime) {
    const kind = pick(node, "kind", "Kind", "div");
    const props = pick(node, "props", "Props", {}) || {};
    const events = pick(node, "events", "Events", {}) || {};
    const children = pick(node, "children", "Children", []) || [];
    if (kind === "#text") {
      return document.createTextNode(String(evaluate(bindingExpression(props.value, runtime.app), runtime.state, {})));
    }
    const element = document.createElement(tagFor(kind));
    applyProps(element, props, runtime);
    applyEvents(element, events, runtime);
    element.append(...renderNodes(children, runtime));
    if (kind === "text" && !children.length && props.value) {
      element.textContent = String(evaluate(bindingExpression(props.value, runtime.app), runtime.state, {}));
    }
    return element;
  }

  function tagFor(kind) {
    if (kind === "surface") return "section";
    if (kind === "row" || kind === "column" || kind === "stack") return "div";
    if (kind === "text") return "span";
    if (kind === "button") return "button";
    if (/^[a-z][a-z0-9-]*$/.test(kind)) return kind;
    return "div";
  }

  function applyProps(element, props, runtime) {
    for (const [name, binding] of Object.entries(props || {})) {
      if (name === "value") continue;
      const value = evaluate(bindingExpression(binding, runtime.app), runtime.state, {});
      if (name === "class") element.className = String(value);
      else if (name === "label") element.setAttribute("aria-label", String(value));
      else if (name === "enabled" && value === false) element.setAttribute("disabled", "");
      else element.setAttribute(name.replaceAll("_", "-"), String(value));
    }
  }

  function applyEvents(element, events, runtime) {
    for (const [slot, route] of Object.entries(events || {})) {
      const eventName = pick(route, "event", "Event", "");
      const args = pick(route, "args", "Args", []) || [];
      if (slot === "on_press") {
        element.addEventListener("click", () => {
          const values = args.map((arg) => evaluate(bindingExpression(arg, runtime.app), runtime.state, {}));
          dispatch(runtime, eventName, values);
        });
      }
    }
  }

  function mount(app) {
    const root = document.getElementById("nova-root");
    const runtime = { app, root, state: initialState(app) };
    window.__NOVA_RUNTIME__ = runtime;
    render(runtime);
    return runtime;
  }

  return { mount };
})();
`
}

func webBundle(bundle irBundle) string {
	return `"use strict";
window.__NOVA_APP__ = ` + mustJSON(bundle) + `;
if (window.NovaRuntime) {
  window.NovaRuntime.mount(window.__NOVA_APP__);
}
`
}
