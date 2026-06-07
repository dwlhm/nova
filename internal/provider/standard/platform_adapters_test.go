package standard

import "testing"

func TestWebPlatformAdapterPaths(t *testing.T) {
	paths := WebPlatformAdapterPaths()
	if len(paths) < 5 {
		t.Fatalf("paths = %v, want bundled web adapters", paths)
	}
	for _, path := range paths {
		content, ok := WebPlatformAdapter(path)
		if !ok || content == "" {
			t.Fatalf("missing adapter content for %s", path)
		}
		if !contains(content, "register") {
			t.Fatalf("adapter %s should export register()", path)
		}
	}
}

func TestAndroidPlatformAdapterSource(t *testing.T) {
	for _, path := range AndroidPlatformAdapterPaths() {
		content, ok := AndroidPlatformAdapterSource(path)
		if !ok || content == "" {
			t.Fatalf("missing adapter content for %s", path)
		}
		class, ok := AndroidPlatformAdapterClass(path)
		if !ok || class == "" {
			t.Fatalf("missing adapter class for %s", path)
		}
		source, ok := AndroidPlatformAdapterSourceID(path)
		if !ok || source == "" {
			t.Fatalf("missing adapter source id for %s", path)
		}
		if !contains(content, "invoke") {
			t.Fatalf("adapter %s should expose invoke()", path)
		}
	}
}

func TestAndroidStorageAdapterPersistsMapsAsJson(t *testing.T) {
	content, ok := AndroidPlatformAdapterSource("platform/android/storage.android.java")
	if !ok {
		t.Fatal("missing android storage adapter source")
	}
	for _, token := range []string{"encodeObject", "decodeObject", "LinkedHashMap", "JSONObject"} {
		if !contains(content, token) {
			t.Fatalf("android storage adapter missing %q", token)
		}
	}
	if !contains(content, "if (value instanceof Map<?, ?>)") {
		t.Fatal("android storage adapter must JSON-encode map values")
	}
}

func contains(value string, needle string) bool {
	return len(value) >= len(needle) && (value == needle || len(needle) == 0 || indexOf(value, needle) >= 0)
}

func indexOf(value string, needle string) int {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
