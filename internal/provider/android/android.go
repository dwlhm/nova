package android

import (
	"strings"

	"github.com/dwlhm/nova/internal/core/diagnostic"
	"github.com/dwlhm/nova/internal/core/security"
	"github.com/dwlhm/nova/internal/project"
	"github.com/dwlhm/nova/internal/provider/shared"
	"github.com/dwlhm/nova/internal/provider/target"
)

type targetConfig struct {
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

func parseTargetConfig(manifest project.Manifest) (targetConfig, []shared.Diagnostic) {
	targetEntry := manifest.Targets["android"]
	options := targetEntry.Options
	renderer := strings.TrimSpace(targetEntry.Renderer)
	if renderer == "" {
		renderer = "@nova/android"
	}
	if renderer != "@nova/android" {
		return targetConfig{}, []shared.Diagnostic{{
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
		return targetConfig{}, diagnostics
	}

	versionName := strings.TrimSpace(options["version_name"])
	if versionName == "" {
		versionName = manifest.Project.Version
	}
	label := strings.TrimSpace(options["label"])
	if label == "" {
		label = manifest.Project.Name
	}
	return targetConfig{
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

func settingsGradle(name string, config targetConfig) string {
	if strings.TrimSpace(name) == "" {
		name = "nova-app"
	}
	return "pluginManagement {\n    repositories {\n        google()\n        mavenCentral()\n        gradlePluginPortal()\n    }\n    plugins {\n        id(\"com.android.application\") version " + shared.QuoteCodeString(config.GradlePlugin) + "\n    }\n}\n\ndependencyResolutionManagement {\n    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)\n    repositories {\n        google()\n        mavenCentral()\n    }\n}\n\nrootProject.name = " + shared.QuoteCodeString(name) + "\ninclude(\":app\")\ninclude(\":nova-scheduler\")\n"
}

func gradleProperties(config targetConfig) string {
	_ = config
	return "android.nonTransitiveRClass=true\n"
}

func rootGradle(name string, config targetConfig) string {
	if strings.TrimSpace(name) == "" {
		name = "nova-app"
	}
	_ = config
	return "// Generated Nova Android project for " + shared.EscapeGradleComment(name) + ".\n"
}

func appGradle(config targetConfig) string {
	javaVersion := "JavaVersion.VERSION_" + strings.ReplaceAll(config.JavaVersion, ".", "_")
	return "plugins {\n    id(\"com.android.application\")\n}\n\nandroid {\n    namespace = " + shared.QuoteCodeString(config.Namespace) + "\n    compileSdk = " + config.CompileSDK + "\n\n    defaultConfig {\n        applicationId = " + shared.QuoteCodeString(config.ApplicationID) + "\n        minSdk = " + config.MinSDK + "\n        targetSdk = " + config.TargetSDK + "\n        versionCode = " + config.VersionCode + "\n        versionName = " + shared.QuoteCodeString(config.VersionName) + "\n    }\n\n    compileOptions {\n        sourceCompatibility = " + javaVersion + "\n        targetCompatibility = " + javaVersion + "\n    }\n}\n\ndependencies {\n    implementation(project(\":nova-scheduler\"))\n}\n"
}

func androidManifestXML(config targetConfig, permissions []security.Permission) string {
	manifestPermissions := target.AndroidManifestPermissions(permissions)
	var usesPermissions strings.Builder
	for _, permission := range manifestPermissions {
		usesPermissions.WriteString("    <uses-permission android:name=\"")
		usesPermissions.WriteString(escapeXML(permission))
		usesPermissions.WriteString("\" />\n")
	}
	return "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<manifest xmlns:android=\"http://schemas.android.com/apk/res/android\">\n" +
		usesPermissions.String() +
		"    <application android:theme=\"@style/" + escapeXML(config.Theme) + "\" android:label=" + quoteXML(config.Label) + ">\n" +
		"        <activity android:name=\"" + escapeXML(config.Namespace) + ".MainActivity\" android:exported=\"true\">\n" +
		"            <intent-filter>\n" +
		"                <action android:name=\"android.intent.action.MAIN\" />\n" +
		"                <category android:name=\"android.intent.category.LAUNCHER\" />\n" +
		"            </intent-filter>\n" +
		"        </activity>\n" +
		"    </application>\n" +
		"</manifest>\n"
}

func quoteXML(value string) string {
	return "\"" + escapeXML(value) + "\""
}

func escapeXML(value string) string {
	value = strings.ReplaceAll(value, "&", "&amp;")
	value = strings.ReplaceAll(value, "\"", "&quot;")
	value = strings.ReplaceAll(value, "<", "&lt;")
	value = strings.ReplaceAll(value, ">", "&gt;")
	return value
}
