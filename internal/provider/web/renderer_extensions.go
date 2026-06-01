package web

import (
	"strings"

	"github.com/dwlhm/nova/internal/provider/build"
)

type rendererExtensionBundle struct {
	Enabled bool
	Content string
}

func rendererExtensions(extensions []build.RendererExtension) rendererExtensionBundle {
	if len(extensions) == 0 {
		return rendererExtensionBundle{}
	}
	var builder strings.Builder
	builder.WriteString("\"use strict\";\n")
	for _, extension := range extensions {
		if strings.TrimSpace(extension.AdapterContent) == "" {
			continue
		}
		builder.WriteString("\n// " + extension.Package + " " + extension.AdapterPath + "\n")
		builder.WriteString("(function(NovaRenderer) {\n")
		builder.WriteString(rewriteRegisterModule(extension.AdapterContent))
		builder.WriteString("\nif (typeof register === \"function\") register(NovaRenderer);\n")
		builder.WriteString("})(window.NovaRenderer);\n")
	}
	return rendererExtensionBundle{Enabled: true, Content: builder.String()}
}

func rewriteRegisterModule(content string) string {
	content = strings.ReplaceAll(content, "export function register", "function register")
	content = strings.ReplaceAll(content, "export default function register", "function register")
	return content
}
