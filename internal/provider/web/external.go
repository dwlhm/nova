package web

import (
	"sort"
	"strings"

	externaljs "github.com/dwlhm/nova/runtime/nova-external-js"
	"github.com/dwlhm/nova/internal/provider/build"
)

type externalAdapterBundle struct {
	Enabled bool
	Content string
}

func externalAdapters(operations []build.ResolvedExternalOperation, overrides map[string]string) externalAdapterBundle {
	if len(operations) == 0 {
		return externalAdapterBundle{}
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
		return externalAdapterBundle{}
	}
	var builder strings.Builder
	builder.WriteString(`"use strict";

`)
	builder.WriteString(externaljs.CoreJS())
	builder.WriteString(`

window.NovaExternal = window.NovaExternal || NovaExternalCore.create();

`)
	for _, path := range paths {
		content, ok := adapterContent(path, overrides)
		if !ok || strings.TrimSpace(content) == "" {
			continue
		}
		builder.WriteString("\n// " + path + "\n")
		builder.WriteString("(function(NovaExternal) {\n")
		builder.WriteString(rewriteRegisterModule(content))
		builder.WriteString("\nif (typeof register === \"function\") register(NovaExternal);\n")
		builder.WriteString("})(window.NovaExternal);\n")
	}
	body := builder.String()
	if !strings.Contains(body, "register(NovaExternal)") {
		return externalAdapterBundle{}
	}
	return externalAdapterBundle{Enabled: true, Content: body}
}
