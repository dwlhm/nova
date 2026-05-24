package artifact

import (
	"strconv"
	"strings"

	"github.com/dwlhm/nova/internal/build"
	"github.com/dwlhm/nova/internal/diagnostic"
	"github.com/dwlhm/nova/internal/lexer"
	"github.com/dwlhm/nova/internal/project"
	"github.com/dwlhm/nova/internal/routing"
	"github.com/dwlhm/nova/internal/view"
)

type androidTargetConfig struct {
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

func androidConfig(manifest project.Manifest) (androidTargetConfig, []Diagnostic) {
	target := manifest.Targets["android"]
	options := target.Options
	renderer := strings.TrimSpace(target.Renderer)
	if renderer == "" {
		renderer = "@nova/android"
	}
	if renderer != "@nova/android" {
		return androidTargetConfig{}, []Diagnostic{{
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
	diagnostics := make([]Diagnostic, 0)
	for _, key := range required {
		if strings.TrimSpace(options[key]) == "" {
			diagnostics = append(diagnostics, Diagnostic{
				Code:     "NVA-ANDROID-001",
				Severity: diagnostic.SeverityError,
				Message:  "targets.android." + key + " is required for Android artifact generation",
			})
		}
	}
	if len(diagnostics) > 0 {
		return androidTargetConfig{}, diagnostics
	}

	versionName := strings.TrimSpace(options["version_name"])
	if versionName == "" {
		versionName = manifest.Project.Version
	}
	label := strings.TrimSpace(options["label"])
	if label == "" {
		label = manifest.Project.Name
	}
	return androidTargetConfig{
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

func androidSettings(name string, config androidTargetConfig) string {
	if strings.TrimSpace(name) == "" {
		name = "nova-app"
	}
	return "pluginManagement {\n    repositories {\n        google()\n        mavenCentral()\n        gradlePluginPortal()\n    }\n    plugins {\n        id(\"com.android.application\") version " + quoteCodeString(config.GradlePlugin) + "\n    }\n}\n\ndependencyResolutionManagement {\n    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)\n    repositories {\n        google()\n        mavenCentral()\n    }\n}\n\nrootProject.name = " + quoteCodeString(name) + "\ninclude(\":app\")\ninclude(\":nova-scheduler\")\n"
}

func androidGradleProperties(config androidTargetConfig) string {
	return "android.nonTransitiveRClass=true\n"
}

func androidGradle(name string, config androidTargetConfig) string {
	if strings.TrimSpace(name) == "" {
		name = "nova-app"
	}
	_ = config
	return "// Generated Nova Android project for " + escapeGradleComment(name) + ".\n"
}

func androidAppGradle(config androidTargetConfig) string {
	javaVersion := "JavaVersion.VERSION_" + strings.ReplaceAll(config.JavaVersion, ".", "_")
	return "plugins {\n    id(\"com.android.application\")\n}\n\nandroid {\n    namespace = " + quoteCodeString(config.Namespace) + "\n    compileSdk = " + config.CompileSDK + "\n\n    defaultConfig {\n        applicationId = " + quoteCodeString(config.ApplicationID) + "\n        minSdk = " + config.MinSDK + "\n        targetSdk = " + config.TargetSDK + "\n        versionCode = " + config.VersionCode + "\n        versionName = " + quoteCodeString(config.VersionName) + "\n    }\n\n    compileOptions {\n        sourceCompatibility = " + javaVersion + "\n        targetCompatibility = " + javaVersion + "\n    }\n}\n\ndependencies {\n    implementation(project(\":nova-scheduler\"))\n}\n"
}

func androidManifest(config androidTargetConfig) string {
	return "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<manifest xmlns:android=\"http://schemas.android.com/apk/res/android\">\n    <application android:theme=\"@style/" + escapeXML(config.Theme) + "\" android:label=" + quoteXML(config.Label) + ">\n        <activity android:name=\"" + escapeXML(config.Namespace) + ".MainActivity\" android:exported=\"true\">\n            <intent-filter>\n                <action android:name=\"android.intent.action.MAIN\" />\n                <category android:name=\"android.intent.category.LAUNCHER\" />\n            </intent-filter>\n        </activity>\n    </application>\n</manifest>\n"
}

func androidStyles(config androidTargetConfig) string {
	return "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<resources>\n    <style name=\"" + escapeXML(config.Theme) + "\" parent=\"" + escapeXML(config.ThemeParent) + "\">\n        <item name=\"android:windowActionBar\">false</item>\n        <item name=\"android:windowNoTitle\">true</item>\n    </style>\n</resources>\n"
}

func androidMainActivity(name string, bundle irBundle, config androidTargetConfig, styles []StyleAsset) string {
	return androidNativeMainActivity(bundle, config, styles)
}

func androidRuntime(config androidTargetConfig) string {
	return androidJavaRuntime(config)
}

func androidJavaRuntime(config androidTargetConfig) string {
	return "package " + config.Namespace + ";\n\n" + `import java.net.URI;
import java.net.URLDecoder;
import java.net.URLEncoder;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

public final class NovaRuntime {
    private NovaRuntime() {}

    public static String routeKey(Object value) {
        Map<String, Object> route = routeObject(value);
        String query = queryString(route.get("query"));
        String fragment = route.get("fragment") == null ? "" : route.get("fragment").toString().replaceFirst("^#", "");
        return route.get("path").toString() + query + (fragment.isEmpty() ? "" : "#" + fragment);
    }

    public static Object cloneRoute(Object value) {
        if (value instanceof Map<?, ?>) {
            Map<String, Object> out = new LinkedHashMap<>();
            for (Map.Entry<?, ?> entry : ((Map<?, ?>) value).entrySet()) {
                out.put(String.valueOf(entry.getKey()), entry.getValue());
            }
            return out;
        }
        return value;
    }

    public static Object routeValueForShape(Object value, List<String> patterns) {
        Map<String, Object> route = routeObject(value);
        if (value instanceof String) return route.get("path");
        Map<String, Object> next = value instanceof Map<?, ?> ? new LinkedHashMap<>() : new LinkedHashMap<>();
        if (value instanceof Map<?, ?>) {
            for (Map.Entry<?, ?> entry : ((Map<?, ?>) value).entrySet()) {
                next.put(String.valueOf(entry.getKey()), entry.getValue());
            }
        }
        next.put("path", route.get("path"));
        Object query = route.get("query");
        if (query instanceof Map<?, ?> && !((Map<?, ?>) query).isEmpty()) next.put("query", query); else next.remove("query");
        Object fragment = route.get("fragment");
        if (fragment != null && !fragment.toString().isEmpty()) next.put("fragment", fragment); else next.remove("fragment");
        RouteMatch best = bestRouteMatch(patterns, route.get("path").toString());
        if (!best.params.isEmpty()) next.put("params", best.params); else next.remove("params");
        return next;
    }

    public static Object evaluate(String expression, Map<String, Object> state, Map<String, Object> payload) {
        String trimmed = expression == null ? "" : expression.trim();
        String[] plus = splitBinary(trimmed, "+");
        if (plus != null) {
            Object left = evaluate(plus[0], state, payload);
            Object right = evaluate(plus[1], state, payload);
            if (left instanceof Number && right instanceof Number) {
                return numberValue(left) + numberValue(right);
            }
            return textValue(left) + textValue(right);
        }
        String[] minus = splitBinary(trimmed, "-");
        if (minus != null) {
            return numberValue(evaluate(minus[0], state, payload)) - numberValue(evaluate(minus[1], state, payload));
        }
        return evaluateAtom(trimmed, state, payload);
    }

    private static Object evaluateAtom(String expression, Map<String, Object> state, Map<String, Object> payload) {
        if (expression.startsWith("({") && expression.endsWith("})")) return evaluateRecord(expression, state, payload);
        if (expression.startsWith("payload.")) return payload.get(expression.substring("payload.".length()));
        if (expression.startsWith("state.")) return state.get(expression.substring("state.".length()));
        if (expression.startsWith("\"") && expression.endsWith("\"") && expression.length() >= 2) {
            return expression.substring(1, expression.length() - 1);
        }
        try {
            if (!expression.isEmpty()) return Double.parseDouble(expression);
        } catch (NumberFormatException ignored) {}
        if ("true".equals(expression)) return Boolean.TRUE;
        if ("false".equals(expression)) return Boolean.FALSE;
        if ("null".equals(expression) || "undefined".equals(expression) || expression.isEmpty()) return null;
        return expression;
    }

    private static Map<String, Object> evaluateRecord(String expression, Map<String, Object> state, Map<String, Object> payload) {
        String body = expression.substring(2, expression.length() - 2).trim();
        if (body.isEmpty()) return Collections.emptyMap();
        Map<String, Object> out = new LinkedHashMap<>();
        for (String field : splitRecordFields(body)) {
            int colon = field.indexOf(':');
            if (colon < 0) continue;
            String name = field.substring(0, colon).trim();
            String value = field.substring(colon + 1).trim();
            out.put(name, evaluate(value, state, payload));
        }
        return out;
    }

    private static List<String> splitRecordFields(String body) {
        List<String> fields = new ArrayList<>();
        int start = 0;
        boolean inString = false;
        for (int index = 0; index < body.length(); index++) {
            char ch = body.charAt(index);
            if (ch == '"') inString = !inString;
            if (ch == ',' && !inString) {
                fields.add(body.substring(start, index).trim());
                start = index + 1;
            }
        }
        fields.add(body.substring(start).trim());
        return fields;
    }

    private static String[] splitBinary(String expression, String operator) {
        String marker = " " + operator + " ";
        int index = expression.indexOf(marker);
        if (index < 0) return null;
        return new String[] { expression.substring(0, index), expression.substring(index + marker.length()) };
    }

    public static String pathOf(Object value) {
        Object path = routeObject(value).get("path");
        return path == null ? "/" : path.toString();
    }

    private static Map<String, Object> routeObject(Object value) {
        if (value instanceof Map<?, ?>) {
            Map<?, ?> input = (Map<?, ?>) value;
            Object rawPath = input.containsKey("path") ? input.get("path") : "/";
            Map<String, Object> parsed = routeObjectFromString(String.valueOf(rawPath));
            Map<String, Object> out = new LinkedHashMap<>();
            out.put("path", parsed.get("path"));
            out.put("query", input.containsKey("query") ? input.get("query") : parsed.get("query"));
            out.put("fragment", input.containsKey("fragment") ? input.get("fragment") : parsed.get("fragment"));
            return out;
        }
        return routeObjectFromString(value == null ? "/" : value.toString());
    }

    private static Map<String, Object> routeObjectFromString(String value) {
        String text = value == null ? "/" : value.trim();
        if (text.isEmpty()) text = "/";
        try {
            URI uri = URI.create(text);
            if (uri.isAbsolute()) {
                return record(
                    entry("path", normalizePath(uri.getRawPath() == null ? "/" : uri.getRawPath())),
                    entry("query", queryMap(uri.getRawQuery() == null ? "" : uri.getRawQuery())),
                    entry("fragment", safeDecodeComponent(uri.getRawFragment() == null ? "" : uri.getRawFragment()))
                );
            }
        } catch (Exception ignored) {}
        int hashIndex = text.indexOf('#');
        String beforeHash = hashIndex >= 0 ? text.substring(0, hashIndex) : text;
        String fragment = hashIndex >= 0 ? safeDecodeComponent(text.substring(hashIndex + 1)) : "";
        int queryIndex = beforeHash.indexOf('?');
        String rawPath = queryIndex >= 0 ? beforeHash.substring(0, queryIndex) : beforeHash;
        String rawQuery = queryIndex >= 0 ? beforeHash.substring(queryIndex + 1) : "";
        return record(entry("path", normalizePath(rawPath)), entry("query", queryMap(rawQuery)), entry("fragment", fragment));
    }

    private static String normalizePath(String value) {
        String cleaned = value == null ? "/" : value.trim();
        if (cleaned.isEmpty()) cleaned = "/";
        int hash = cleaned.indexOf('#');
        if (hash >= 0) cleaned = cleaned.substring(0, hash);
        int query = cleaned.indexOf('?');
        if (query >= 0) cleaned = cleaned.substring(0, query);
        if (!cleaned.startsWith("/")) cleaned = "/" + cleaned;
        List<String> segments = new ArrayList<>();
        for (String raw : cleaned.split("/")) {
            String segment = safeDecodePathSegment(raw);
            if (segment.isEmpty() || ".".equals(segment)) continue;
            if ("..".equals(segment)) {
                if (!segments.isEmpty()) segments.remove(segments.size() - 1);
            } else {
                segments.add(segment);
            }
        }
        return segments.isEmpty() ? "/" : "/" + joinStrings("/", segments);
    }

    public static boolean routeMatches(String pattern, String value) {
        return routeMatch(pattern, value).matched;
    }

    public static int routeMatchScore(String pattern, String value) {
        RouteMatch match = routeMatch(pattern, value);
        return match.matched ? match.score : -1;
    }

    public static int bestRouteScore(List<String> patterns, String value) {
        return bestRouteMatch(patterns, value).score;
    }

    private static RouteMatch bestRouteMatch(List<String> patterns, String value) {
        RouteMatch best = new RouteMatch(false, "", Collections.emptyMap(), -1, false);
        for (String pattern : patterns) {
            if (pattern == null || pattern.isEmpty()) continue;
            RouteMatch match = routeMatch(pattern, value);
            if (match.matched && (!best.matched || match.score > best.score)) best = match;
        }
        return best;
    }

    private static RouteMatch routeMatch(String pattern, String value) {
        String normalizedPattern = normalizePattern(pattern);
        String normalizedPath = normalizePath(value);
        if ("*".equals(normalizedPattern)) return new RouteMatch(true, normalizedPattern, Collections.emptyMap(), 0, true);
        List<String> patternSegments = routeSegments(normalizedPattern);
        List<String> pathSegments = routeSegments(normalizedPath);
        Map<String, String> params = new LinkedHashMap<>();
        int score = 0;
        for (int index = 0; index < patternSegments.size(); index++) {
            String segment = patternSegments.get(index);
            if ("*".equals(segment)) {
                return index == patternSegments.size() - 1
                    ? new RouteMatch(true, normalizedPattern, params, score + 10, false)
                    : new RouteMatch(false, normalizedPattern, Collections.emptyMap(), -1, false);
            }
            if (index >= pathSegments.size()) return new RouteMatch(false, normalizedPattern, Collections.emptyMap(), -1, false);
            String name = dynamicSegmentName(segment);
            if (name != null) {
                params.put(name, pathSegments.get(index));
                score += 50;
                continue;
            }
            if (!segment.equals(pathSegments.get(index))) return new RouteMatch(false, normalizedPattern, Collections.emptyMap(), -1, false);
            score += 100;
        }
        if (patternSegments.size() != pathSegments.size()) return new RouteMatch(false, normalizedPattern, Collections.emptyMap(), -1, false);
        return new RouteMatch(true, normalizedPattern, params, score + 1000, false);
    }

    private static String normalizePattern(String pattern) {
        String trimmed = pattern == null ? "" : pattern.trim();
        if ("*".equals(trimmed) || "/*".equals(trimmed)) return "*";
        if (trimmed.endsWith("/*")) return normalizePath(trimmed.substring(0, trimmed.length() - 2)) + "/*";
        return normalizePath(trimmed);
    }

    private static List<String> routeSegments(String value) {
        String normalized = normalizePath(value);
        if ("/".equals(normalized)) return Collections.emptyList();
        return Arrays.asList(normalized.substring(1).split("/"));
    }

    private static String dynamicSegmentName(String segment) {
        if (segment.startsWith(":") && segment.length() > 1) return segment.substring(1);
        if (segment.startsWith("{") && segment.endsWith("}") && segment.length() > 2) return segment.substring(1, segment.length() - 1);
        return null;
    }

    private static Map<String, String> queryMap(String query) {
        if (query == null || query.isEmpty()) return Collections.emptyMap();
        Map<String, String> out = new LinkedHashMap<>();
        for (String part : query.split("&")) {
            if (part.isEmpty()) continue;
            String[] pieces = part.split("=", 2);
            out.put(safeDecodeComponent(pieces[0]), safeDecodeComponent(pieces.length > 1 ? pieces[1] : ""));
        }
        return out;
    }

    private static String queryString(Object value) {
        if (!(value instanceof Map<?, ?>) || ((Map<?, ?>) value).isEmpty()) return "";
        List<String> parts = new ArrayList<>();
        for (Map.Entry<?, ?> entry : ((Map<?, ?>) value).entrySet()) {
            parts.add(encodeComponent(String.valueOf(entry.getKey())) + "=" + encodeComponent(entry.getValue() == null ? "" : entry.getValue().toString()));
        }
        Collections.sort(parts);
        return "?" + joinStrings("&", parts);
    }

    private static String safeDecodePathSegment(String value) {
        return safeDecodeComponent(value.replace("+", "%2B"));
    }

    private static String joinStrings(String delimiter, List<String> values) {
        StringBuilder builder = new StringBuilder();
        for (int index = 0; index < values.size(); index++) {
            if (index > 0) builder.append(delimiter);
            builder.append(values.get(index));
        }
        return builder.toString();
    }

    private static String safeDecodeComponent(String value) {
        try {
            return URLDecoder.decode(value, "UTF-8");
        } catch (Exception ignored) {
            return value;
        }
    }

    private static String encodeComponent(String value) {
        try {
            return URLEncoder.encode(value, "UTF-8").replace("+", "%20");
        } catch (Exception ignored) {
            return value;
        }
    }

    public static Entry entry(String key, Object value) {
        return new Entry(key, value);
    }

    public static Map<String, Object> record(Entry... fields) {
        Map<String, Object> out = new LinkedHashMap<>();
        for (Entry field : fields) out.put(field.key, field.value);
        return out;
    }

    public static String textValue(Object value) {
        if (value == null) return "";
        if (value instanceof Double && ((Double) value) % 1.0 == 0.0) return String.valueOf(((Double) value).longValue());
        return value.toString();
    }

    private static double numberValue(Object value) {
        if (value instanceof Number) return ((Number) value).doubleValue();
        if (value instanceof String) {
            try {
                return Double.parseDouble((String) value);
            } catch (NumberFormatException ignored) {}
        }
        return 0.0;
    }
}

final class Entry {
    final String key;
    final Object value;
    Entry(String key, Object value) {
        this.key = key;
        this.value = value;
    }
}
final class RouteMatch {
    final boolean matched;
    final String pattern;
    final Map<String, String> params;
    final int score;
    final boolean fallback;
    RouteMatch(boolean matched, String pattern, Map<String, String> params, int score, boolean fallback) {
        this.matched = matched;
        this.pattern = pattern;
        this.params = params;
        this.score = score;
        this.fallback = fallback;
    }
}
`
}

func androidNativeMainActivity(bundle irBundle, config androidTargetConfig, styles []StyleAsset) string {
	routePatterns := androidPagePaths(bundle)
	renderer := androidJavaViewRenderer{stateNames: androidStateNames(bundle.Model), styles: newAndroidStyleSheet(styles)}
	var builder strings.Builder
	builder.WriteString("package " + config.Namespace + ";\n\n")
	builder.WriteString("import static " + config.Namespace + ".NovaRuntime.*;\n\n")
	builder.WriteString("import nova.scheduler.NovaScheduler;\n")
	builder.WriteString("import nova.scheduler.NovaTransition;\n\n")
	builder.WriteString("import android.app.Activity;\n")
	builder.WriteString("import android.graphics.Color;\n")
	builder.WriteString("import android.graphics.Typeface;\n")
	builder.WriteString("import android.graphics.drawable.GradientDrawable;\n")
	builder.WriteString("import android.os.Bundle;\n")
	builder.WriteString("import android.util.TypedValue;\n")
	builder.WriteString("import android.view.Gravity;\n")
	builder.WriteString("import android.view.View;\n")
	builder.WriteString("import android.view.ViewGroup;\n")
	builder.WriteString("import android.widget.Button;\n")
	builder.WriteString("import android.widget.FrameLayout;\n")
	builder.WriteString("import android.widget.GridLayout;\n")
	builder.WriteString("import android.widget.LinearLayout;\n")
	builder.WriteString("import android.widget.TextView;\n")
	builder.WriteString("import java.util.ArrayList;\n")
	builder.WriteString("import java.util.Arrays;\n")
	builder.WriteString("import java.util.Collections;\n")
	builder.WriteString("import java.util.LinkedHashMap;\n")
	builder.WriteString("import java.util.LinkedHashSet;\n")
	builder.WriteString("import java.util.List;\n")
	builder.WriteString("import java.util.Map;\n")
	builder.WriteString("import java.util.Set;\n\n")
	builder.WriteString("public final class MainActivity extends Activity implements NovaScheduler.Host {\n")
	builder.WriteString("    private final Map<String, Object> state = new LinkedHashMap<>();\n")
	builder.WriteString("    private final Map<String, View> views = new LinkedHashMap<>();\n")
	builder.WriteString("    private final List<Object> routeBackStack = new ArrayList<>();\n")
	builder.WriteString("    private final NovaScheduler scheduler = new NovaScheduler(this);\n")
	builder.WriteString("    private final NovaPrimitiveRegistry primitiveRegistry = NovaRendererExtensions.register(new NovaPrimitiveRegistry());\n")
	builder.WriteString("    private boolean applyingSystemBack = false;\n\n")
	builder.WriteString("    @Override\n")
	builder.WriteString("    protected void onCreate(Bundle savedInstanceState) {\n")
	builder.WriteString("        super.onCreate(savedInstanceState);\n")
	builder.WriteString("        initializeState();\n")
	builder.WriteString("        initializeNavigationStack();\n")
	builder.WriteString("        setContentView(buildViewTree());\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    @Override\n")
	builder.WriteString("    public void onBackPressed() {\n")
	builder.WriteString("        if (canNavigateBack()) handleSystemBack(); else super.onBackPressed();\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    private void initializeState() {\n")
	builder.WriteString("        if (!state.isEmpty()) return;\n")
	builder.WriteString(androidJavaStateInitializers(bundle.Model))
	builder.WriteString("    }\n\n")
	builder.WriteString("    private View buildViewTree() {\n")
	builder.WriteString("        views.clear();\n")
	builder.WriteString("        LinearLayout root = new LinearLayout(this);\n")
	builder.WriteString("        root.setOrientation(LinearLayout.VERTICAL);\n")
	builder.WriteString("        root.setGravity(Gravity.CENTER);\n")
	builder.WriteString("        root.setPadding(dp(24), dp(24), dp(24), dp(24));\n")
	builder.WriteString("        root.setLayoutParams(new ViewGroup.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT));\n")
	builder.WriteString(renderer.renderBuildNodes(bundle.ViewIR.Nodes, "root", "        ", nil))
	builder.WriteString("        applyBindings(null);\n")
	builder.WriteString("        updatePageVisibility();\n")
	builder.WriteString("        return root;\n")
	builder.WriteString("    }\n\n")
	builder.WriteString(androidJavaTransitionTable(bundle.Model))
	builder.WriteString(androidJavaRoutePatternTable(routePatterns))
	builder.WriteString("    private List<String> stateNames() {\n")
	builder.WriteString("        return Arrays.asList(\n")
	for _, state := range bundle.Model.States {
		builder.WriteString("            " + quoteCodeString(state.Name) + ",\n")
	}
	builder.WriteString("            \"\"\n")
	builder.WriteString("        );\n")
	builder.WriteString("    }\n\n")
	builder.WriteString(renderer.renderApplyBindings(bundle.ViewIR.Nodes, bundle.ViewIR.Metadata.Bindings))
	builder.WriteString(renderer.renderUpdatePageVisibility(bundle.ViewIR.Nodes))
	builder.WriteString(androidJavaSchedulerActivityHost())
	builder.WriteString("    private void applyStateCommit(Set<String> invalidations) {\n")
	builder.WriteString("        if (invalidations.isEmpty()) return;\n")
	builder.WriteString("        applyBindings(invalidations);\n")
	builder.WriteString("        updatePageVisibility();\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    private boolean hasRouteState() { return state.containsKey(\"route\"); }\n\n")
	builder.WriteString("    private boolean hasTransition(String eventName) {\n")
	builder.WriteString("        for (NovaTransition transition : transitions()) {\n")
	builder.WriteString("            if (transition.eventName.equals(eventName)) return true;\n")
	builder.WriteString("        }\n")
	builder.WriteString("        return false;\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    private boolean canNavigateBack() { return routeBackStack.size() > 1; }\n\n")
	builder.WriteString("    private void handleSystemBack() {\n")
	builder.WriteString("        if (!canNavigateBack()) return;\n")
	builder.WriteString("        Object targetRoute = cloneRoute(routeBackStack.get(routeBackStack.size() - 2));\n")
	builder.WriteString("        Object beforeRoute = cloneRoute(state.get(\"route\"));\n")
	builder.WriteString("        applyingSystemBack = true;\n")
	builder.WriteString("        try {\n")
	builder.WriteString("            if (hasTransition(\"@navigate\")) {\n")
	builder.WriteString("                dispatch(\"@navigate\", Collections.singletonList(record(entry(\"kind\", \"back\"))));\n")
	builder.WriteString("            } else {\n")
	builder.WriteString("                dispatch(\"@route_changed\", Collections.singletonList(targetRoute));\n")
	builder.WriteString("            }\n")
	builder.WriteString("        } finally {\n")
	builder.WriteString("            applyingSystemBack = false;\n")
	builder.WriteString("        }\n")
	builder.WriteString("        reconcileAfterSystemBack(beforeRoute, state.get(\"route\"), targetRoute);\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    private void reconcileAfterSystemBack(Object beforeRoute, Object afterRoute, Object targetRoute) {\n")
	builder.WriteString("        if (routeKey(beforeRoute).equals(routeKey(afterRoute))) return;\n")
	builder.WriteString("        if (routeKey(afterRoute).equals(routeKey(targetRoute))) {\n")
	builder.WriteString("            routeBackStack.remove(routeBackStack.size() - 1);\n")
	builder.WriteString("            return;\n")
	builder.WriteString("        }\n")
	builder.WriteString("        routeBackStack.remove(routeBackStack.size() - 1);\n")
	builder.WriteString("        Object last = routeBackStack.isEmpty() ? null : routeBackStack.get(routeBackStack.size() - 1);\n")
	builder.WriteString("        if (!routeKey(last).equals(routeKey(afterRoute))) routeBackStack.add(cloneRoute(afterRoute));\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    private void reconcileRouteBackStack(Object beforeRoute, Object afterRoute) {\n")
	builder.WriteString("        if (!hasRouteState()) return;\n")
	builder.WriteString("        if (routeBackStack.isEmpty()) {\n")
	builder.WriteString("            routeBackStack.add(cloneRoute(afterRoute));\n")
	builder.WriteString("            return;\n")
	builder.WriteString("        }\n")
	builder.WriteString("        if (routeKey(beforeRoute).equals(routeKey(afterRoute)) || applyingSystemBack) return;\n")
	builder.WriteString("        Object last = routeBackStack.get(routeBackStack.size() - 1);\n")
	builder.WriteString("        if (!routeKey(last).equals(routeKey(afterRoute))) routeBackStack.add(cloneRoute(afterRoute));\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    private void initializeNavigationStack() {\n")
	builder.WriteString("        if (!hasRouteState() || !routeBackStack.isEmpty()) return;\n")
	builder.WriteString("        state.put(\"route\", routeValueForShape(state.get(\"route\"), routePatterns()));\n")
	builder.WriteString("        routeBackStack.add(cloneRoute(state.get(\"route\")));\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    private String activeRoutePath() { return pathOf(state.get(\"route\")); }\n\n")
	builder.WriteString("    private boolean shouldApply(Set<String> invalidations, List<String> states) {\n")
	builder.WriteString("        if (invalidations == null) return true;\n")
	builder.WriteString("        for (String state : states) {\n")
	builder.WriteString("            if (invalidations.contains(state)) return true;\n")
	builder.WriteString("        }\n")
	builder.WriteString("        return false;\n")
	builder.WriteString("    }\n\n")
	builder.WriteString("    private boolean booleanValue(Object value) { return Boolean.TRUE.equals(value) || \"true\".equals(String.valueOf(value)); }\n\n")
	builder.WriteString("    private int dp(int value) { return (int) (value * getResources().getDisplayMetrics().density); }\n")
	builder.WriteString("}\n")
	return builder.String()
}

type androidJavaViewRenderer struct {
	stateNames map[string]bool
	styles     androidStyleSheet
}

func (renderer androidJavaViewRenderer) renderBuildNodes(nodes []view.Node, parent string, indent string, path []int) string {
	var builder strings.Builder
	for index, node := range nodes {
		nodePath := append(cloneIntPath(path), index)
		builder.WriteString(renderer.renderBuildNode(node, parent, indent, nodePath))
	}
	return builder.String()
}

func (renderer androidJavaViewRenderer) renderBuildNode(node view.Node, parent string, indent string, path []int) string {
	key := androidPathKey(path)
	name := androidJavaVar("node", path)
	switch node.Kind {
	case "page":
		return renderer.renderJavaContainer(node, parent, indent, path, key, name, "FrameLayout", "")
	case "text", "#text":
		value := quoteCodeString("")
		if binding, ok := node.Props["value"]; ok {
			if expression, ok := androidJavaStringExpression(binding.Tokens, renderer.stateNames); ok {
				value = "textValue(" + expression + ")"
			}
		}
		var builder strings.Builder
		builder.WriteString(indent + "TextView " + name + " = new TextView(this);\n")
		builder.WriteString(indent + name + ".setText(" + value + ");\n")
		builder.WriteString(renderer.renderStaticStyles(node, name, indent))
		builder.WriteString(indent + "views.put(" + quoteCodeString(key) + ", " + name + ");\n")
		builder.WriteString(indent + parent + ".addView(" + name + ");\n")
		return builder.String()
	case "button":
		return renderer.renderJavaButton(node, parent, indent, path, key, name)
	case "row":
		return renderer.renderJavaRow(node, parent, indent, path, key, name)
	case "stack":
		return renderer.renderJavaContainer(node, parent, indent, path, key, name, "FrameLayout", "")
	default:
		return renderer.renderJavaContainer(node, parent, indent, path, key, name, "LinearLayout", "LinearLayout.VERTICAL")
	}
}

func (renderer androidJavaViewRenderer) renderJavaContainer(node view.Node, parent string, indent string, path []int, key string, name string, className string, orientation string) string {
	var builder strings.Builder
	builder.WriteString(indent + className + " " + name + " = new " + className + "(this);\n")
	if orientation != "" {
		builder.WriteString(indent + name + ".setOrientation(" + orientation + ");\n")
	}
	if node.Kind == "surface" || node.Kind == "column" || node.Kind == "page" {
		builder.WriteString(indent + name + ".setPadding(0, dp(4), 0, dp(4));\n")
	}
	builder.WriteString(renderer.renderStaticStyles(node, name, indent))
	builder.WriteString(indent + "views.put(" + quoteCodeString(key) + ", " + name + ");\n")
	builder.WriteString(indent + parent + ".addView(" + name + ");\n")
	builder.WriteString(renderer.renderBuildNodes(node.Children, name, indent, path))
	return builder.String()
}

func (renderer androidJavaViewRenderer) renderJavaRow(node view.Node, parent string, indent string, path []int, key string, name string) string {
	var builder strings.Builder
	builder.WriteString(indent + "GridLayout " + name + " = new GridLayout(this);\n")
	builder.WriteString(indent + name + ".setColumnCount(" + javaInt(androidRowColumnCount(node)) + ");\n")
	builder.WriteString(renderer.renderStaticStyles(node, name, indent))
	builder.WriteString(indent + "views.put(" + quoteCodeString(key) + ", " + name + ");\n")
	builder.WriteString(indent + parent + ".addView(" + name + ");\n")
	builder.WriteString(renderer.renderBuildNodes(node.Children, name, indent, path))
	return builder.String()
}

func (renderer androidJavaViewRenderer) renderJavaButton(node view.Node, parent string, indent string, path []int, key string, name string) string {
	var builder strings.Builder
	builder.WriteString(indent + "Button " + name + " = new Button(this);\n")
	builder.WriteString(indent + name + ".setAllCaps(false);\n")
	builder.WriteString(indent + name + ".setText(" + androidJavaButtonLabel(node, renderer.stateNames) + ");\n")
	if route, ok := node.Events["on_press"]; ok {
		builder.WriteString(indent + name + ".setOnClickListener(view -> dispatch(" + quoteCodeString(string(route.Event)) + ", " + androidJavaEventArgs(route.Args, renderer.stateNames) + "));\n")
	}
	builder.WriteString(renderer.renderStaticStyles(node, name, indent))
	builder.WriteString(indent + "views.put(" + quoteCodeString(key) + ", " + name + ");\n")
	builder.WriteString(indent + parent + ".addView(" + name + ");\n")
	return builder.String()
}

func androidRowColumnCount(node view.Node) int {
	count := len(node.Children)
	if count <= 0 {
		return 1
	}
	if count > 3 {
		return 3
	}
	return count
}

func (renderer androidJavaViewRenderer) renderStaticStyles(node view.Node, target string, indent string) string {
	binding, ok := node.Props["class"]
	if !ok {
		return ""
	}
	classList, ok := staticBindingString(binding)
	if !ok {
		return ""
	}
	style := renderer.styles.StyleForClassList(classList)
	if style.Empty() {
		return ""
	}
	return androidJavaStyleApplication(target, node.Kind, style, indent)
}

func androidJavaStyleApplication(target string, nodeKind string, style androidResolvedStyle, indent string) string {
	var builder strings.Builder
	if androidJavaTextStyleTarget(nodeKind) {
		if color, ok := androidCSSColorProperty(style, "color"); ok {
			builder.WriteString(indent + target + ".setTextColor(" + color + ");\n")
		}
		if size, ok := androidCSSIntProperty(style, "font-size"); ok {
			builder.WriteString(indent + target + ".setTextSize(TypedValue.COMPLEX_UNIT_SP, " + javaInt(size) + ");\n")
		}
		if weight, ok := style.Value("font-weight"); ok && androidCSSBoldWeight(weight) {
			builder.WriteString(indent + target + ".setTypeface(Typeface.DEFAULT, Typeface.BOLD);\n")
		}
		if transform, ok := style.Value("text-transform"); ok && strings.EqualFold(transform, "uppercase") {
			builder.WriteString(indent + target + ".setAllCaps(true);\n")
		}
		if align, ok := style.Value("text-align"); ok && strings.EqualFold(align, "center") {
			builder.WriteString(indent + target + ".setGravity(Gravity.CENTER);\n")
		}
		if lineHeight, ok := androidCSSLineHeight(style); ok {
			builder.WriteString(indent + target + ".setLineSpacing(0f, " + lineHeight + "f);\n")
		}
	}
	if padding, ok := androidCSSBoxProperty(style, "padding"); ok {
		builder.WriteString(indent + target + ".setPadding(dp(" + javaInt(padding[3]) + "), dp(" + javaInt(padding[0]) + "), dp(" + javaInt(padding[1]) + "), dp(" + javaInt(padding[2]) + "));\n")
	}
	if minHeight, ok := androidCSSIntProperty(style, "min-height"); ok {
		builder.WriteString(indent + target + ".setMinimumHeight(dp(" + javaInt(minHeight) + "));\n")
	}
	if alignContent, ok := style.Value("align-content"); ok && strings.EqualFold(alignContent, "center") && androidJavaLinearStyleTarget(nodeKind) {
		builder.WriteString(indent + target + ".setGravity(Gravity.CENTER_VERTICAL);\n")
	}
	if background := androidJavaBackgroundDrawable(target, style, indent); background != "" {
		builder.WriteString(background)
	}
	return builder.String()
}

func androidJavaTextStyleTarget(kind string) bool {
	return kind == "text" || kind == "#text" || kind == "button"
}

func androidJavaLinearStyleTarget(kind string) bool {
	return kind == "surface" || kind == "column"
}

func androidJavaBackgroundDrawable(target string, style androidResolvedStyle, indent string) string {
	background, hasBackground := androidCSSColorProperty(style, "background-color", "background")
	borderColor, hasBorderColor := androidCSSBorderColor(style)
	borderWidth, hasBorderWidth := androidCSSBorderWidth(style)
	radius, hasRadius := androidCSSIntProperty(style, "border-radius")
	if !hasBackground && !hasBorderColor && !hasBorderWidth && !hasRadius {
		return ""
	}
	if !hasBackground {
		background = "Color.TRANSPARENT"
	}
	if !hasBorderColor {
		borderColor = "Color.TRANSPARENT"
	}
	if !hasBorderWidth {
		borderWidth = 0
	}
	styleVar := target + "Style"
	var builder strings.Builder
	builder.WriteString(indent + "GradientDrawable " + styleVar + " = new GradientDrawable();\n")
	builder.WriteString(indent + styleVar + ".setColor(" + background + ");\n")
	if hasBorderColor || hasBorderWidth {
		builder.WriteString(indent + styleVar + ".setStroke(dp(" + javaInt(borderWidth) + "), " + borderColor + ");\n")
	}
	if hasRadius {
		builder.WriteString(indent + styleVar + ".setCornerRadius(dp(" + javaInt(radius) + "));\n")
	}
	builder.WriteString(indent + target + ".setBackground(" + styleVar + ");\n")
	return builder.String()
}

func (renderer androidJavaViewRenderer) renderApplyBindings(nodes []view.Node, bindings []view.BindingRef) string {
	var builder strings.Builder
	builder.WriteString("    private void applyBindings(Set<String> invalidations) {\n")
	for _, binding := range bindings {
		if binding.Prop == "key" || strings.Contains(binding.Prop, "#arg") {
			continue
		}
		node, ok := nodeAtPath(nodes, binding.NodePath)
		if !ok {
			continue
		}
		expression, ok := androidJavaBindingExpression(node, binding.Prop, renderer.stateNames)
		if !ok {
			continue
		}
		pathKey := androidPathKey(binding.NodePath)
		states := androidJavaStringList(binding.States)
		builder.WriteString("        if (shouldApply(invalidations, " + states + ")) {\n")
		builder.WriteString("            View target = views.get(" + quoteCodeString(pathKey) + ");\n")
		builder.WriteString("            if (target != null) {\n")
		switch {
		case binding.Prop == "value" && (node.Kind == "text" || node.Kind == "#text"):
			builder.WriteString("                ((TextView) target).setText(textValue(" + expression + "));\n")
		case binding.Prop == "enabled":
			builder.WriteString("                target.setEnabled(booleanValue(" + expression + "));\n")
		case binding.Prop == "label":
			builder.WriteString("                target.setContentDescription(textValue(" + expression + "));\n")
		default:
			builder.WriteString("                target.setTag(textValue(" + expression + "));\n")
		}
		builder.WriteString("            }\n")
		builder.WriteString("        }\n")
	}
	builder.WriteString("    }\n\n")
	return builder.String()
}

func (renderer androidJavaViewRenderer) renderUpdatePageVisibility(nodes []view.Node) string {
	var builder strings.Builder
	builder.WriteString("    private void updatePageVisibility() {\n")
	builder.WriteString("        String activePath = activeRoutePath();\n")
	builder.WriteString("        int selectedRouteScore = bestRouteScore(routePatterns(), activePath);\n")
	renderer.appendPageVisibility(&builder, nodes, nil)
	builder.WriteString("    }\n\n")
	return builder.String()
}

func (renderer androidJavaViewRenderer) appendPageVisibility(builder *strings.Builder, nodes []view.Node, path []int) {
	for index, node := range nodes {
		nodePath := append(cloneIntPath(path), index)
		if node.Kind == "page" {
			expression, ok := androidJavaValueExpression(node.Props["path"].Tokens, renderer.stateNames)
			if !ok {
				expression = quoteCodeString("/")
			}
			key := androidPathKey(nodePath)
			scoreVar := "pageScore" + androidJavaPathSuffix(nodePath)
			builder.WriteString("        View page" + androidJavaPathSuffix(nodePath) + " = views.get(" + quoteCodeString(key) + ");\n")
			builder.WriteString("        int " + scoreVar + " = routeMatchScore(textValue(" + expression + "), activePath);\n")
			builder.WriteString("        if (page" + androidJavaPathSuffix(nodePath) + " != null) page" + androidJavaPathSuffix(nodePath) + ".setVisibility(" + scoreVar + " >= 0 && " + scoreVar + " == selectedRouteScore ? View.VISIBLE : View.GONE);\n")
		}
		renderer.appendPageVisibility(builder, node.Children, nodePath)
	}
}

func androidJavaBindingExpression(node view.Node, prop string, stateNames map[string]bool) (string, bool) {
	binding, ok := node.Props[prop]
	if !ok {
		return "", false
	}
	if prop == "value" || prop == "label" {
		return androidJavaStringExpression(binding.Tokens, stateNames)
	}
	return androidJavaValueExpression(binding.Tokens, stateNames)
}

func androidJavaStateInitializers(model appModel) string {
	var builder strings.Builder
	for _, state := range model.States {
		builder.WriteString("        state.put(" + quoteCodeString(state.Name) + ", " + androidJavaInitialValue(state.Initial) + ");\n")
	}
	return builder.String()
}

func androidJavaTransitionTable(model appModel) string {
	var builder strings.Builder
	builder.WriteString("    private List<NovaTransition> transitions() {\n")
	builder.WriteString("        return Arrays.asList(\n")
	for _, state := range model.States {
		for _, transition := range state.Transitions {
			builder.WriteString("            new NovaTransition(")
			builder.WriteString(quoteCodeString(state.Name))
			builder.WriteString(", ")
			builder.WriteString(quoteCodeString(transition.Event))
			builder.WriteString(", ")
			builder.WriteString(androidJavaStringList(transition.Params))
			builder.WriteString(", ")
			builder.WriteString(quoteCodeString(transition.Expression))
			builder.WriteString("),\n")
		}
	}
	builder.WriteString("            new NovaTransition(\"\", \"\", Collections.emptyList(), \"\")\n")
	builder.WriteString("        );\n")
	builder.WriteString("    }\n\n")
	return builder.String()
}

func androidJavaRoutePatternTable(patterns []string) string {
	var builder strings.Builder
	builder.WriteString("    private List<String> routePatterns() {\n")
	if len(patterns) == 0 {
		builder.WriteString("        return Collections.emptyList();\n")
	} else {
		builder.WriteString("        return Arrays.asList(\n")
		for _, pattern := range patterns {
			builder.WriteString("            " + quoteCodeString(pattern) + ",\n")
		}
		builder.WriteString("            \"\"\n")
		builder.WriteString("        );\n")
	}
	builder.WriteString("    }\n\n")
	return builder.String()
}

func androidJavaInitialValue(expression string) string {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return "null"
	}
	if strings.HasPrefix(expression, "({") && strings.HasSuffix(expression, "})") {
		return androidJavaInitialRecord(expression)
	}
	if strings.HasPrefix(expression, "\"") && strings.HasSuffix(expression, "\"") {
		return quoteCodeString(strings.Trim(expression, "\""))
	}
	if _, err := strconv.ParseFloat(expression, 64); err == nil {
		if strings.Contains(expression, ".") {
			return expression
		}
		return expression + ".0"
	}
	switch expression {
	case "true":
		return "Boolean.TRUE"
	case "false":
		return "Boolean.FALSE"
	case "null", "undefined":
		return "null"
	default:
		return "null"
	}
}

func androidJavaInitialRecord(expression string) string {
	body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(expression, "({"), "})"))
	if body == "" {
		return "record()"
	}
	fields := splitAndroidRecordFields(body)
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		name, value, ok := strings.Cut(field, ":")
		if !ok {
			continue
		}
		parts = append(parts, "entry("+quoteCodeString(strings.TrimSpace(name))+", "+androidJavaInitialValue(strings.TrimSpace(value))+")")
	}
	if len(parts) == 0 {
		return "record()"
	}
	return "record(" + strings.Join(parts, ", ") + ")"
}

func splitAndroidRecordFields(body string) []string {
	fields := make([]string, 0)
	start := 0
	depth := 0
	inString := false
	escaped := false
	for index, ch := range body {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch ch {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				fields = append(fields, strings.TrimSpace(body[start:index]))
				start = index + 1
			}
		}
	}
	fields = append(fields, strings.TrimSpace(body[start:]))
	return fields
}

func androidJavaStringExpression(tokens []lexer.Token, stateNames map[string]bool) (string, bool) {
	tokens = trimExpressionTokens(tokens)
	parts := splitAndroidTopLevel(tokens, lexer.PLUS)
	if len(parts) > 1 {
		expressions := make([]string, 0, len(parts))
		for _, part := range parts {
			expression, ok := androidJavaStringExpression(part, stateNames)
			if !ok {
				return "", false
			}
			expressions = append(expressions, "textValue("+expression+")")
		}
		return strings.Join(expressions, " + "), true
	}
	if len(tokens) == 1 && tokens[0].Type == lexer.STRING {
		return quoteCodeString(tokens[0].Literal), true
	}
	if value, ok := androidJavaValueExpression(tokens, stateNames); ok {
		return value, true
	}
	return quoteCodeString(androidJoinTokenLiterals(tokens)), true
}

func androidJavaValueExpression(tokens []lexer.Token, stateNames map[string]bool) (string, bool) {
	tokens = trimExpressionTokens(tokens)
	if len(tokens) == 0 {
		return "null", true
	}
	if fields, ok := parseRecordExpressionFields(tokens); ok {
		parts := make([]string, 0, len(fields))
		for _, field := range fields {
			value, ok := androidJavaValueExpression(field.Value, stateNames)
			if !ok {
				return "", false
			}
			parts = append(parts, "entry("+quoteCodeString(field.Name.Literal)+", "+value+")")
		}
		return "record(" + strings.Join(parts, ", ") + ")", true
	}
	if len(tokens) == 1 {
		switch tokens[0].Type {
		case lexer.STRING:
			return quoteCodeString(tokens[0].Literal), true
		case lexer.NUMBER:
			if strings.Contains(tokens[0].Literal, ".") {
				return tokens[0].Literal, true
			}
			return tokens[0].Literal + ".0", true
		case lexer.TRUE:
			return "Boolean.TRUE", true
		case lexer.FALSE:
			return "Boolean.FALSE", true
		case lexer.NULL, lexer.VOID:
			return "null", true
		case lexer.IDENT:
			if stateNames[tokens[0].Literal] {
				return "state.get(" + quoteCodeString(tokens[0].Literal) + ")", true
			}
		}
	}
	if isRoutePathExpression(tokens) {
		return "pathOf(state.get(\"route\"))", true
	}
	return "", false
}

func androidJavaEventArgs(args []view.Binding, stateNames map[string]bool) string {
	if len(args) == 0 {
		return "Collections.emptyList()"
	}
	values := make([]string, 0, len(args))
	for _, arg := range args {
		value, ok := androidJavaValueExpression(arg.Tokens, stateNames)
		if !ok {
			value = "null"
		}
		values = append(values, value)
	}
	return "Arrays.<Object>asList(" + strings.Join(values, ", ") + ")"
}

func androidJavaButtonLabel(node view.Node, stateNames map[string]bool) string {
	for _, child := range node.Children {
		if child.Kind == "text" || child.Kind == "#text" {
			if value, ok := child.Props["value"]; ok {
				if expression, ok := androidJavaStringExpression(value.Tokens, stateNames); ok {
					return "textValue(" + expression + ")"
				}
			}
		}
	}
	if label, ok := node.Props["label"]; ok {
		if expression, ok := androidJavaStringExpression(label.Tokens, stateNames); ok {
			return "textValue(" + expression + ")"
		}
	}
	return quoteCodeString("Button")
}

func androidJavaStringList(values []string) string {
	if len(values) == 0 {
		return "Collections.emptyList()"
	}
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, quoteCodeString(value))
	}
	return "Arrays.asList(" + strings.Join(quoted, ", ") + ")"
}

