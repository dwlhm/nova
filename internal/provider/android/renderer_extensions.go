package android

import (
	"strings"

	"github.com/dwlhm/nova/internal/provider/build"
	"github.com/dwlhm/nova/internal/provider/shared"
)

func rendererAdapterFiles(extensions []build.RendererExtension) []shared.File {
	files := make([]shared.File, 0, len(extensions))
	for _, extension := range extensions {
		if strings.TrimSpace(extension.AdapterContent) == "" {
			continue
		}
		files = append(files, shared.File{
			Path:    "build/android/renderer/" + shared.SafeRendererPackagePath(extension.Package) + "/" + shared.SafeRendererAdapterName(extension.AdapterPath),
			Content: extension.AdapterContent,
		})
	}
	return files
}
