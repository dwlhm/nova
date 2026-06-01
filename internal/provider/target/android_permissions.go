package target

import (
	"sort"

	"github.com/dwlhm/nova/internal/core/security"
)

// AndroidManifestPermission maps a Nova permission to an Android manifest permission name.
func AndroidManifestPermission(permission security.Permission) (string, bool) {
	switch permission {
	case "storage.read":
		return "android.permission.READ_EXTERNAL_STORAGE", true
	case "storage.write":
		return "android.permission.WRITE_EXTERNAL_STORAGE", true
	case "network.request":
		return "android.permission.INTERNET", true
	case "clipboard.read", "clipboard.write":
		return "android.permission.READ_CLIPBOARD", false
	case "notification.send":
		return "android.permission.POST_NOTIFICATIONS", true
	case "device.info":
		return "android.permission.READ_PHONE_STATE", true
	default:
		return "", false
	}
}

// AndroidManifestPermissions returns sorted unique Android manifest permission names.
func AndroidManifestPermissions(permissions []security.Permission) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(permissions))
	for _, permission := range permissions {
		name, ok := AndroidManifestPermission(permission)
		if !ok || name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
