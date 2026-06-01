package target

import "github.com/dwlhm/nova/internal/core/security"

type PlanManifest struct {
	ID                   string
	Families             []string
	ExternalCapabilities []ExternalCapability
	PermissionMappings   security.PermissionMap
}

type ExternalCapability struct {
	Source     string
	Operations []ExternalOperation
}

type ExternalOperation struct {
	Name            string
	Inputs          []Field
	Output          string
	Permissions     []security.Permission
	Implementations []Implementation
}

type Field struct {
	Name     string
	Optional bool
	Type     string
}

type Implementation struct {
	Path    string
	Target  string
	Family  string
	Common  bool
	Default bool
}

func WebPlanManifest() PlanManifest {
	return PlanManifest{
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
		PermissionMappings: defaultPlanPermissionMappings(),
	}
}

func AndroidPlanManifest() PlanManifest {
	return PlanManifest{
		ID:       "android",
		Families: []string{"mobile"},
		ExternalCapabilities: []ExternalCapability{
			storageCapability("android", "platform/android/storage.android.java"),
			networkCapability("android", "platform/android/network.android.java"),
			clipboardCapability("android", "platform/android/clipboard.android.java"),
			notifyCapability("android", "platform/android/notify.android.java"),
			deviceCapability("android", "platform/android/device.android.java"),
			dspCapability("android", "platform/android/dsp.android.java"),
			osCapability("android", "platform/android/os.android.java"),
		},
		PermissionMappings: defaultPlanPermissionMappings(),
	}
}

func PlanManifestFor(profile string) (PlanManifest, bool) {
	switch profile {
	case "web":
		return WebPlanManifest(), true
	case "android":
		return AndroidPlanManifest(), true
	default:
		return PlanManifest{}, false
	}
}

func storageCapability(profile string, implementation string) ExternalCapability {
	return ExternalCapability{
		Source: "@env/storage",
		Operations: []ExternalOperation{
			{Name: "load", Inputs: []Field{{Name: "key", Type: "string"}}, Output: "unknown", Permissions: []security.Permission{"storage.read"}, Implementations: exactImplementation(profile, implementation)},
			{Name: "get", Inputs: []Field{{Name: "key", Type: "string"}}, Output: "unknown", Permissions: []security.Permission{"storage.read"}, Implementations: exactImplementation(profile, implementation)},
			{Name: "set", Inputs: []Field{{Name: "key", Type: "string"}, {Name: "value", Type: "unknown"}}, Output: "void", Permissions: []security.Permission{"storage.write"}, Implementations: exactImplementation(profile, implementation)},
			{Name: "remove", Inputs: []Field{{Name: "key", Type: "string"}}, Output: "void", Permissions: []security.Permission{"storage.write"}, Implementations: exactImplementation(profile, implementation)},
			{Name: "clear", Inputs: []Field{{Name: "scope", Type: "string", Optional: true}}, Output: "void", Permissions: []security.Permission{"storage.write"}, Implementations: exactImplementation(profile, implementation)},
		},
	}
}

func networkCapability(profile string, implementation string) ExternalCapability {
	return ExternalCapability{
		Source: "@env/network",
		Operations: []ExternalOperation{
			{Name: "request", Inputs: []Field{{Name: "input", Type: "unknown"}}, Output: "unknown", Permissions: []security.Permission{"network.request"}, Implementations: exactImplementation(profile, implementation)},
		},
	}
}

func clipboardCapability(profile string, implementation string) ExternalCapability {
	return ExternalCapability{
		Source: "@env/clipboard",
		Operations: []ExternalOperation{
			{Name: "read", Output: "string", Permissions: []security.Permission{"clipboard.read"}, Implementations: exactImplementation(profile, implementation)},
			{Name: "write", Inputs: []Field{{Name: "value", Type: "string"}}, Output: "void", Permissions: []security.Permission{"clipboard.write"}, Implementations: exactImplementation(profile, implementation)},
		},
	}
}

func notifyCapability(profile string, implementation string) ExternalCapability {
	return ExternalCapability{
		Source: "@env/notify",
		Operations: []ExternalOperation{
			{Name: "send", Inputs: []Field{{Name: "msg", Type: "string"}, {Name: "type", Type: "string", Optional: true}}, Output: "void", Permissions: []security.Permission{"notification.send"}, Implementations: exactImplementation(profile, implementation)},
		},
	}
}

func deviceCapability(profile string, implementation string) ExternalCapability {
	return ExternalCapability{
		Source: "@env/device",
		Operations: []ExternalOperation{
			{Name: "info", Output: "unknown", Permissions: []security.Permission{"device.info"}, Implementations: exactImplementation(profile, implementation)},
		},
	}
}

func dspCapability(profile string, implementation string) ExternalCapability {
	return ExternalCapability{
		Source: "@env/dsp",
		Operations: []ExternalOperation{
			{Name: "setProfile", Inputs: []Field{{Name: "bass", Type: "number"}, {Name: "mid", Type: "number"}, {Name: "treble", Type: "number"}, {Name: "masterGain", Type: "number"}}, Output: "void", Permissions: []security.Permission{"device.info"}, Implementations: exactImplementation(profile, implementation)},
			{Name: "reset", Output: "void", Permissions: []security.Permission{"device.info"}, Implementations: exactImplementation(profile, implementation)},
		},
	}
}

func osCapability(profile string, implementation string) ExternalCapability {
	return ExternalCapability{
		Source: "@env/os",
		Operations: []ExternalOperation{
			{Name: "send", Inputs: []Field{{Name: "msg", Type: "string"}, {Name: "type", Type: "string", Optional: true}}, Output: "void", Permissions: []security.Permission{"notification.send"}, Implementations: exactImplementation(profile, implementation)},
		},
	}
}

func exactImplementation(profile string, implementation string) []Implementation {
	return []Implementation{{Path: implementation, Target: profile}}
}

func defaultPlanPermissionMappings() security.PermissionMap {
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