func androidCSSColorProperty(style androidResolvedStyle, names ...string) (string, bool) {
	for _, name := range names {
		value, ok := style.Value(name)
		if !ok {
			continue
		}
		if color, ok := androidCSSColor(value); ok {
			return color, true
		}
	}
	return "", false
}

func androidCSSColor(value string) (string, bool) {
	for _, field := range strings.Fields(strings.TrimSpace(value)) {
		cleaned := strings.Trim(field, ",")
		if strings.HasPrefix(cleaned, "#") || androidCSSNamedColor(cleaned) {
			return "Color.parseColor(" + quoteCodeString(cleaned) + ")", true
		}
	}
	return "", false
}

func androidCSSNamedColor(value string) bool {
	switch strings.ToLower(value) {
	case "black", "blue", "cyan", "darkgray", "gray", "green", "lightgray", "magenta", "red", "white", "yellow":
		return true
	default:
		return false
	}
}

func androidCSSIntProperty(style androidResolvedStyle, name string) (int, bool) {
	value, ok := style.Value(name)
	if !ok {
		return 0, false
	}
	return androidCSSInt(value)
}

func androidCSSInt(value string) (int, bool) {
	cleaned := strings.TrimSpace(value)
	for _, suffix := range []string{"px", "dp", "sp"} {
		cleaned = strings.TrimSuffix(cleaned, suffix)
	}
	number, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, false
	}
	return int(number + 0.5), true
}

