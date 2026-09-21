package com.bimcc.headscaleclient;

import android.content.Context;
import android.content.Intent;
import android.net.VpnService;
import android.net.Uri;
import androidx.test.core.app.ActivityScenario;
import androidx.test.core.app.ApplicationProvider;
import androidx.test.ext.junit.runners.AndroidJUnit4;
import androidx.test.platform.app.InstrumentationRegistry;
import android.webkit.WebView;
import android.widget.FrameLayout;
import android.os.Build;
import android.view.View;
import android.view.ViewGroup;
import android.view.inputmethod.InputMethodManager;
import androidx.core.view.ViewCompat;
import androidx.core.view.WindowInsetsCompat;
import androidx.core.graphics.Insets;
import java.util.concurrent.LinkedBlockingQueue;
import java.util.concurrent.TimeUnit;
import engine.Client;
import org.json.JSONObject;
import org.json.JSONArray;
import org.junit.Test;
import org.junit.Before;
import org.junit.runner.RunWith;
import static org.junit.Assert.*;

@RunWith(AndroidJUnit4.class)
public class PreviewSmokeTest {
    @Before public void grantEmulatorNotificationPermission() {
        ClientApplication app = ApplicationProvider.getApplicationContext();
        if (Build.VERSION.SDK_INT >= 33) InstrumentationRegistry.getInstrumentation().getUiAutomation()
            .grantRuntimePermission(app.getPackageName(), android.Manifest.permission.POST_NOTIFICATIONS);
    }

    @Test public void embeddedCoreSurvivesActivityRecreation() throws Exception {
        ClientApplication app = ApplicationProvider.getApplicationContext();
        try (ActivityScenario<MainActivity> activity = ActivityScenario.launch(MainActivity.class)) {
            Client client = app.awaitClient();
            JSONObject response = new JSONObject(client.request("GetSnapshot", "[]"));
            assertFalse(response.toString(), response.has("error"));
            JSONObject snapshot = response.getJSONObject("result");
            assertEquals("native", snapshot.getString("source"));
            assertEquals("android", snapshot.getJSONObject("diagnostics").getString("platform"));
            assertEquals("ready", snapshot.getJSONObject("runtime").getString("daemon"));
            assertEquals("stopped", snapshot.getJSONObject("runtime").getString("connection"));
            assertEquals("1.102.2", snapshot.getJSONObject("diagnostics").getString("daemonVersion"));
            assertFalse(snapshot.getJSONArray("healthNotices").toString().contains("Tailscale is stopped."));
            assertOverviewRendered(activity);
            activity.recreate();
            assertSame(client, app.awaitClient());
            assertFalse(new JSONObject(client.request("GetSnapshot", "[]")).has("error"));
            assertOverviewRendered(activity);
            Thread.sleep(500); // Let the rendered web frame reach SurfaceFlinger.
            captureScreen("headscale-overview.png");
            assertTrue(new JSONObject(client.request("CallLocalAPI", "[\"prefs\"]")).has("error"));
        }
    }

    @Test public void authorizedForegroundServiceSurvivesScreenRecreationAndDisconnects() throws Exception {
        ClientApplication app = ApplicationProvider.getApplicationContext(); app.awaitClient();
        // This grants permission only inside the disposable emulator. Production
        // always uses VpnService.prepare and the Android user-consent dialog.
        try (android.os.ParcelFileDescriptor command = InstrumentationRegistry.getInstrumentation().getUiAutomation()
            .executeShellCommand("appops set " + app.getPackageName() + " ACTIVATE_VPN allow");
             java.io.InputStream output = new android.os.ParcelFileDescriptor.AutoCloseInputStream(command)) {
            byte[] buffer = new byte[1024]; while (output.read(buffer) != -1) { }
        }
        assertNull("Emulator VPN authorization", VpnService.prepare(app));
        try (ActivityScenario<MainActivity> activity = ActivityScenario.launch(MainActivity.class)) {
            app.setDesired(true);
            activity.onActivity(screen -> screen.startForegroundService(new Intent(screen, TunnelService.class)));
            for (int i = 0; i < 50 && app.activeVPN.get() == null; i++) Thread.sleep(200);
            String serviceId = app.activeVPN.get(); assertNotNull("Foreground service did not start", serviceId);
            activity.recreate(); assertEquals(serviceId, app.activeVPN.get());
            activity.onActivity(screen -> screen.startService(new Intent(screen, TunnelService.class).setAction(TunnelService.DISCONNECT)));
            for (int i = 0; i < 50 && app.activeVPN.get() != null; i++) Thread.sleep(200);
            assertNull("Foreground service did not stop", app.activeVPN.get()); assertFalse(app.desired());
        } finally { app.setDesired(false); app.stopService(new Intent(app, TunnelService.class)); }
    }

