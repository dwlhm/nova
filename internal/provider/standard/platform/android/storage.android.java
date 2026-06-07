package com.nova.env;

import android.content.Context;
import android.content.SharedPreferences;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import org.json.JSONArray;
import org.json.JSONObject;
import org.json.JSONTokener;

public final class StorageAdapter {
    private StorageAdapter() {}

    public static Object invoke(Context context, String operation, Map<String, Object> input) {
        SharedPreferences prefs = context.getSharedPreferences("nova", Context.MODE_PRIVATE);
        switch (operation) {
            case "load":
            case "get": {
                String raw = prefs.getString(String.valueOf(input.get("key")), null);
                return decodeValue(raw);
            }
            case "set": {
                prefs.edit()
                    .putString(String.valueOf(input.get("key")), encodeValue(input.get("value")))
                    .commit();
                return null;
            }
            case "remove": {
                prefs.edit().remove(String.valueOf(input.get("key"))).commit();
                return null;
            }
            case "clear": {
                prefs.edit().clear().commit();
                return null;
            }
            default:
                throw new IllegalStateException("unsupported storage operation " + operation);
        }
    }

    private static String encodeValue(Object value) {
        if (value == null) {
            return "null";
        }
        if (value instanceof String) {
            return "\"" + escapeJsonString((String) value) + "\"";
        }
        if (value instanceof Boolean || value instanceof Number) {
            return String.valueOf(value);
        }
        if (value instanceof Map<?, ?>) {
            return encodeObject((Map<?, ?>) value).toString();
        }
        if (value instanceof List<?>) {
            return encodeArray((List<?>) value).toString();
        }
        return "\"" + escapeJsonString(String.valueOf(value)) + "\"";
    }

    private static JSONObject encodeObject(Map<?, ?> map) {
        JSONObject out = new JSONObject();
        try {
            for (Map.Entry<?, ?> entry : map.entrySet()) {
                out.put(String.valueOf(entry.getKey()), encodeJsonValue(entry.getValue()));
            }
        } catch (Exception error) {
            throw new IllegalStateException("failed to encode storage object", error);
        }
        return out;
    }

    private static JSONArray encodeArray(List<?> list) {
        JSONArray out = new JSONArray();
        try {
            for (Object item : list) {
                out.put(encodeJsonValue(item));
            }
        } catch (Exception error) {
            throw new IllegalStateException("failed to encode storage array", error);
        }
        return out;
    }

    private static Object encodeJsonValue(Object value) {
        if (value == null) {
            return JSONObject.NULL;
        }
        if (value instanceof Boolean || value instanceof Number || value instanceof String) {
            return value;
        }
        if (value instanceof Map<?, ?>) {
            return encodeObject((Map<?, ?>) value);
        }
        if (value instanceof List<?>) {
            return encodeArray((List<?>) value);
        }
        return String.valueOf(value);
    }

    private static Object decodeValue(String raw) {
        if (raw == null) {
            return null;
        }
        try {
            Object parsed = new JSONTokener(raw).nextValue();
            return decodeJsonValue(parsed);
        } catch (Exception ignored) {
            return raw;
        }
    }

    private static Object decodeJsonValue(Object value) {
        if (value == null || value == JSONObject.NULL) {
            return null;
        }
        if (value instanceof JSONObject) {
            return decodeObject((JSONObject) value);
        }
        if (value instanceof JSONArray) {
            return decodeArray((JSONArray) value);
        }
        return value;
    }

    private static Map<String, Object> decodeObject(JSONObject object) {
        Map<String, Object> out = new LinkedHashMap<>();
        JSONArray names = object.names();
        if (names == null) {
            return out;
        }
        for (int index = 0; index < names.length(); index++) {
            String key = names.optString(index, "");
            out.put(key, decodeJsonValue(object.opt(key)));
        }
        return out;
    }

    private static List<Object> decodeArray(JSONArray array) {
        List<Object> out = new ArrayList<>(array.length());
        for (int index = 0; index < array.length(); index++) {
            out.add(decodeJsonValue(array.opt(index)));
        }
        return out;
    }

    private static String escapeJsonString(String value) {
        return value
            .replace("\\", "\\\\")
            .replace("\"", "\\\"")
            .replace("\n", "\\n")
            .replace("\r", "\\r")
            .replace("\t", "\\t");
    }
}
