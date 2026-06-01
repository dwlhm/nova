package artifact

import (
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/provider/build"
)

func TestWebExternalAdaptersBundleKnownSources(t *testing.T) {
	bundle := webExternalAdapters([]build.ResolvedExternalOperation{
		{CapabilitySource: "@env/storage", Operation: "set", Implementation: build.Implementation{Path: "platform/web/storage.web.js"}},
		{CapabilitySource: "@env/network", Operation: "request", Implementation: build.Implementation{Path: "platform/web/network.web.js"}},
	}, nil)
	if !bundle.Enabled {
		t.Fatal("expected external adapter bundle")
	}
	for _, token := range []string{"NovaExternal", "@env/storage", "@env/network", "isSerializable"} {
		if !strings.Contains(bundle.Content, token) {
			t.Fatalf("bundle missing %q", token)
		}
	}
}

func TestAndroidExternalAdapterFiles(t *testing.T) {
	operations := []build.ResolvedExternalOperation{
		{CapabilitySource: "@env/storage", Operation: "set", Implementation: build.Implementation{Path: "platform/android/storage.android.java"}},
		{CapabilitySource: "@env/network", Operation: "request", Implementation: build.Implementation{Path: "platform/android/network.android.java"}},
		{CapabilitySource: "@env/clipboard", Operation: "write", Implementation: build.Implementation{Path: "platform/android/clipboard.android.java"}},
		{CapabilitySource: "@env/notify", Operation: "send", Implementation: build.Implementation{Path: "platform/android/notify.android.java"}},
		{CapabilitySource: "@env/device", Operation: "info", Implementation: build.Implementation{Path: "platform/android/device.android.java"}},
	}
	javaRoot := "build/android/app/src/main/java/com/example/app"
	files := androidExternalAdapterFiles(operations, javaRoot, nil)
	if len(files) != 6 {
		t.Fatalf("files = %d, want adapter sources + dispatcher", len(files))
	}
	assertArtifactFile(t, files, javaRoot+"/NovaExternalAdapters.java", "NovaExternalAdapters")
	assertArtifactFile(t, files, "build/android/app/src/main/java/com/nova/env/StorageAdapter.java", "StorageAdapter")
	assertArtifactFile(t, files, "build/android/app/src/main/java/com/nova/env/NetworkAdapter.java", "NetworkAdapter")
}