    private void assertOverviewRendered(ActivityScenario<MainActivity> activity) throws Exception {
        String text = "";
        for (int attempt = 0; attempt < 40; attempt++) {
            LinkedBlockingQueue<String> results = new LinkedBlockingQueue<>();
            activity.onActivity(screen -> {
                FrameLayout root = screen.findViewById(android.R.id.content);
                WebView web = findWebView(root);
                if (web == null) { results.add("WebView unavailable"); return; }
                web.evaluateJavascript("document.body.innerText", results::add);
            });
            text = results.poll(2, TimeUnit.SECONDS);
            if (text != null && text.contains("概览") && text.contains("网络与账号")) return;
            Thread.sleep(500);
        }
        fail("Shared UI did not render native overview: " + text);
    }

    private static WebView findWebView(View view) {
        if (view instanceof WebView) return (WebView)view;
        if (view instanceof ViewGroup) {
            ViewGroup group = (ViewGroup)view;
            for (int i = 0; i < group.getChildCount(); i++) {
                WebView web = findWebView(group.getChildAt(i)); if (web != null) return web;
            }
        }
        return null;
    }

    private String evaluate(ActivityScenario<MainActivity> activity, String script) throws Exception {
        LinkedBlockingQueue<String> results = new LinkedBlockingQueue<>();
        activity.onActivity(screen -> findWebView(screen.findViewById(android.R.id.content)).evaluateJavascript(script, results::add));
        String result = results.poll(3, TimeUnit.SECONDS); assertNotNull("Web evaluation timed out", result); return result;
    }

    private void awaitJS(ActivityScenario<MainActivity> activity, String script) throws Exception {
        for (int i = 0; i < 40; i++) { if ("true".equals(evaluate(activity, script))) return; Thread.sleep(200); }
        fail("UI condition failed: " + script);
    }

    private void captureScreen(String name) throws Exception {
        try (android.os.ParcelFileDescriptor capture = InstrumentationRegistry.getInstrumentation().getUiAutomation()
            .executeShellCommand("screencap -p /sdcard/Download/" + name);
            java.io.InputStream output = new android.os.ParcelFileDescriptor.AutoCloseInputStream(capture)) {
            byte[] buffer = new byte[1024]; while (output.read(buffer) != -1) { }
        }
    }

    @Test public void serverFormFitsRealKeyboardAndSystemBars() throws Exception {
        try (ActivityScenario<MainActivity> activity = ActivityScenario.launch(MainActivity.class)) {
            assertOverviewRendered(activity);
            evaluate(activity, "document.querySelectorAll('.navigation-items button')[2].click()");
            awaitJS(activity, "!!document.querySelector('.endpoint-sidebar-header button')");
            evaluate(activity, "document.querySelector('.endpoint-sidebar-header button').click()");
            awaitJS(activity, "!!document.querySelector('.modal input[type=url]')");
            assertEquals("true", evaluate(activity, "document.activeElement.tagName !== 'INPUT'"));
            int fullHeight = Integer.parseInt(evaluate(activity, "innerHeight"));
            activity.onActivity(screen -> {
                WebView web = findWebView(screen.findViewById(android.R.id.content));
                web.requestFocus();
                web.evaluateJavascript("document.querySelector('.modal input[type=url]').focus()", ignored ->
                    ((InputMethodManager)screen.getSystemService(Context.INPUT_METHOD_SERVICE)).showSoftInput(web, InputMethodManager.SHOW_IMPLICIT));
            });
            boolean visible = false;
            for (int attempt = 0; attempt < 40; attempt++) {
                LinkedBlockingQueue<Boolean> state = new LinkedBlockingQueue<>();
                activity.onActivity(screen -> state.add(ViewCompat.getRootWindowInsets(screen.getWindow().getDecorView()).isVisible(WindowInsetsCompat.Type.ime())));
                if (Boolean.TRUE.equals(state.poll(2, TimeUnit.SECONDS))) { visible = true; break; }
                Thread.sleep(200);
            }
            assertTrue("Real soft keyboard did not open", visible);
            // isVisible(IME) becomes true BEFORE WebView/CSS completes resizing.
            // Assertions against the old innerHeight give a false pass.
            awaitJS(activity, "innerHeight < " + (fullHeight - 100));
            Thread.sleep(600);
            awaitJS(activity, "(() => { const input=document.querySelector('.modal input[type=url]').getBoundingClientRect(); const save=document.querySelector('.modal button[type=submit]').getBoundingClientRect(); return input.top >= 0 && input.bottom <= innerHeight && save.top >= 0 && save.bottom <= innerHeight; })()");
            activity.onActivity(screen -> {
                View decor = screen.getWindow().getDecorView();
                WebView web = findWebView(screen.findViewById(android.R.id.content));
                WindowInsetsCompat insets = ViewCompat.getRootWindowInsets(decor);
                Insets bars = insets.getInsets(WindowInsetsCompat.Type.systemBars());
                Insets ime = insets.getInsets(WindowInsetsCompat.Type.ime());
                int[] location = new int[2]; web.getLocationInWindow(location);
                assertTrue("WebView overlaps status bar", location[1] >= bars.top);
                assertTrue("WebView overlaps keyboard", location[1] + web.getHeight() <= decor.getHeight() - Math.max(bars.bottom, ime.bottom));
            });
            captureScreen("headscale-keyboard.png");
            // Native Back is handled by the sheet once the IME has dismissed.
            evaluate(activity, "window.dispatchEvent(new Event('headscale:back',{cancelable:true}))");
            awaitJS(activity, "!document.querySelector('.modal')");
            awaitJS(activity, "(() => { const header=document.querySelector('.endpoint-accounts .section-header'); const badge=header.querySelector('.status-badge'); return badge.getBoundingClientRect().right <= innerWidth && document.documentElement.scrollWidth <= innerWidth; })()");
        }
    }