func androidCSSBoxProperty(style androidResolvedStyle, name string) ([4]int, bool) {
	value, ok := style.Value(name)
	if !ok {
		return [4]int{}, false
	}
	fields := strings.Fields(value)
	if len(fields) == 0 || len(fields) > 4 {
		return [4]int{}, false
	}
	values := make([]int, 0, len(fields))
	for _, field := range fields {
		size, ok := androidCSSInt(field)
		if !ok {
			return [4]int{}, false
		}
		values = append(values, size)
	}
	switch len(values) {
	case 1:
		return [4]int{values[0], values[0], values[0], values[0]}, true
	case 2:
		return [4]int{values[0], values[1], values[0], values[1]}, true
	case 3:
		return [4]int{values[0], values[1], values[2], values[1]}, true
	default:
		return [4]int{values[0], values[1], values[2], values[3]}, true
	}
}

func androidCSSBorderWidth(style androidResolvedStyle) (int, bool) {
	if width, ok := androidCSSIntProperty(style, "border-width"); ok {
		return width, true
	}
	value, ok := style.Value("border")
	if !ok {
		return 0, false
	}
	for _, field := range strings.Fields(value) {
		if width, ok := androidCSSInt(field); ok {
			return width, true
		}
	}
	return 0, false
}

func androidCSSBorderColor(style androidResolvedStyle) (string, bool) {
	if color, ok := androidCSSColorProperty(style, "border-color"); ok {
		return color, true
	}
	value, ok := style.Value("border")
	if !ok {
		return "", false
	}
	return androidCSSColor(value)
}

