package com.nova.env;

import android.content.Context;
import android.content.SharedPreferences;
import java.util.Map;
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
                    .apply();
                return null;
            }
            case "remove": {
                prefs.edit().remove(String.valueOf(input.get("key"))).apply();
                return null;
            }
            case "clear": {
                prefs.edit().clear().apply();
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
        return "\"" + escapeJsonString(String.valueOf(value)) + "\"";
    }

    private static Object decodeValue(String raw) {
        if (raw == null) {
            return null;
        }
        try {
            return new JSONTokener(raw).nextValue();
        } catch (Exception ignored) {
            return raw;
        }
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