    @Test public void officialLoginReturnsBrowserURLFromEmbeddedCore() throws Exception {
        ClientApplication app = ApplicationProvider.getApplicationContext();
        Client client = app.awaitClient();
        JSONObject snapshot = new JSONObject(client.request("GetSnapshot", "[]")).getJSONObject("result");
        JSONArray endpoints = snapshot.getJSONArray("endpoints"); String endpointId = null;
        for (int i = 0; i < endpoints.length(); i++) {
            JSONObject endpoint = endpoints.getJSONObject(i);
            if ("tailscale".equals(endpoint.getString("kind"))) endpointId = endpoint.getString("id");
        }
        assertNotNull("Built-in official endpoint", endpointId);
        try {
            JSONObject response = new JSONObject(client.request("BeginLogin", new JSONArray().put(endpointId).toString()));
            // Never log the token-bearing authentication URL.
            assertFalse("Official login failed: " + response.optString("error"), response.has("error"));
            Uri url = Uri.parse(response.getJSONObject("result").getString("authUrl"));
            assertEquals("https", url.getScheme()); assertEquals("login.tailscale.com", url.getHost());
            assertTrue(url.getPath().length() > 1);
        } finally { client.request("SetConnection", "[false]"); }
    }

    @Test public void identitiesAreEncryptedAndBoundToTheirKey() throws Exception {
        ClientApplication app = ApplicationProvider.getApplicationContext(); app.awaitClient();
        String key = "preview-storage-test";
        try {
            app.encryptToPref(key, "test-secret");
            assertEquals("test-secret", app.decryptFromPref(key));
            String stored = app.getSharedPreferences("encrypted-identities", Context.MODE_PRIVATE).getString(key, "");
            assertFalse(stored.contains("test-secret"));
            app.getSharedPreferences("encrypted-identities", Context.MODE_PRIVATE).edit().putString(key + "-other", stored).commit();
            try { app.decryptFromPref(key + "-other"); fail("ciphertext must be authenticated for its preference key"); }
            catch (javax.crypto.AEADBadTagException expected) { }
        } finally { app.encryptToPref(key, ""); app.encryptToPref(key + "-other", ""); }
    }

    @Test public void privilegedOriginIsExact() {
        assertTrue(MainActivity.trusted(Uri.parse(MainActivity.ORIGIN + "/index.html")));
        for (String url : new String[]{"http://appassets.androidplatform.net/index.html", "https://example.com/",
            "https://user@appassets.androidplatform.net/", "https://appassets.androidplatform.net:444/", "file:///sdcard/index.html"})
            assertFalse(url, MainActivity.trusted(Uri.parse(url)));
    }
}