func androidCSSBoldWeight(value string) bool {
	cleaned := strings.TrimSpace(strings.ToLower(value))
	if cleaned == "bold" || cleaned == "bolder" {
		return true
	}
	weight, err := strconv.Atoi(cleaned)
	return err == nil && weight >= 600
}

func androidCSSLineHeight(style androidResolvedStyle) (string, bool) {
	value, ok := style.Value("line-height")
	if !ok {
		return "", false
	}
	cleaned := strings.TrimSpace(strings.TrimSuffix(value, "em"))
	if strings.HasSuffix(cleaned, "px") {
		return "", false
	}
	number, err := strconv.ParseFloat(cleaned, 64)
	if err != nil || number <= 0 {
		return "", false
	}
	return strconv.FormatFloat(number, 'f', -1, 64), true
}

func nodeAtPath(nodes []view.Node, path []int) (view.Node, bool) {
	list := nodes
	var node view.Node
	for _, index := range path {
		if index < 0 || index >= len(list) {
			return view.Node{}, false
		}
		node = list[index]
		list = node.Children
	}
	return node, true
}

func androidPathKey(path []int) string {
	parts := make([]string, 0, len(path))
	for _, value := range path {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, ".")
}

func androidJavaVar(prefix string, path []int) string {
	suffix := androidJavaPathSuffix(path)
	if suffix == "" {
		return prefix
	}
	return prefix + suffix
}

