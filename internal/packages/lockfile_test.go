package packages

import "testing"

func TestParseLockfileCapturesPackageEntriesAndTargetAdapters(t *testing.T) {
	input := `[[packages]]
name = "@env/storage"
version = "1.0.0"
content_hash = "storage-hash"

[packages.target_adapters]
web = "platform/web/storage.web.js"

[[packages]]
name = "@nova/ui"
version = "1.1.0"
content_hash = "ui-hash"

[packages.target_adapters]
web = "platform/web/ui.web.js"
android = "platform/android/ui.android.java"
`
	lockfile, diagnostics := ParseLockfile(input)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}
	if len(lockfile.Entries) != 2 {
		t.Fatalf("entries = %+v", lockfile.Entries)
	}
	if lockfile.Entries[0].Name != "@env/storage" || lockfile.Entries[0].ContentHash != "storage-hash" {
		t.Fatalf("first entry = %+v", lockfile.Entries[0])
	}
	if lockfile.Entries[0].TargetAdapters["web"] != "platform/web/storage.web.js" {
		t.Fatalf("adapters = %+v", lockfile.Entries[0].TargetAdapters)
	}
}
