package web

import (
	"strings"

	"github.com/dwlhm/nova/internal/provider/shared"
)

type styleBundleResult struct {
	Files     []shared.File
	Manifest  map[string]any
	RootScope string
}

type styleManifestAsset struct {
	SourcePath string `json:"sourcePath"`
	OutputPath string `json:"outputPath"`
	Scope      string `json:"scope"`
	Order      int    `json:"order"`
}

func styleBundle(styles []shared.StyleAsset) styleBundleResult {
	files := make([]shared.File, 0, len(styles))
	manifestAssets := make([]styleManifestAsset, 0, len(styles))
	seen := make(map[string]bool, len(styles))
	hasAppScope := false
	for _, style := range styles {
		outputPath := styleOutputPath(style.SourcePath)
		if outputPath == "" || seen[outputPath] {
			continue
		}
		seen[outputPath] = true
		scope := shared.NormalizeStyleScope(style.Scope)
		if scope == shared.StyleScopeApp {
			hasAppScope = true
		}
		files = append(files, shared.File{Path: outputPath, Content: styleContent(style.Content, scope)})
		manifestAssets = append(manifestAssets, styleManifestAsset{
			SourcePath: style.SourcePath,
			OutputPath: strings.TrimPrefix(outputPath, "build/web/"),
			Scope:      string(scope),
			Order:      len(manifestAssets),
		})
	}
	rootScope := ""
	if hasAppScope {
		rootScope = string(shared.StyleScopeApp)
	}
	return styleBundleResult{
		Files: files,
		Manifest: map[string]any{
			"assets": manifestAssets,
		},
		RootScope: rootScope,
	}
}

func styleContent(content string, scope shared.StyleScope) string {
	if scope != shared.StyleScopeApp {
		return content
	}
	return shared.ScopeCSS(content, `#nova-root[data-nova-style-scope~="app"]`)
}

func styleHrefs(files []shared.File) []string {
	hrefs := make([]string, 0, len(files))
	for _, file := range files {
		if href, ok := strings.CutPrefix(file.Path, "build/web/"); ok {
			hrefs = append(hrefs, href)
		}
	}
	return hrefs
}

func styleOutputPath(sourcePath string) string {
	clean := strings.Trim(strings.ReplaceAll(sourcePath, "\\", "/"), "/")
	if clean == "" || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return ""
	}
	return "build/web/assets/styles/" + clean
}
