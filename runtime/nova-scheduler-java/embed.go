package schedulerjava

import (
	"embed"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// Version matches Nova artifact metadata schedulerVersion.
const Version = "0.1.0"

// PackagePath is the Java package directory embedded into Android artifacts.
const PackagePath = "nova/scheduler"

//go:embed all:src/main/java
var sources embed.FS

// SourceFile is one Java source file from the scheduler library.
type SourceFile struct {
	RelativePath string
	Content      string
}

// SourceFiles returns scheduler library sources sorted by path.
func SourceFiles() ([]SourceFile, error) {
	root := "src/main/java/" + PackagePath
	entries, err := fs.ReadDir(sources, root)
	if err != nil {
		return nil, err
	}
	files := make([]SourceFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".java") {
			continue
		}
		content, err := fs.ReadFile(sources, path.Join(root, entry.Name()))
		if err != nil {
			return nil, err
		}
		files = append(files, SourceFile{
			RelativePath: entry.Name(),
			Content:      string(content),
		})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].RelativePath < files[j].RelativePath
	})
	return files, nil
}
