package android

import (
	"strings"
	"testing"

	"github.com/dwlhm/nova/internal/provider/build"
	"github.com/dwlhm/nova/internal/provider/shared"
)

func TestExternalAdapterFiles(t *testing.T) {
	operations := []build.ResolvedExternalOperation{
		{CapabilitySource: "@env/storage", Operation: "set", Implementation: build.Implementation{Path: "platform/android/storage.android.java"}},
		{CapabilitySource: "@env/network", Operation: "request", Implementation: build.Implementation{Path: "platform/android/network.android.java"}},
		{CapabilitySource: "@env/clipboard", Operation: "write", Implementation: build.Implementation{Path: "platform/android/clipboard.android.java"}},
		{CapabilitySource: "@env/notify", Operation: "send", Implementation: build.Implementation{Path: "platform/android/notify.android.java"}},
		{CapabilitySource: "@env/device", Operation: "info", Implementation: build.Implementation{Path: "platform/android/device.android.java"}},
	}
	javaRoot := "build/android/app/src/main/java/com/example/app"
	files := externalAdapterFiles(operations, javaRoot, nil)
	if len(files) != 6 {
		t.Fatalf("files = %d, want adapter sources + dispatcher", len(files))
	}
	assertFile(t, files, javaRoot+"/NovaExternalAdapters.java", "NovaExternalAdapters")
	assertFile(t, files, "build/android/app/src/main/java/com/nova/env/StorageAdapter.java", "StorageAdapter")
	assertFile(t, files, "build/android/app/src/main/java/com/nova/env/NetworkAdapter.java", "NetworkAdapter")
}

func assertFile(t *testing.T, files []shared.File, path string, want string) {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			if !strings.Contains(file.Content, want) {
				t.Fatalf("file %s missing %q", path, want)
			}
			return
		}
	}
	t.Fatalf("missing file %s", path)
}