func androidJavaPathSuffix(path []int) string {
	if len(path) == 0 {
		return ""
	}
	parts := make([]string, 0, len(path))
	for _, value := range path {
		parts = append(parts, strconv.Itoa(value))
	}
	return "_" + strings.Join(parts, "_")
}

func androidJoinTokenLiterals(tokens []lexer.Token) string {
	parts := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if strings.TrimSpace(token.Literal) == "" {
			continue
		}
		parts = append(parts, token.Literal)
	}
	return strings.Join(parts, " ")
}

func androidApp(name string, target string, config androidTargetConfig) string {
	if strings.TrimSpace(name) == "" {
		name = "NovaApp"
	}
	return "package " + config.Namespace + ";\n\npublic final class NovaApp {\n    public final String name = " + quoteCodeString(name) + ";\n    public final String target = " + quoteCodeString(target) + ";\n}\n"
}

func androidRoutes(bundle irBundle, config androidTargetConfig) string {
	paths := androidPagePaths(bundle)
	if len(paths) == 0 {
		paths = []string{"/"}
	}
	names := make(map[string]int)
	var builder strings.Builder
	builder.WriteString("package " + config.Namespace + ";\n\npublic final class NovaRoutes {\n    private NovaRoutes() {}\n")
	for _, path := range paths {
		baseName := androidRouteConstName(path)
		names[baseName]++
		name := baseName
		if names[baseName] > 1 {
			name = baseName + "_" + javaInt(names[baseName])
		}
		builder.WriteString("    public static final String ")
		builder.WriteString(strings.ToUpper(name))
		builder.WriteString(" = ")
		builder.WriteString(quoteCodeString(path))
		builder.WriteString(";\n")
	}
	builder.WriteString("}\n")
	return builder.String()
}

