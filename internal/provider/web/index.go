package web

import (
	"strings"

	"github.com/dwlhm/nova/internal/provider/shared"
)

func indexHTML(name string, styleHrefs []string, rootStyleScope string, rendererExtensions bool, externalAdapters bool) string {
	if strings.TrimSpace(name) == "" {
		name = "Nova App"
	}
	links := "  <link rel=\"stylesheet\" href=\"assets/nova-runtime.css\">\n"
	for _, href := range styleHrefs {
		links += "  <link rel=\"stylesheet\" href=\"" + shared.EscapeHTML(href) + "\">\n"
	}
	scopeAttr := ""
	if strings.TrimSpace(rootStyleScope) != "" {
		scopeAttr = " data-nova-style-scope=\"" + shared.EscapeHTML(rootStyleScope) + "\""
	}
	extensionScript := ""
	if rendererExtensions {
		extensionScript = "  <script src=\"assets/renderer-extensions.js\"></script>\n"
	}
	externalScript := ""
	if externalAdapters {
		externalScript = "  <script src=\"assets/external-adapters.js\"></script>\n"
	}
	return "<!doctype html>\n<html lang=\"en\">\n<head>\n  <meta charset=\"utf-8\">\n  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n  <title>" + shared.EscapeHTML(name) + "</title>\n" + links + "</head>\n<body>\n  <main id=\"nova-root\"" + scopeAttr + " aria-label=\"" + shared.EscapeHTML(name) + "\"></main>\n  <script src=\"assets/nova-scheduler.js\"></script>\n  <script src=\"assets/nova-renderer.js\"></script>\n" + extensionScript + externalScript + "  <script src=\"assets/nova-app-lifecycle.js\"></script>\n  <script src=\"assets/nova-runtime.js\"></script>\n  <script src=\"app.bundle.js\"></script>\n</body>\n</html>\n"
}
