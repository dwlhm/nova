package androidtarget

import (
	"strings"

	"github.com/dwlhm/nova/internal/core/diagnostic"
	"github.com/dwlhm/nova/internal/project"
	"github.com/dwlhm/nova/internal/provider/shared"
)

// Config holds Android target options required for artifact generation.
type Config struct {
	ApplicationID string
	Namespace     string
	CompileSDK    string
	MinSDK        string
	TargetSDK     string
	VersionCode   string
	VersionName   string
	GradlePlugin  string
	Theme         string
	ThemeParent   string
	Label         string
	JavaVersion   string
}

// ParseConfig validates and parses targets.android from the project manifest.
func ParseConfig(manifest project.Manifest) (Config, []shared.Diagnostic) {
	target := manifest.Targets["android"]
	options := target.Options
	renderer := strings.TrimSpace(target.Renderer)
	if renderer == "" {
		renderer = "@nova/android"
	}
	if renderer != "@nova/android" {
		return Config{}, []shared.Diagnostic{{
			Code:     "NVA-ANDROID-002",
			Severity: diagnostic.SeverityError,
			Message:  "targets.android.renderer must be @nova/android",
		}}
	}
	required := []string{
		"application_id",
		"namespace",
		"compile_sdk",
		"min_sdk",
		"target_sdk",
		"version_code",
		"gradle_plugin",
		"theme",
		"theme_parent",
		"java_version",
	}
	diagnostics := make([]shared.Diagnostic, 0)
	for _, key := range required {
		if strings.TrimSpace(options[key]) == "" {
			diagnostics = append(diagnostics, shared.Diagnostic{
				Code:     "NVA-ANDROID-001",
				Severity: diagnostic.SeverityError,
				Message:  "targets.android." + key + " is required for Android artifact generation",
			})
		}
	}
	if len(diagnostics) > 0 {
		return Config{}, diagnostics
	}

	versionName := strings.TrimSpace(options["version_name"])
	if versionName == "" {
		versionName = manifest.Project.Version
	}
	label := strings.TrimSpace(options["label"])
	if label == "" {
		label = manifest.Project.Name
	}
	return Config{
		ApplicationID: options["application_id"],
		Namespace:     options["namespace"],
		CompileSDK:    options["compile_sdk"],
		MinSDK:        options["min_sdk"],
		TargetSDK:     options["target_sdk"],
		VersionCode:   options["version_code"],
		VersionName:   versionName,
		GradlePlugin:  options["gradle_plugin"],
		Theme:         options["theme"],
		ThemeParent:   options["theme_parent"],
		Label:         label,
		JavaVersion:   options["java_version"],
	}, nil
}

func StylesXML(config Config) string {
	return "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<resources>\n    <style name=\"" + escapeXML(config.Theme) + "\" parent=\"" + escapeXML(config.ThemeParent) + "\">\n        <item name=\"android:windowActionBar\">false</item>\n        <item name=\"android:windowNoTitle\">true</item>\n    </style>\n</resources>\n"
}

func escapeXML(value string) string {
	value = strings.ReplaceAll(value, "&", "&amp;")
	value = strings.ReplaceAll(value, "\"", "&quot;")
	value = strings.ReplaceAll(value, "<", "&lt;")
	value = strings.ReplaceAll(value, ">", "&gt;")
	return value
}
