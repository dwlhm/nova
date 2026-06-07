package web

import (
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/provider/build"
)

func TestExternalAdaptersBundleKnownSources(t *testing.T) {
	bundle := externalAdapters([]build.ResolvedExternalOperation{
		{CapabilitySource: "@env/storage", Operation: "set", Implementation: build.Implementation{Path: "platform/web/storage.web.js"}},
		{CapabilitySource: "@env/network", Operation: "request", Implementation: build.Implementation{Path: "platform/web/network.web.js"}},
	}, nil)
	if !bundle.Enabled {
		t.Fatal("expected external adapter bundle")
	}
	for _, token := range []string{"NovaExternal", "NovaExternalCore", "@env/storage", "@env/network", "checkPermission", "isSerializable"} {
		if !strings.Contains(bundle.Content, token) {
			t.Fatalf("bundle missing %q", token)
		}
	}
}
