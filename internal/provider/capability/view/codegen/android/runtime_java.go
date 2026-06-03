package androidcodegen

import (
	androidtarget "github.com/dwlhm/nova/internal/provider/capability/view/target/android"
)

func JavaRuntime(config androidtarget.Config) string {
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
