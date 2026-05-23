package schedulerjava

import (
	"strings"
	"testing"
)

func TestSourceFilesIncludesSchedulerTypes(t *testing.T) {
	files, err := SourceFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 3 {
		t.Fatalf("expected scheduler sources, got %d files", len(files))
	}
	joined := make([]string, 0, len(files))
	for _, file := range files {
		joined = append(joined, file.RelativePath)
		if !strings.HasPrefix(file.Content, "package nova.scheduler;") {
			t.Fatalf("unexpected package header in %s", file.RelativePath)
		}
	}
	if !strings.Contains(strings.Join(joined, ","), "NovaScheduler.java") {
		t.Fatalf("missing NovaScheduler.java in %v", joined)
	}
}
