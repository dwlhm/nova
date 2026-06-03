package style

import (
	"strings"

	"github.com/dwlhm/nova/internal/core/style"
	"github.com/dwlhm/nova/internal/provider/shared"
)

type bundleResult struct {
	Files     []shared.File
	Manifest  map[string]any
	RootScope string
}

type manifestAsset struct {
	SourcePath string `json:"sourcePath"`
	OutputPath string `json:"outputPath"`
	Scope      string `json:"scope"`
	Order      int    `json:"order"`
}

func webStyleBundle(bundle style.Bundle) bundleResult {
	files := make([]shared.File, 0, len(bundle.Sheets)+len(bundle.WebStylesheets))
	manifestAssets := make([]manifestAsset, 0, len(bundle.Sheets)+len(bundle.WebStylesheets))
	seen := make(map[string]bool)
	hasAppScope := false
	order := 0

	appendAsset := func(sourcePath, content string, scope style.Scope) {
		outputPath := styleOutputPath(sourcePath)
		if outputPath == "" || seen[outputPath] {
			return
		}
		seen[outputPath] = true
		scope = style.NormalizeScope(scope)
		if scope == style.ScopeApp {
			hasAppScope = true
		}
		files = append(files, shared.File{Path: outputPath, Content: styleContent(content, scope)})
		manifestAssets = append(manifestAssets, manifestAsset{
			SourcePath: sourcePath,
			OutputPath: strings.TrimPrefix(outputPath, "build/web/"),
			Scope:      string(scope),
			Order:      order,
		})
		order++
	}

	for _, sheet := range bundle.Sheets {
		appendAsset(sheet.SourcePath, style.EmitCSS(sheet), sheet.Scope)
	}
	for _, sheet := range bundle.WebStylesheets {
		appendAsset(sheet.SourcePath, sheet.Content, sheet.Scope)
	}

	rootScope := ""
	if hasAppScope {
		rootScope = string(style.ScopeApp)
	}
	return bundleResult{
		Files: files,
		Manifest: map[string]any{
			"assets": manifestAssets,
		},
		RootScope: rootScope,
	}
}

func styleContent(content string, scope style.Scope) string {
	if scope != style.ScopeApp {
		return content
	}
	return shared.ScopeCSS(content, `#nova-root[data-nova-style-scope~="app"]`)
}

func styleOutputPath(sourcePath string) string {
	clean := strings.Trim(strings.ReplaceAll(sourcePath, "\\", "/"), "/")
	if clean == "" || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return ""
	}
	return "build/web/assets/styles/" + clean
}

func Bundle(bundle style.Bundle) bundleResult {
	return webStyleBundle(bundle)
}

func StyleHrefs(files []shared.File) []string {
	hrefs := make([]string, 0, len(files))
	for _, file := range files {
		if href, ok := strings.CutPrefix(file.Path, "build/web/"); ok {
			hrefs = append(hrefs, href)
		}
	}
	return hrefs
}