func androidExternalBindings(operations []build.ResolvedExternalOperation, config androidTargetConfig) string {
	names := externalOperationNames(operations)
	var builder strings.Builder
	builder.WriteString("package " + config.Namespace + ";\n\nimport java.util.Arrays;\nimport java.util.List;\n\npublic final class NovaExternalBindings {\n    private NovaExternalBindings() {}\n    public static final List<String> OPERATIONS = Arrays.asList(\n")
	for _, name := range names {
		builder.WriteString("        ")
		builder.WriteString(quoteCodeString(name))
		builder.WriteString(",\n")
	}
	builder.WriteString("        \"\"\n    );\n}\n")
	return builder.String()
}

func androidPagePaths(bundle irBundle) []string {
	seen := make(map[string]bool)
	paths := make([]string, 0, len(bundle.ViewIR.Metadata.Pages))
	for _, page := range bundle.ViewIR.Metadata.Pages {
		path, ok := staticBindingString(page.Path)
		if !ok {
			continue
		}
		path = routing.DescribePattern(path).Pattern
		if seen[path] {
			continue
		}
		seen[path] = true
		paths = append(paths, path)
	}
	return paths
}

func staticBindingString(binding view.Binding) (string, bool) {
	tokens := trimExpressionTokens(binding.Tokens)
	if len(tokens) == 1 && tokens[0].Type == lexer.STRING {
		return tokens[0].Literal, true
	}
	if binding.Text != "" {
		return binding.Text, true
	}
	return "", false
}

