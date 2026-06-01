package com.nova.env;

import android.content.Context;
import java.util.LinkedHashMap;
import java.util.Map;

public final class DspAdapter {
    private static final Map<String, Object> PROFILE = new LinkedHashMap<>();

    private DspAdapter() {}

    public static Object invoke(Context context, String operation, Map<String, Object> input) {
        switch (operation) {
            case "setProfile": {
                PROFILE.clear();
                PROFILE.put("bass", numberValue(input.get("bass")));
                PROFILE.put("mid", numberValue(input.get("mid")));
                PROFILE.put("treble", numberValue(input.get("treble")));
                PROFILE.put("masterGain", numberValue(input.get("masterGain")));
                return null;
            }
            case "reset": {
                PROFILE.clear();
                return null;
            }
            default:
                throw new IllegalStateException("unsupported dsp operation " + operation);
        }
    }

    private static double numberValue(Object value) {
        if (value instanceof Number) {
            return ((Number) value).doubleValue();
        }
        try {
            return Double.parseDouble(String.valueOf(value));
        } catch (NumberFormatException error) {
            return 0.0;
        }
    }
}
