package style

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExampleNovaStyleFilesParseClean(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples")
	matches, err := filepath.Glob(filepath.Join(root, "*", "src", "*.nova-style"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range matches {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		_, diagnostics := ParseDocument(path, string(content))
		for _, diagnostic := range diagnostics {
			if diagnostic.Code == "NVA-STYLE-010" || diagnostic.Code == "NVA-STYLE-009" {
				t.Errorf("%s: %s", path, diagnostic.Message)
			}
		}
	}
	if len(matches) == 0 {
		t.Fatal("expected example .nova-style files")
	}
}
