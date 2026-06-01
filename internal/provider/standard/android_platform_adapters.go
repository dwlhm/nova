package standard

import _ "embed"

//go:embed platform/android/storage.android.java
var storageAndroidJava string

//go:embed platform/android/network.android.java
var networkAndroidJava string

//go:embed platform/android/clipboard.android.java
var clipboardAndroidJava string

//go:embed platform/android/notify.android.java
var notifyAndroidJava string

//go:embed platform/android/device.android.java
var deviceAndroidJava string

//go:embed platform/android/dsp.android.java
var dspAndroidJava string

//go:embed platform/android/os.android.java
var osAndroidJava string

type androidPlatformAdapter struct {
	content  string
	class    string
	sourceID string
}

func androidPlatformAdapterRegistry() map[string]androidPlatformAdapter {
	return map[string]androidPlatformAdapter{
		"platform/android/storage.android.java":   {content: storageAndroidJava, class: "com.nova.env.StorageAdapter", sourceID: "@env/storage"},
		"platform/android/network.android.java":   {content: networkAndroidJava, class: "com.nova.env.NetworkAdapter", sourceID: "@env/network"},
		"platform/android/clipboard.android.java": {content: clipboardAndroidJava, class: "com.nova.env.ClipboardAdapter", sourceID: "@env/clipboard"},
		"platform/android/notify.android.java":    {content: notifyAndroidJava, class: "com.nova.env.NotifyAdapter", sourceID: "@env/notify"},
		"platform/android/device.android.java":    {content: deviceAndroidJava, class: "com.nova.env.DeviceAdapter", sourceID: "@env/device"},
		"platform/android/dsp.android.java":       {content: dspAndroidJava, class: "com.nova.env.DspAdapter", sourceID: "@env/dsp"},
		"platform/android/os.android.java":        {content: osAndroidJava, class: "com.nova.env.OsAdapter", sourceID: "@env/os"},
	}
}

// AndroidPlatformAdapterPaths returns sorted bundled android adapter paths.
func AndroidPlatformAdapterPaths() []string {
	return []string{
		"platform/android/clipboard.android.java",
		"platform/android/device.android.java",
		"platform/android/dsp.android.java",
		"platform/android/network.android.java",
		"platform/android/notify.android.java",
		"platform/android/os.android.java",
		"platform/android/storage.android.java",
	}
}

// AndroidPlatformAdapterSource returns bundled adapter source for copy into generated artifacts.
func AndroidPlatformAdapterSource(path string) (string, bool) {
	adapter, ok := androidPlatformAdapterRegistry()[path]
	if !ok || adapter.content == "" {
		return "", false
	}
	return adapter.content, true
}

// AndroidPlatformAdapterClass returns the Java adapter class for a platform/android adapter path.
func AndroidPlatformAdapterClass(path string) (string, bool) {
	adapter, ok := androidPlatformAdapterRegistry()[path]
	if !ok || adapter.class == "" {
		return "", false
	}
	return adapter.class, true
}

// AndroidPlatformAdapterSourceID maps an adapter path to the Nova external source id.
func AndroidPlatformAdapterSourceID(path string) (string, bool) {
	adapter, ok := androidPlatformAdapterRegistry()[path]
	if !ok || adapter.sourceID == "" {
		return "", false
	}
	return adapter.sourceID, true
}
