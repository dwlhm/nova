package standard

import _ "embed"

//go:embed platform/web/storage.web.js
var storageWebJS string

//go:embed platform/web/network.web.js
var networkWebJS string

//go:embed platform/web/clipboard.web.js
var clipboardWebJS string

//go:embed platform/web/notify.web.js
var notifyWebJS string

//go:embed platform/web/device.web.js
var deviceWebJS string

// WebPlatformAdapter returns bundled adapter source for a platform/web adapter path.
func WebPlatformAdapter(path string) (string, bool) {
	switch path {
	case "platform/web/storage.web.js":
		return storageWebJS, true
	case "platform/web/network.web.js":
		return networkWebJS, true
	case "platform/web/clipboard.web.js":
		return clipboardWebJS, true
	case "platform/web/notify.web.js":
		return notifyWebJS, true
	case "platform/web/device.web.js":
		return deviceWebJS, true
	default:
		return "", false
	}
}

// WebPlatformAdapterPaths returns sorted known bundled web adapter paths.
func WebPlatformAdapterPaths() []string {
	return []string{
		"platform/web/clipboard.web.js",
		"platform/web/device.web.js",
		"platform/web/network.web.js",
		"platform/web/notify.web.js",
		"platform/web/storage.web.js",
	}
}
