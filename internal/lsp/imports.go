package lsp

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/parser"
)

func findProjectRootForURI(documentURI string) string {
	modulePath := modulePathFromURI(documentURI)
	if modulePath == "" {
		return ""
	}
	root, ok := findProjectRoot(filepath.Dir(modulePath))
	if !ok {
		return ""
	}
	return root
}

func modulePathFromURI(uri string) string {
	path := strings.TrimSpace(uri)
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "file://") {
		path = strings.TrimPrefix(path, "file://")
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(path)
}

func findProjectRoot(start string) (string, bool) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "nova.toml")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func loadProjectNovaFiles(projectRoot string) map[string]parser.File {
	modules := make(map[string]parser.File)
	srcRoot := filepath.Join(projectRoot, "src")
	_ = filepath.WalkDir(srcRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(path) != ".nova" {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		tokens := lexer.Tokenize(string(content))
		file, parseDiagnostics := parser.Parse(tokens)
		if len(parseDiagnostics) > 0 {
			return nil
		}
		modules[filepath.ToSlash(path)] = file
		return nil
	})
	return modules
}
