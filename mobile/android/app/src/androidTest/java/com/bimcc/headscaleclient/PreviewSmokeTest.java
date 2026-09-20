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
import java.util.concurrent.LinkedBlockingQueue;
import java.util.concurrent.TimeUnit;
import engine.Client;
import org.json.JSONObject;
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
                if (!(root.getChildAt(0) instanceof WebView)) { results.add("WebView unavailable"); return; }
                ((WebView)root.getChildAt(0)).evaluateJavascript("document.body.innerText", results::add);
            });
            text = results.poll(2, TimeUnit.SECONDS);
            if (text != null && text.contains("概览") && text.contains("网络与账号")) return;
            Thread.sleep(500);
        }
        fail("Shared UI did not render native overview: " + text);
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
