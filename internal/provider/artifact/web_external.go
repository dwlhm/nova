package artifact

import (
	"sort"
	"strings"

	"github.com/dwlhm/nova/internal/provider/build"
)

type webExternalAdapterBundle struct {
	Enabled bool
	Content string
}

func webExternalAdapters(operations []build.ResolvedExternalOperation, overrides map[string]string) webExternalAdapterBundle {
	if len(operations) == 0 {
		return webExternalAdapterBundle{}
	}
	paths := make([]string, 0, len(operations))
	seen := make(map[string]bool)
	for _, operation := range operations {
		path := strings.TrimSpace(operation.Implementation.Path)
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		paths = append(paths, path)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return webExternalAdapterBundle{}
	}
	var builder strings.Builder
	builder.WriteString(`"use strict";

window.NovaExternal = window.NovaExternal || (() => {
  function isSerializable(value, seen) {
    if (value === undefined || value === null) return true;
    const type = typeof value;
    if (type === "string" || type === "boolean") return true;
    if (type === "number") return Number.isFinite(value);
    if (type !== "object") return false;
    const visited = seen || new WeakSet();
    if (visited.has(value)) return false;
    visited.add(value);
    let ok = false;
    if (Array.isArray(value)) {
      ok = value.every((item) => isSerializable(item, visited));
    } else {
      ok = Object.keys(value).every((key) => isSerializable(value[key], visited));
    }
    visited.delete(value);
    return ok;
  }

  const adapters = new Map();
  return {
    define(source, adapter) {
      adapters.set(source, adapter || {});
    },
    async invoke(effectId, input) {
      const separator = effectId.indexOf("#");
      if (separator <= 0) throw new Error("invalid effect id " + effectId);
      const source = effectId.slice(0, separator);
      const operation = effectId.slice(separator + 1);
      const adapter = adapters.get(source);
      if (!adapter) throw new Error("missing external adapter for " + source);
      const handler = adapter[operation];
      if (typeof handler !== "function") throw new Error(source + " has no operation " + operation);
      const output = await handler({ input: input || {} });
      if (!isSerializable(output)) {
        throw new Error("external operation " + effectId + " returned non-serializable output");
      }
      return output;
    }
  };
})();

`)
	for _, path := range paths {
		content, ok := webAdapterContent(path, overrides)
		if !ok || strings.TrimSpace(content) == "" {
			continue
		}
		builder.WriteString("\n// " + path + "\n")
		builder.WriteString("(function(NovaExternal) {\n")
		builder.WriteString(rewriteWebRegisterModule(content))
		builder.WriteString("\nif (typeof register === \"function\") register(NovaExternal);\n")
		builder.WriteString("})(window.NovaExternal);\n")
	}
	body := builder.String()
	if !strings.Contains(body, "register(NovaExternal)") {
		return webExternalAdapterBundle{}
	}
	return webExternalAdapterBundle{Enabled: true, Content: body}
}
