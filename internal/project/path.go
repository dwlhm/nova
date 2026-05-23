package project

import (
	"path"
	"strings"
)

func normalizePath(filePath string) string {
	filePath = strings.ReplaceAll(filePath, "\\", "/")
	return path.Clean(filePath)
}
