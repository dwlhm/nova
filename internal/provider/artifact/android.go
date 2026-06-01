package artifact

import (
	"strconv"
	"strings"

	"github.com/dwlhm/nova/internal/core/contract"
	"github.com/dwlhm/nova/internal/core/diagnostic"
	"github.com/dwlhm/nova/internal/core/security"
	"github.com/dwlhm/nova/internal/project"
	"github.com/dwlhm/nova/internal/provider/build"
	"github.com/dwlhm/nova/internal/provider/target"
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

func androidManifest(config androidTargetConfig, permissions []security.Permission) string {
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

func androidStyles(config androidTargetConfig) string {
	return "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<resources>\n    <style name=\"" + escapeXML(config.Theme) + "\" parent=\"" + escapeXML(config.ThemeParent) + "\">\n        <item name=\"android:windowActionBar\">false</item>\n        <item name=\"android:windowNoTitle\">true</item>\n    </style>\n</resources>\n"
}

func androidMainActivity(app contract.App, config androidTargetConfig, styles []StyleAsset) string {
	return androidMainActivityFromContract(app, config, styles)
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
            return unescapeStringLiteral(expression.substring(1, expression.length() - 1));
        }
        try {
            if (!expression.isEmpty()) return Double.parseDouble(expression);
        } catch (NumberFormatException ignored) {}
        if ("true".equals(expression)) return Boolean.TRUE;
        if ("false".equals(expression)) return Boolean.FALSE;
        if ("null".equals(expression) || "undefined".equals(expression) || expression.isEmpty()) return null;
        return expression;
    }

    private static String unescapeStringLiteral(String value) {
        StringBuilder out = new StringBuilder();
        boolean escaped = false;
        for (int index = 0; index < value.length(); index++) {
            char ch = value.charAt(index);
            if (escaped) {
                switch (ch) {
                    case 'n': out.append('\n'); break;
                    case 'r': out.append('\r'); break;
                    case 't': out.append('\t'); break;
                    case '"': out.append('"'); break;
                    case '\\': out.append('\\'); break;
                    default: out.append(ch); break;
                }
                escaped = false;
                continue;
            }
            if (ch == '\\') {
                escaped = true;
                continue;
            }
            out.append(ch);
        }
        if (escaped) out.append('\\');
        return out.toString();
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
	return kind == "text" || kind == "#text" || kind == "button" || kind == "text_input" || kind == "number_input"
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
		return quoteCodeString(androidJavaStringLiteralValue(expression))
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

func androidJavaStringLiteralValue(expression string) string {
	value, err := strconv.Unquote(expression)
	if err != nil {
		return strings.Trim(expression, "\"")
	}
	return value
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
