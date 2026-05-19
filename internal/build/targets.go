package build

import "github.com/dwlhm/nova/internal/security"

func WebTargetManifest() TargetManifest {
	return TargetManifest{
		ID:       "web",
		Families: []string{"browser"},
		ExternalCapabilities: []ExternalCapability{
			storageCapability("web", "platform/web/storage.web.js"),
			networkCapability("web", "platform/web/network.web.js"),
			clipboardCapability("web", "platform/web/clipboard.web.js"),
			notifyCapability("web", "platform/web/notify.web.js"),
			deviceCapability("web", "platform/web/device.web.js"),
			dspCapability("web", "platform/web/dsp.web.js"),
			osCapability("web", "platform/web/os.web.js"),
		},
		PermissionMappings: defaultPermissionMappings(),
	}
}

func AndroidTargetManifest() TargetManifest {
	return TargetManifest{
		ID:       "android",
		Families: []string{"mobile"},
		ExternalCapabilities: []ExternalCapability{
			storageCapability("android", "platform/android/storage.android.kt"),
			networkCapability("android", "platform/android/network.android.kt"),
			clipboardCapability("android", "platform/android/clipboard.android.kt"),
			notifyCapability("android", "platform/android/notify.android.kt"),
			deviceCapability("android", "platform/android/device.android.kt"),
			dspCapability("android", "platform/android/dsp.android.kt"),
			osCapability("android", "platform/android/os.android.kt"),
		},
		PermissionMappings: defaultPermissionMappings(),
	}
}

func TargetManifestFor(target string) (TargetManifest, bool) {
	switch target {
	case "web":
		return WebTargetManifest(), true
	case "android":
		return AndroidTargetManifest(), true
	default:
		return TargetManifest{}, false
	}
}

func storageCapability(target string, implementation string) ExternalCapability {
	return ExternalCapability{
		Source: "@env/storage",
		Operations: []ExternalOperation{
			{Name: "load", Inputs: []Field{{Name: "key", Type: "string"}}, Output: "unknown", Permissions: []security.Permission{"storage.read"}, Implementations: exactImplementation(target, implementation)},
			{Name: "get", Inputs: []Field{{Name: "key", Type: "string"}}, Output: "unknown", Permissions: []security.Permission{"storage.read"}, Implementations: exactImplementation(target, implementation)},
			{Name: "set", Inputs: []Field{{Name: "key", Type: "string"}, {Name: "value", Type: "unknown"}}, Output: "void", Permissions: []security.Permission{"storage.write"}, Implementations: exactImplementation(target, implementation)},
			{Name: "remove", Inputs: []Field{{Name: "key", Type: "string"}}, Output: "void", Permissions: []security.Permission{"storage.write"}, Implementations: exactImplementation(target, implementation)},
			{Name: "clear", Inputs: []Field{{Name: "scope", Type: "string", Optional: true}}, Output: "void", Permissions: []security.Permission{"storage.write"}, Implementations: exactImplementation(target, implementation)},
		},
	}
}

func networkCapability(target string, implementation string) ExternalCapability {
	return ExternalCapability{
		Source: "@env/network",
		Operations: []ExternalOperation{
			{Name: "request", Inputs: []Field{{Name: "input", Type: "unknown"}}, Output: "unknown", Permissions: []security.Permission{"network.request"}, Implementations: exactImplementation(target, implementation)},
		},
	}
}

func clipboardCapability(target string, implementation string) ExternalCapability {
	return ExternalCapability{
		Source: "@env/clipboard",
		Operations: []ExternalOperation{
			{Name: "read", Output: "string", Permissions: []security.Permission{"clipboard.read"}, Implementations: exactImplementation(target, implementation)},
			{Name: "write", Inputs: []Field{{Name: "value", Type: "string"}}, Output: "void", Permissions: []security.Permission{"clipboard.write"}, Implementations: exactImplementation(target, implementation)},
		},
	}
}

func notifyCapability(target string, implementation string) ExternalCapability {
	return ExternalCapability{
		Source: "@env/notify",
		Operations: []ExternalOperation{
			{Name: "send", Inputs: []Field{{Name: "msg", Type: "string"}, {Name: "type", Type: "string", Optional: true}}, Output: "void", Permissions: []security.Permission{"notification.send"}, Implementations: exactImplementation(target, implementation)},
		},
	}
}

func deviceCapability(target string, implementation string) ExternalCapability {
	return ExternalCapability{
		Source: "@env/device",
		Operations: []ExternalOperation{
			{Name: "info", Output: "unknown", Permissions: []security.Permission{"device.info"}, Implementations: exactImplementation(target, implementation)},
		},
	}
}

func dspCapability(target string, implementation string) ExternalCapability {
	return ExternalCapability{
		Source: "@env/dsp",
		Operations: []ExternalOperation{
			{Name: "setProfile", Inputs: []Field{{Name: "bass", Type: "number"}, {Name: "mid", Type: "number"}, {Name: "treble", Type: "number"}, {Name: "masterGain", Type: "number"}}, Output: "void", Permissions: []security.Permission{"device.info"}, Implementations: exactImplementation(target, implementation)},
			{Name: "reset", Output: "void", Permissions: []security.Permission{"device.info"}, Implementations: exactImplementation(target, implementation)},
		},
	}
}

func osCapability(target string, implementation string) ExternalCapability {
	return ExternalCapability{
		Source: "@env/os",
		Operations: []ExternalOperation{
			{Name: "send", Inputs: []Field{{Name: "msg", Type: "string"}, {Name: "type", Type: "string", Optional: true}}, Output: "void", Permissions: []security.Permission{"notification.send"}, Implementations: exactImplementation(target, implementation)},
		},
	}
}

func exactImplementation(target string, implementation string) []Implementation {
	return []Implementation{{Path: implementation, Target: target}}
}

func defaultPermissionMappings() security.PermissionMap {
	return security.PermissionSet(
		"storage.read",
		"storage.write",
		"network.request",
		"clipboard.read",
		"clipboard.write",
		"notification.send",
		"device.info",
	)
}
