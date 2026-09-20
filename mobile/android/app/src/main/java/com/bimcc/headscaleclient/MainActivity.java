package com.bimcc.headscaleclient;

import android.app.Activity;
import android.content.Intent;
import android.net.Uri;
import android.net.VpnService;
import android.os.*;
import android.view.View;
import android.webkit.*;
import android.widget.TextView;
import androidx.webkit.*;
import org.json.*;
import java.io.ByteArrayInputStream;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.atomic.AtomicInteger;

public final class MainActivity extends Activity implements ClientApplication.EventListener {
    static final String ORIGIN = "https://appassets.androidplatform.net";
    private WebView web;
    private ClientApplication app;
    private Runnable afterPermission;
    private Runnable deniedPermission;
    private boolean destroyed;
    private final Handler main = new Handler(Looper.getMainLooper());
    private final AtomicInteger pending = new AtomicInteger();

    @Override public void onCreate(Bundle state) {
        super.onCreate(state); app = (ClientApplication)getApplication();
        if (!WebViewFeature.isFeatureSupported(WebViewFeature.WEB_MESSAGE_LISTENER)) {
            TextView text = new TextView(this); text.setText("请更新 Android System WebView 后重试。 / Please update Android System WebView.");
            setContentView(text); return;
        }
        web = new WebView(this);
        web.getSettings().setJavaScriptEnabled(true); web.getSettings().setDomStorageEnabled(true);
        web.getSettings().setAllowFileAccess(false); web.getSettings().setAllowContentAccess(false);
        web.getSettings().setMixedContentMode(WebSettings.MIXED_CONTENT_NEVER_ALLOW);
        web.setImportantForAutofill(View.IMPORTANT_FOR_AUTOFILL_NO_EXCLUDE_DESCENDANTS);
        WebViewAssetLoader assets = new WebViewAssetLoader.Builder().addPathHandler("/", new WebViewAssetLoader.AssetsPathHandler(this)).build();
        web.setWebViewClient(new WebViewClient() {
            @Override public WebResourceResponse shouldInterceptRequest(WebView view, WebResourceRequest request) {
                WebResourceResponse response = assets.shouldInterceptRequest(request.getUrl());
                if (response == null) return new WebResourceResponse("text/plain", "UTF-8", 403, "Forbidden", Map.of(), new ByteArrayInputStream(new byte[0]));
                response.setResponseHeaders(Map.of("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'none'; frame-src 'none'; object-src 'none'; base-uri 'none'"));
                return response;
            }
            @Override public boolean shouldOverrideUrlLoading(WebView view, WebResourceRequest request) {
                if (trusted(request.getUrl())) return false;
                if (request.isForMainFrame() && request.hasGesture()) openBrowser(request.getUrl().toString());
                return true;
            }
        });
        WebViewCompat.addWebMessageListener(web, "HeadscaleAndroid", Set.of(ORIGIN), (view, message, origin, mainFrame, reply) -> {
            if (!mainFrame || !ORIGIN.equals(origin.toString()) || pending.get() >= 8) return;
            try {
                String raw = message.getData(); if (raw == null || raw.length() > 65536) return;
                JSONObject request = new JSONObject(raw); String id = request.getString("id");
                String method = request.getString("method"); JSONArray args = request.getJSONArray("args");
                pending.incrementAndGet();
                if ("OpenURL".equals(method)) {
                    boolean opened = args.length() == 1 && openBrowser(args.getString(0));
                    respond(reply, id, opened ? "{\"result\":null}" : error(getString(R.string.browser_failed))); return;
                }
                boolean wasDesired = app.desired();
                java.util.concurrent.ExecutorService executor = Set.of("BeginLogin", "SwitchProfile", "SetConnection", "Logout").contains(method) ? app.vpnWorker : app.worker;
                Runnable execute = () -> executor.execute(() -> {
                    try {
                        String response = app.awaitClient().request(method, args.toString());
                        boolean ok = !new JSONObject(response).has("error");
                        if (!ok && !wasDesired && ("BeginLogin".equals(method) || "SwitchProfile".equals(method) || "SetConnection".equals(method))) {
                            app.setDesired(false); stopService(new Intent(this, TunnelService.class));
                        }
                        if (ok && ("Logout".equals(method) || ("SetConnection".equals(method) && !args.getBoolean(0)))) {
                            app.setDesired(false); stopService(new Intent(this, TunnelService.class));
                        }
                        respond(reply, id, response);
                    } catch (Exception e) { respond(reply, id, error(getString(R.string.startup_failed))); }
                });
                boolean needsVpn = "BeginLogin".equals(method) || "SwitchProfile".equals(method)
                    || ("SetConnection".equals(method) && args.length() == 1 && args.getBoolean(0));
                if (needsVpn) {
                    if (afterPermission != null) { respond(reply,id,error(getString(R.string.vpn_permission_denied))); return; }
                    Runnable start = () -> {
                        try {
                            app.setDesired(true); startForegroundService(new Intent(this, TunnelService.class)); execute.run();
                        } catch (Exception e) { app.setDesired(false); respond(reply,id,error(getString(R.string.vpn_start_failed))); }
                    };
                    Intent permission = VpnService.prepare(this);
                    if (permission == null) start.run();
                    else {
                        afterPermission = start; deniedPermission = () -> respond(reply,id,error(getString(R.string.vpn_permission_denied)));
                        main.postDelayed(() -> {
                            if (afterPermission != start) return;
                            Runnable deny = deniedPermission; afterPermission = null; deniedPermission = null;
                            if (deny != null) deny.run();
                        }, 30000);
                        startActivityForResult(permission, 100);
                    }
                } else execute.run();
            } catch (Exception ignored) { /* Invalid bridge envelopes have no authority. */ }
        });
        setContentView(web);
        // API 35 edge-to-edge: keep web controls outside status/navigation bars.
        web.setOnApplyWindowInsetsListener((view, insets) -> {
            view.setPadding(insets.getSystemWindowInsetLeft(), insets.getSystemWindowInsetTop(), insets.getSystemWindowInsetRight(), insets.getSystemWindowInsetBottom());
            return insets;
        });
        web.loadUrl(ORIGIN + "/index.html");
        if (Build.VERSION.SDK_INT >= 33 && checkSelfPermission(android.Manifest.permission.POST_NOTIFICATIONS) != android.content.pm.PackageManager.PERMISSION_GRANTED)
            requestPermissions(new String[]{android.Manifest.permission.POST_NOTIFICATIONS}, 101);
    }

