package com.nova.env;

import android.content.ClipData;
import android.content.ClipboardManager;
import android.content.Context;
import java.util.Map;

public final class ClipboardAdapter {
    private ClipboardAdapter() {}

    public static Object invoke(Context context, String operation, Map<String, Object> input) {
        ClipboardManager clipboard = (ClipboardManager) context.getSystemService(Context.CLIPBOARD_SERVICE);
        if (clipboard == null) {
            throw new IllegalStateException("clipboard service unavailable");
        }
        switch (operation) {
            case "read": {
                if (!clipboard.hasPrimaryClip() || clipboard.getPrimaryClip() == null || clipboard.getPrimaryClip().getItemCount() == 0) {
                    return "";
                }
                CharSequence text = clipboard.getPrimaryClip().getItemAt(0).getText();
                return text == null ? "" : text.toString();
            }
            case "write": {
                String value = input.get("value") == null ? "" : String.valueOf(input.get("value"));
                clipboard.setPrimaryClip(ClipData.newPlainText("nova", value));
                return null;
            }
            default:
                throw new IllegalStateException("unsupported clipboard operation " + operation);
        }
    }
}
