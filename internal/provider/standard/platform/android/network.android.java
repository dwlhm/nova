package com.nova.env;

import android.content.Context;
import java.io.BufferedReader;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.LinkedHashMap;
import java.util.Map;

public final class NetworkAdapter {
    private NetworkAdapter() {}

    public static Object invoke(Context context, String operation, Map<String, Object> input) {
        if (!"request".equals(operation)) {
            throw new IllegalStateException("unsupported network operation " + operation);
        }
        Map<String, Object> payload = payloadMap(input.get("input"));
        String urlValue = stringValue(payload.get("url"));
        if (urlValue.isEmpty()) {
            urlValue = stringValue(payload.get("href"));
        }
        if (urlValue.isEmpty()) {
            throw new IllegalStateException("network.request requires url");
        }
        String method = stringValue(payload.get("method"));
        if (method.isEmpty()) {
            method = "GET";
        }
        HttpURLConnection connection = null;
        try {
            connection = (HttpURLConnection) new URL(urlValue).openConnection();
            connection.setRequestMethod(method.toUpperCase());
            connection.setConnectTimeout(15000);
            connection.setReadTimeout(15000);
            Object body = payload.get("body");
            if (body != null && !"GET".equalsIgnoreCase(method) && !"HEAD".equalsIgnoreCase(method)) {
                connection.setDoOutput(true);
                byte[] bytes = String.valueOf(body).getBytes("UTF-8");
                OutputStream output = connection.getOutputStream();
                output.write(bytes);
                output.flush();
                output.close();
            }
            int status = connection.getResponseCode();
            InputStream stream = status >= 400 ? connection.getErrorStream() : connection.getInputStream();
            String data = readStream(stream);
            Map<String, Object> result = new LinkedHashMap<>();
            result.put("ok", status >= 200 && status < 300);
            result.put("status", status);
            result.put("data", data);
            return result;
        } catch (Exception error) {
            throw new IllegalStateException(error.getMessage(), error);
        } finally {
            if (connection != null) {
                connection.disconnect();
            }
        }
    }

    private static Map<String, Object> payloadMap(Object value) {
        if (value instanceof Map<?, ?>) {
            Map<String, Object> out = new LinkedHashMap<>();
            for (Map.Entry<?, ?> entry : ((Map<?, ?>) value).entrySet()) {
                out.put(String.valueOf(entry.getKey()), entry.getValue());
            }
            return out;
        }
        return new LinkedHashMap<>();
    }

    private static String stringValue(Object value) {
        return value == null ? "" : String.valueOf(value);
    }

    private static String readStream(InputStream stream) throws Exception {
        if (stream == null) {
            return "";
        }
        BufferedReader reader = new BufferedReader(new InputStreamReader(stream, "UTF-8"));
        StringBuilder builder = new StringBuilder();
        String line;
        while ((line = reader.readLine()) != null) {
            builder.append(line);
        }
        reader.close();
        return builder.toString();
    }
}
