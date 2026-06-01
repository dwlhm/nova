package com.nova.env;

import android.content.Context;
import java.util.Map;

public final class OsAdapter {
    private OsAdapter() {}

    public static Object invoke(Context context, String operation, Map<String, Object> input) {
        if (!"send".equals(operation)) {
            throw new IllegalStateException("unsupported os operation " + operation);
        }
        return NotifyAdapter.invoke(context, "send", input);
    }
}