    static boolean trusted(Uri uri) { return "https".equals(uri.getScheme()) && "appassets.androidplatform.net".equals(uri.getHost()) && uri.getPort() == -1 && uri.getUserInfo() == null; }
    private boolean openBrowser(String url) {
        Uri uri = Uri.parse(url);
        if (!"https".equals(uri.getScheme()) || uri.getHost() == null || uri.getUserInfo() != null) return false;
        try { startActivity(new Intent(Intent.ACTION_VIEW, uri).addCategory(Intent.CATEGORY_BROWSABLE)); return true; }
        catch (Exception e) { return false; }
    }
    private static String error(String message) { return "{\"error\":" + JSONObject.quote(message) + "}"; }
    private void respond(JavaScriptReplyProxy reply, String id, String response) {
        pending.decrementAndGet();
        runOnUiThread(() -> { if (!destroyed) reply.postMessage("{\"id\":" + JSONObject.quote(id) + ",\"response\":" + response + "}"); });
    }
    @Override protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        super.onActivityResult(requestCode,resultCode,data);
        if (requestCode != 100) return;
        Runnable accepted = afterPermission, denied = deniedPermission; afterPermission = null; deniedPermission = null;
        if (resultCode == RESULT_OK && accepted != null) accepted.run(); else if (denied != null) denied.run();
    }
    @Override protected void onStart() { super.onStart(); app.listeners.add(this); }
    @Override protected void onStop() { app.listeners.remove(this); super.onStop(); }
    @Override protected void onResume() {
        super.onResume();
        if (web != null) web.evaluateJavascript("window.dispatchEvent(new Event('headscale:resume'))", null);
    }
    @Override public void onEvent(String name, String payload) {
        runOnUiThread(() -> { if (!destroyed && web != null) web.evaluateJavascript(
            "window.dispatchEvent(new CustomEvent('headscale:native',{detail:{name:" + JSONObject.quote(name) + ",payload:" + payload + "}}))", null); });
    }
    @Override protected void onDestroy() {
        destroyed = true; app.listeners.remove(this); afterPermission = null; deniedPermission = null;
        main.removeCallbacksAndMessages(null);
        if (web != null) { WebViewCompat.removeWebMessageListener(web,"HeadscaleAndroid"); web.destroy(); }
        // Deliberately leave the Application's Go runtime and VPN service alive.
        super.onDestroy();
    }
}
