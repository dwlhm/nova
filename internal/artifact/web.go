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

  function schedulerHost(runtime) {
    return {
      app: runtime.app,
      state: () => runtime.state,
      commitState: (state) => { runtime.state = state; },
      evaluate,
      cloneState,
      pick,
      stateInvalidations: (beforeState, afterState) => stateInvalidations(runtime, beforeState, afterState),
      hasRouteState: () => hasRouteState(runtime),
      routeKey: () => routeKey(runtime),
      routeObject,
      routeValueForShape: (route, shape) => routeValueForShape(route, shape, runtime),
      update: (invalidations) => update(runtime, invalidations),
      reconcileNavigation: (beforeRoute, options) => reconcileNavigation(runtime, beforeRoute, options)
    };
  }

  function dispatch(runtime, event, args, options) {
    runtime.scheduler.dispatch(event, args || [], options || {});
  }

  function render(runtime) {
    const root = runtime.root;
    runtime.refs = new Map();
    root.replaceChildren(...renderNodes(pick(runtime.app.viewIR, "nodes", "Nodes", []), runtime, []));
    update(runtime, null);
  }

  function update(runtime, invalidations) {
    if (!runtime.refs) {
      render(runtime);
      return;
    }
    if (invalidations && !invalidations.size) return;
    updateBindings(runtime, invalidations);
    updatePages(runtime, pick(runtime.app.viewIR, "nodes", "Nodes", []), []);
  }

  function renderNodes(nodes, runtime, parentPath) {
    const list = nodes || [];
    const selectedPages = selectedPageNodes(list, runtime);
    return list.map((node, index) => {
      return renderNode(node, runtime, parentPath.concat(index), selectedPages.has(index));
    }).filter(Boolean);
  }

  function selectedPageNodes(nodes, runtime) {
    const selected = new Set();
    let bestScore = null;
    (nodes || []).forEach((node, index) => {
      if (nodeKind(node) !== "page") return;
      const props = pick(node, "props", "Props", {}) || {};
      const pattern = pagePattern(props, runtime);
      const match = routeMatch(pattern, activeRoutePath(runtime));
      if (!match.matched) return;
      if (bestScore === null || match.score > bestScore) {
        selected.clear();
        selected.add(index);
        bestScore = match.score;
        return;
      }
      if (match.score === bestScore) selected.add(index);
    });
    return selected;
  }

  function updateBindings(runtime, invalidations) {
    for (const ref of metadataBindings(runtime.app)) {
      const states = pick(ref, "states", "States", []) || [];
      const prop = pick(ref, "prop", "Prop", "");
      if (prop === "key" || prop.includes("#arg")) continue;
      if (!shouldApplyBinding(states, invalidations)) continue;
      const path = pick(ref, "nodePath", "NodePath", []) || [];
      const target = runtime.refs.get(pathKey(path));
      if (!target) continue;
      const node = nodeAtPath(runtime.app, path);
      if (!node) continue;
      updateNodeBinding(target, node, prop, runtime);
    }
  }

  function updatePages(runtime, nodes, parentPath) {
    const list = nodes || [];
    const selectedPages = selectedPageNodes(list, runtime);
    list.forEach((node, index) => {
      const path = parentPath.concat(index);
      if (nodeKind(node) === "page") {
        const target = runtime.refs.get(pathKey(path));
        if (target) target.hidden = !selectedPages.has(index);
      }
      updatePages(runtime, pick(node, "children", "Children", []) || [], path);
    });
  }

  function updateNodeBinding(target, node, prop, runtime) {
    const props = pick(node, "props", "Props", {}) || {};
    const binding = props[prop];
    if (!binding) return;
    const value = evaluate(bindingExpression(binding, runtime.app), runtime.state, {});
    if (target.nodeType === Node.TEXT_NODE) {
      target.textContent = String(value);
      return;
    }
    if (prop === "value" && nodeKind(node) === "text") {
      target.textContent = String(value);
      return;
    }
    applyPropValue(target, prop, value);
  }

  function metadataBindings(app) {
    const viewIR = pick(app, "viewIR", "ViewIR", {}) || {};
    const metadata = pick(viewIR, "metadata", "Metadata", {}) || {};
    return pick(metadata, "bindings", "Bindings", []) || [];
  }

  function shouldApplyBinding(states, invalidations) {
    if (!invalidations) return true;
    return (states || []).some((state) => invalidations.has(state));
  }

  function nodeAtPath(app, path) {
    let list = pick(pick(app, "viewIR", "ViewIR", {}) || {}, "nodes", "Nodes", []) || [];
    let node = null;
    for (const index of path || []) {
      node = list[index];
      if (!node) return null;
      list = pick(node, "children", "Children", []) || [];
    }
    return node;
  }

  function pathKey(path) {
    return (path || []).join(".");
  }

  function cloneState(state) {
    return { ...(state || {}) };
  }

  function stateInvalidations(runtime, beforeState, afterState) {
    const invalidations = new Set();
    const state = afterState || runtime.state;
    for (const cell of (runtime.app.model || {}).states || []) {
      if (!sameValue(beforeState[cell.name], state[cell.name])) {
        invalidations.add(cell.name);
      }
    }
    return invalidations;
  }

  function sameValue(left, right) {
    if (Object.is(left, right)) return true;
    try {
      return JSON.stringify(left) === JSON.stringify(right);
    } catch (_) {
      return false;
    }
  }

  function nodeKind(node) {
    return pick(node, "kind", "Kind", "div");
  }

  function renderNode(node, runtime, path, pageActive) {
    const kind = pick(node, "kind", "Kind", "div");
    const props = pick(node, "props", "Props", {}) || {};
    const events = pick(node, "events", "Events", {}) || {};
    const children = pick(node, "children", "Children", []) || [];
    if (kind === "page") {
      return renderPage(node, props, children, runtime, path, pageActive);
    }
    if (kind === "#text") {
      const text = document.createTextNode("");
      runtime.refs.set(pathKey(path), text);
      updateNodeBinding(text, node, "value", runtime);
      return text;
    }
    const element = document.createElement(tagFor(kind));
    runtime.refs.set(pathKey(path), element);
    element.dataset.novaKind = kind;
    applyProps(element, props, runtime);
    applyEvents(element, events, runtime);
    element.append(...renderNodes(children, runtime, path));
    if (kind === "text" && !children.length && props.value) {
      updateNodeBinding(element, node, "value", runtime);
    }
    return element;
  }

  function renderPage(node, props, children, runtime, path, active) {
    const element = document.createElement("div");
    runtime.refs.set(pathKey(path), element);
    element.dataset.novaKind = "page";
    element.hidden = !active;
    element.append(...renderNodes(children, runtime, path));
    return element;
  }

  function pagePattern(props, runtime) {
    return normalizeRoutePattern(evaluate(bindingExpression(props.path, runtime.app), runtime.state, {}));
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

  function reconcileRouteState(runtime) {
    if (!hasRouteState(runtime)) return;
    runtime.state.route = routeValueForShape(routeObject(runtime.state.route), runtime.state.route, runtime);
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
      runtime.state.route = routeValueForShape(platformRoute, runtime.state.route, runtime);
    }
    reconcileRouteState(runtime);
    writeHistoryState(runtime, "replace");
    window.addEventListener("popstate", (event) => {
      const route = event && event.state && event.state.novaRoute
        ? routeObject(event.state.novaRoute)
        : routeObjectFromLocation(window.location);
      dispatch(runtime, ROUTE_CHANGED_EVENT, [routeValueForShape(route, runtime.state.route, runtime)], { source: "platform" });
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

  function routeValueForShape(route, shape, runtime) {
    if (typeof shape === "string") return route.path;
    const next = shape && typeof shape === "object" && !Array.isArray(shape) ? { ...shape } : {};
    next.path = route.path;
    if (Object.keys(route.query || {}).length) next.query = route.query;
    else delete next.query;
    if (route.fragment) next.fragment = route.fragment;
    else delete next.fragment;
    const match = runtime ? bestRouteMatch(collectPagePatterns(runtime), route.path) : { params: {} };
    if (match && match.params && Object.keys(match.params).length) next.params = match.params;
    else delete next.params;
    return next;
  }

  function routeObject(value) {
    if (typeof value === "string") return routeObjectFromString(value);
    if (value && typeof value === "object" && !Array.isArray(value)) {
      const parsed = routeObjectFromString(value.path || "/");
      return {
        ...value,
        path: parsed.path,
        query: value.query && typeof value.query === "object" ? value.query : parsed.query,
        fragment: value.fragment ? String(value.fragment).replace(/^#/, "") : parsed.fragment,
        params: value.params && typeof value.params === "object" ? value.params : {}
      };
    }
    return { path: "/", query: {}, fragment: "" };
  }

  function routeObjectFromString(value) {
    const text = String(value || "/").trim() || "/";
    try {
      if (/^[a-z][a-z0-9+.-]*:\/\//i.test(text)) {
        const parsed = new URL(text);
        return {
          path: normalizePath(parsed.pathname || "/"),
          query: queryFromSearch(parsed.search || ""),
          fragment: parsed.hash ? safeDecodeURIComponent(parsed.hash.slice(1)) : ""
        };
      }
    } catch (_) {}
    const hashIndex = text.indexOf("#");
    const beforeHash = hashIndex >= 0 ? text.slice(0, hashIndex) : text;
    const fragment = hashIndex >= 0 ? safeDecodeURIComponent(text.slice(hashIndex + 1)) : "";
    const queryIndex = beforeHash.indexOf("?");
    const rawPath = queryIndex >= 0 ? beforeHash.slice(0, queryIndex) : beforeHash;
    const rawQuery = queryIndex >= 0 ? beforeHash.slice(queryIndex) : "";
    return { path: normalizePath(rawPath || "/"), query: queryFromSearch(rawQuery), fragment };
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
    return encodeRoutePath(normalizePath(route.path)) + searchFromQuery(route.query) + fragment;
  }

  function normalizePath(value) {
    let path = String(value || "/").trim();
    const hash = path.indexOf("#");
    if (hash >= 0) path = path.slice(0, hash);
    const query = path.indexOf("?");
    if (query >= 0) path = path.slice(0, query);
    if (!path.startsWith("/")) path = "/" + path;
    const segments = [];
    for (const raw of path.split("/")) {
      if (!raw || raw === ".") continue;
      if (raw === "..") {
        segments.pop();
        continue;
      }
      segments.push(safeDecodeURIComponent(raw));
    }
    return segments.length ? "/" + segments.join("/") : "/";
  }

  function normalizeRoutePattern(value) {
    const pattern = String(value || "/").trim();
    if (pattern === "*" || pattern === "/*") return "*";
    if (pattern.endsWith("/*")) return normalizePath(pattern.slice(0, -2)) + "/*";
    return normalizePath(pattern);
  }

  function collectPagePatterns(runtime) {
    const patterns = [];
    collectPagePatternsFromNodes(pick(runtime.app.viewIR, "nodes", "Nodes", []), runtime, patterns);
    return patterns;
  }

  function collectPagePatternsFromNodes(nodes, runtime, patterns) {
    for (const node of nodes || []) {
      const props = pick(node, "props", "Props", {}) || {};
      if (nodeKind(node) === "page") patterns.push(pagePattern(props, runtime));
      collectPagePatternsFromNodes(pick(node, "children", "Children", []) || [], runtime, patterns);
    }
  }

  function bestRouteMatch(patterns, path) {
    let best = null;
    for (const pattern of patterns || []) {
      const match = routeMatch(pattern, path);
      if (!match.matched) continue;
      if (!best || match.score > best.score) best = match;
    }
    return best || { matched: false, params: {}, score: -1 };
  }

  function routeMatch(pattern, path) {
    const normalizedPattern = normalizeRoutePattern(pattern);
    const normalizedPath = normalizePath(path);
    const params = {};
    if (normalizedPattern === "*") {
      return { matched: true, pattern: normalizedPattern, path: normalizedPath, params, score: 0, fallback: true };
    }
    const patternSegments = routeSegments(normalizedPattern);
    const pathSegments = routeSegments(normalizedPath);
    let score = 0;
    for (let i = 0; i < patternSegments.length; i++) {
      const segment = patternSegments[i];
      if (segment === "*") {
        return i === patternSegments.length - 1
          ? { matched: true, pattern: normalizedPattern, path: normalizedPath, params, score: score + 10, fallback: false }
          : { matched: false, pattern: normalizedPattern, path: normalizedPath, params, score: -1, fallback: false };
      }
      if (i >= pathSegments.length) return { matched: false, pattern: normalizedPattern, path: normalizedPath, params, score: -1, fallback: false };
      const name = dynamicSegmentName(segment);
      if (name) {
        params[name] = pathSegments[i];
        score += 50;
        continue;
      }
      if (segment !== pathSegments[i]) return { matched: false, pattern: normalizedPattern, path: normalizedPath, params, score: -1, fallback: false };
      score += 100;
    }
    if (patternSegments.length !== pathSegments.length) {
      return { matched: false, pattern: normalizedPattern, path: normalizedPath, params, score: -1, fallback: false };
    }
    return { matched: true, pattern: normalizedPattern, path: normalizedPath, params, score: score + 1000, fallback: false };
  }

  function routeSegments(value) {
    const path = normalizePath(value);
    return path === "/" ? [] : path.slice(1).split("/");
  }

  function dynamicSegmentName(segment) {
    if (segment.startsWith(":") && segment.length > 1) return segment.slice(1);
    if (segment.startsWith("{") && segment.endsWith("}") && segment.length > 2) return segment.slice(1, -1);
    return "";
  }

  function encodeRoutePath(path) {
    const normalized = normalizePath(path);
    if (normalized === "/") return "/";
    return "/" + normalized.slice(1).split("/").map(encodeURIComponent).join("/");
  }

  function safeDecodeURIComponent(value) {
    try {
      return decodeURIComponent(value);
    } catch (_) {
      return value;
    }
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
      applyPropValue(element, name, value);
    }
  }

  function applyPropValue(element, name, value) {
    if (name === "class") {
      element.className = String(value || "");
      return;
    }
    if (name === "label") {
      element.setAttribute("aria-label", String(value));
      return;
    }
    if (name === "enabled") {
      if (value === false) element.setAttribute("disabled", "");
      else element.removeAttribute("disabled");
      return;
    }
    element.setAttribute(name.replaceAll("_", "-"), String(value));
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
    if (!window.NovaScheduler) {
      throw new Error("NovaScheduler module is required before NovaRuntime");
    }
    const runtime = { app, root, state: initialState(app) };
    runtime.scheduler = window.NovaScheduler.create(schedulerHost(runtime));
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
