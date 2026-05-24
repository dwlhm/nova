package rendererjs

import (
	"strings"
	"testing"
)

func TestEmbeddedSourceExportsNovaRenderer(t *testing.T) {
	if !strings.Contains(Source, "NovaRenderer") {
		t.Fatal("embedded renderer source missing NovaRenderer export")
	}
	if Version != "0.1.0" {
		t.Fatalf("unexpected renderer version %s", Version)
	}
}