func androidStateNames(model appModel) map[string]bool {
	names := make(map[string]bool, len(model.States))
	for _, state := range model.States {
		names[state.Name] = true
	}
	return names
}

func splitAndroidTopLevel(tokens []lexer.Token, delimiter lexer.TokenType) [][]lexer.Token {
	segments := make([][]lexer.Token, 0)
	start := 0
	depth := 0
	for i, tok := range tokens {
		if depth == 0 && tok.Type == delimiter {
			segments = append(segments, trimExpressionTokens(tokens[start:i]))
			start = i + 1
			continue
		}
		depth = expressionDepth(depth, tok.Type)
	}
	if start == 0 {
		return nil
	}
	segments = append(segments, trimExpressionTokens(tokens[start:]))
	return segments
}

func isRoutePathExpression(tokens []lexer.Token) bool {
	return len(tokens) == 3 &&
		tokens[0].Type == lexer.IDENT && tokens[0].Literal == "route" &&
		tokens[1].Type == lexer.DOT &&
		tokens[2].Type == lexer.IDENT && tokens[2].Literal == "path"
}

func androidRouteConstName(path string) string {
	if strings.TrimSpace(path) == "*" || strings.TrimSpace(path) == "/*" {
		return "fallback"
	}
	name := strings.Trim(strings.ToLower(path), "/")
	if name == "" {
		return "root"
	}
	var builder strings.Builder
	lastUnderscore := false
	for _, ch := range name {
		switch {
		case ch >= 'a' && ch <= 'z':
			builder.WriteRune(ch)
			lastUnderscore = false
		case ch >= '0' && ch <= '9':
			builder.WriteRune(ch)
			lastUnderscore = false
		default:
			if !lastUnderscore {
				builder.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	out := strings.Trim(builder.String(), "_")
	if out == "" {
		return "route"
	}
	if out[0] >= '0' && out[0] <= '9' {
		return "route_" + out
	}
	return out
}

func javaInt(value int) string {
	return strconv.Itoa(value)
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
