package ast

import (
	"path"
	"strings"

	"github.com/dwlhm/nova/internal/core/parser"
)

// RawFile is the syntax tree produced by the parser before semantic analysis.
type RawFile = parser.File

// RawModule is a parsed Nova source module (raw AST).
type RawModule struct {
	Path string
	File RawFile
}

// CheckedFile is a raw AST that passed semantic analysis.
type CheckedFile struct {
	File RawFile
}

// CheckedModule is a source module after semantic analysis.
type CheckedModule struct {
	Path string
	File CheckedFile
}

// FileMap returns checked modules keyed by normalized path.
func FileMap(modules []CheckedModule) map[string]RawFile {
	out := make(map[string]RawFile, len(modules))
	for _, module := range modules {
		out[NormalizePath(module.Path)] = module.File.File
	}
	return out
}

func NormalizePath(filePath string) string {
	filePath = strings.ReplaceAll(filePath, "\\", "/")
	return path.Clean(filePath)
}
