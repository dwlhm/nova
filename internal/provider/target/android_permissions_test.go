package target

import (
	"testing"

	"github.com/dwlhm/nova/internal/core/security"
)

func TestAndroidManifestPermissions(t *testing.T) {
	got := AndroidManifestPermissions([]security.Permission{
		"storage.read",
		"storage.write",
		"network.request",
		"storage.read",
	})
	want := []string{
		"android.permission.INTERNET",
		"android.permission.READ_EXTERNAL_STORAGE",
		"android.permission.WRITE_EXTERNAL_STORAGE",
	}
	if len(got) != len(want) {
		t.Fatalf("permissions = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("permissions[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
