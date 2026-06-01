package com.nova.env;

import android.content.Context;
import android.os.Build;
import java.util.LinkedHashMap;
import java.util.Map;

public final class DeviceAdapter {
    private DeviceAdapter() {}

    public static Object invoke(Context context, String operation, Map<String, Object> input) {
        if (!"info".equals(operation)) {
            throw new IllegalStateException("unsupported device operation " + operation);
        }
        Map<String, Object> result = new LinkedHashMap<>();
        result.put("userAgent", "NovaAndroid/" + Build.VERSION.RELEASE);
        result.put("platform", "android");
        result.put("model", Build.MODEL);
        result.put("manufacturer", Build.MANUFACTURER);
        result.put("language", context.getResources().getConfiguration().locale.toLanguageTag());
        return result;
    }
}
