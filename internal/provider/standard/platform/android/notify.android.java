package com.nova.env;

import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.content.Context;
import android.os.Build;
import java.util.Map;

public final class NotifyAdapter {
    private static final String CHANNEL_ID = "nova_notifications";

    private NotifyAdapter() {}

    public static Object invoke(Context context, String operation, Map<String, Object> input) {
        if (!"send".equals(operation)) {
            throw new IllegalStateException("unsupported notify operation " + operation);
        }
        String message = input.get("msg") == null ? "" : String.valueOf(input.get("msg"));
        if (message.isEmpty()) {
            message = input.get("message") == null ? "" : String.valueOf(input.get("message"));
        }
        if (message.isEmpty()) {
            throw new IllegalStateException("notify.send requires msg");
        }
        NotificationManager manager = (NotificationManager) context.getSystemService(Context.NOTIFICATION_SERVICE);
        if (manager == null) {
            throw new IllegalStateException("notification service unavailable");
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            NotificationChannel channel = new NotificationChannel(CHANNEL_ID, "Nova", NotificationManager.IMPORTANCE_DEFAULT);
            manager.createNotificationChannel(channel);
        }
        android.app.Notification.Builder builder = Build.VERSION.SDK_INT >= Build.VERSION_CODES.O
            ? new android.app.Notification.Builder(context, CHANNEL_ID)
            : new android.app.Notification.Builder(context);
        builder.setContentTitle("Nova")
            .setContentText(message)
            .setSmallIcon(android.R.drawable.ic_dialog_info);
        manager.notify((int) System.currentTimeMillis(), builder.build());
        return null;
    }
}
