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

[data-nova-kind="surface"] {
  width: min(760px, 100%);
  display: grid;
  gap: 18px;
  padding: 24px;
  border: 1px solid #d8dee9;
  border-radius: 8px;
  background: #ffffff;
  box-shadow: 0 18px 50px rgba(39, 52, 79, 0.12);
}

[data-nova-kind="row"] {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

[data-nova-kind="column"] {
  display: grid;
  gap: 12px;
}

[data-nova-kind="text"] {
  max-width: 58ch;
  font-size: 16px;
  line-height: 1.6;
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
  const ROUTE_CHANGED_EVENT = "@route_changed";

  function pick(value, lower, upper, fallback) {
    if (!value) return fallback;
    if (Object.prototype.hasOwnProperty.call(value, lower)) return value[lower];
    if (Object.prototype.hasOwnProperty.call(value, upper)) return value[upper];
    return fallback;
  }

  function tokenExpression(tokens, stateNames) {
    return expressionFromTokens(tokens || [], stateNames);
  }

  function expressionFromTokens(tokens, stateNames) {
    const clean = trimExpressionTokens(tokens);
    const record = recordExpression(clean, stateNames);
    if (record) return record;
    return clean.map((token) => {
      const type = pick(token, "type", "Type", "");
      const literal = pick(token, "literal", "Literal", "");
      if (type === "STRING") return JSON.stringify(literal);
      if (type === "IDENT" && stateNames.has(literal)) return "state." + literal;
      if (type === "true" || type === "false" || type === "null") return type;
      if (type === "void") return "undefined";
      return literal;
    }).filter(Boolean).join(" ");
  }

  function recordExpression(tokens, stateNames) {
    if (tokens.length < 2 || tokenType(tokens[0]) !== "{" || tokenType(tokens[tokens.length - 1]) !== "}") {
      return null;
    }
    const fields = recordFields(tokens.slice(1, -1));
    if (!fields) return null;
    return "({ " + fields.map((field) => propertyName(field.name) + ": " + expressionFromTokens(field.value, stateNames)).join(", ") + " })";
  }

  function recordFields(tokens) {
    const fields = [];
    let pos = 0;
    while (pos < tokens.length) {
      while (pos < tokens.length && isRecordSeparator(tokenType(tokens[pos]))) pos++;
      if (pos >= tokens.length) break;

      const name = tokenLiteral(tokens[pos]);
      if (!name) return null;
      pos++;
      if (pos >= tokens.length || tokenType(tokens[pos]) !== "<-") return null;
      pos++;

      const start = pos;
      let depth = 0;
      while (pos < tokens.length) {
        const type = tokenType(tokens[pos]);
        if (depth === 0 && isRecordSeparator(type)) break;
        depth = expressionDepth(depth, type);
        pos++;
      }
      const value = trimExpressionTokens(tokens.slice(start, pos));
      if (!value.length) return null;
      fields.push({ name, value });
    }
    return fields.length ? fields : null;
  }

  function trimExpressionTokens(tokens) {
    let start = 0;
    while (start < tokens.length && isExpressionNoise(tokenType(tokens[start]))) start++;
    let end = tokens.length;
    while (end > start && isExpressionNoise(tokenType(tokens[end - 1]))) end--;
    return tokens.slice(start, end);
  }

  function expressionDepth(depth, type) {
    if (type === "(" || type === "[" || type === "{") return depth + 1;
    if ((type === ")" || type === "]" || type === "}") && depth > 0) return depth - 1;
    return depth;
  }

  function tokenType(token) {
    return pick(token, "type", "Type", "");
  }

  function tokenLiteral(token) {
    return pick(token, "literal", "Literal", "");
  }

  function propertyName(name) {
    return /^[A-Za-z_$][A-Za-z0-9_$]*$/.test(name) ? name : JSON.stringify(name);
  }

  function isExpressionNoise(type) {
    return type === "EOF" || type === "//" || type === ";";
  }

  function isRecordSeparator(type) {
    return type === ";" || type === "," || type === "//";
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

  function dispatch(runtime, event, args, options) {
    const beforeRoute = routeKey(runtime);
    const payload = payloadFor(runtime.app, event, args || []);
    for (const cell of (runtime.app.model || {}).states || []) {
      const transition = (cell.transitions || []).find((candidate) => candidate.event === event);
      if (!transition) continue;
      runtime.state[cell.name] = evaluate(transition.expression, runtime.state, payload);
    }
    render(runtime);
    reconcileNavigation(runtime, beforeRoute, options || {});
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
    if (kind === "page") {
      return renderPage(props, children, runtime);
    }
    if (kind === "#text") {
      return document.createTextNode(String(evaluate(bindingExpression(props.value, runtime.app), runtime.state, {})));
    }
    const element = document.createElement(tagFor(kind));
    element.dataset.novaKind = kind;
    applyProps(element, props, runtime);
    applyEvents(element, events, runtime);
    element.append(...renderNodes(children, runtime));
    if (kind === "text" && !children.length && props.value) {
      element.textContent = String(evaluate(bindingExpression(props.value, runtime.app), runtime.state, {}));
    }
    return element;
  }

  function renderPage(props, children, runtime) {
    const expected = normalizePath(evaluate(bindingExpression(props.path, runtime.app), runtime.state, {}));
    if (expected !== activeRoutePath(runtime)) return null;
    const fragment = document.createDocumentFragment();
    fragment.append(...renderNodes(children, runtime));
    return fragment;
  }

  function activeRoutePath(runtime) {
    const route = runtime.state.route;
    if (typeof route === "string") return normalizePath(route);
    if (route && typeof route.path === "string") return normalizePath(route.path);
    return "/";
  }

  function routeStateCell(app) {
    return ((app.model || {}).states || []).find((state) => state.name === "route");
  }

  function hasRouteState(runtime) {
    return !!routeStateCell(runtime.app);
  }

  function routeKey(runtime) {
    if (!hasRouteState(runtime)) return null;
    return routeURL(routeObject(runtime.state.route));
  }

  function canUseHistory(runtime) {
    return hasRouteState(runtime) &&
      typeof window !== "undefined" &&
      !!window.history &&
      !!window.location &&
      window.location.protocol !== "file:";
  }

  function setupNavigation(runtime) {
    if (!canUseHistory(runtime)) return;
    const platformRoute = routeObjectFromLocation(window.location);
    if (isExplicitPlatformRoute(window.location)) {
      runtime.state.route = routeValueForShape(platformRoute, runtime.state.route);
    }
    writeHistoryState(runtime, "replace");
    window.addEventListener("popstate", (event) => {
      const route = event && event.state && event.state.novaRoute
        ? routeObject(event.state.novaRoute)
        : routeObjectFromLocation(window.location);
      dispatch(runtime, ROUTE_CHANGED_EVENT, [routeValueForShape(route, runtime.state.route)], { source: "platform" });
    });
  }

  function reconcileNavigation(runtime, beforeRoute, options) {
    if (!canUseHistory(runtime)) return;
    const afterRoute = routeKey(runtime);
    if (beforeRoute === afterRoute) return;
    writeHistoryState(runtime, options.source === "platform" ? "replace" : "push");
  }

  function writeHistoryState(runtime, mode) {
    const url = routeURL(routeObject(runtime.state.route));
    const method = mode === "push" ? "pushState" : "replaceState";
    try {
      if (currentPlatformRouteKey() === url && method === "pushState") {
        window.history.replaceState(historyState(runtime.state.route), "", url);
        return;
      }
      window.history[method](historyState(runtime.state.route), "", url);
    } catch (_) {
      try {
        window.history.replaceState(historyState(runtime.state.route), "", currentPlatformRouteKey());
      } catch (_) {}
    }
  }

  function historyState(route) {
    return { nova: true, novaRoute: routeObject(route) };
  }

  function currentPlatformRouteKey() {
    return routeURL(routeObjectFromLocation(window.location));
  }

  function isExplicitPlatformRoute(location) {
    const route = routeObjectFromLocation(location);
    return route.path !== "/" || Object.keys(route.query || {}).length > 0 || !!route.fragment;
  }

  function routeObjectFromLocation(location) {
    return {
      path: normalizePath(location.pathname || "/"),
      query: queryFromSearch(location.search || ""),
      fragment: location.hash ? decodeURIComponent(location.hash.slice(1)) : ""
    };
  }

  function routeValueForShape(route, shape) {
    if (typeof shape === "string") return route.path;
    const next = shape && typeof shape === "object" && !Array.isArray(shape) ? { ...shape } : {};
    next.path = route.path;
    if (Object.keys(route.query || {}).length) next.query = route.query;
    else delete next.query;
    if (route.fragment) next.fragment = route.fragment;
    else delete next.fragment;
    return next;
  }

  function routeObject(value) {
    if (typeof value === "string") return { path: normalizePath(value), query: {}, fragment: "" };
    if (value && typeof value === "object" && !Array.isArray(value)) {
      return {
        ...value,
        path: normalizePath(value.path || "/"),
        query: value.query && typeof value.query === "object" ? value.query : {},
        fragment: value.fragment ? String(value.fragment).replace(/^#/, "") : ""
      };
    }
    return { path: "/", query: {}, fragment: "" };
  }

  function queryFromSearch(search) {
    const query = {};
    const params = new URLSearchParams(search || "");
    for (const [key, value] of params.entries()) {
      query[key] = value;
    }
    return query;
  }

  function searchFromQuery(query) {
    if (!query || typeof query !== "object") return "";
    const params = new URLSearchParams();
    for (const key of Object.keys(query).sort()) {
      const value = query[key];
      if (value === undefined || value === null) continue;
      if (Array.isArray(value)) {
        value.forEach((item) => params.append(key, String(item)));
      } else {
        params.append(key, String(value));
      }
    }
    const text = params.toString();
    return text ? "?" + text : "";
  }

  function routeURL(route) {
    const fragment = route.fragment ? "#" + encodeURIComponent(String(route.fragment).replace(/^#/, "")) : "";
    return normalizePath(route.path) + searchFromQuery(route.query) + fragment;
  }

  function normalizePath(value) {
    const path = String(value || "/");
    return path.startsWith("/") ? path : "/" + path;
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
    setupNavigation(runtime);
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
