package schedulerjs

import (
	"strings"
	"testing"
)

func TestEmbeddedSourceExportsNovaScheduler(t *testing.T) {
	if !strings.Contains(Source, "window.NovaScheduler") {
		t.Fatal("embedded scheduler source missing NovaScheduler export")
	}
	if Version != "0.1.0" {
		t.Fatalf("unexpected scheduler version %s", Version)
	}
}
