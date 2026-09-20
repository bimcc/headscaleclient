package com.bimcc.headscaleclient;

import android.app.Application;
import android.content.SharedPreferences;
import android.net.*;
import android.os.*;
import android.provider.Settings;
import android.security.keystore.KeyGenParameterSpec;
import android.security.keystore.KeyProperties;
import android.util.Base64;
import org.json.*;
import java.net.NetworkInterface;
import java.nio.charset.StandardCharsets;
import java.security.KeyStore;
import java.util.*;
import java.util.concurrent.*;
import java.util.concurrent.atomic.AtomicReference;
import javax.crypto.*;
import javax.crypto.spec.GCMParameterSpec;
import engine.Client;
import engine.Engine;
import engine.Events;
import libtailscale.AppContext;
import libtailscale.Libtailscale;

public final class ClientApplication extends Application implements AppContext, Events {
    final ExecutorService worker = Executors.newFixedThreadPool(3);
    final ExecutorService vpnWorker = Executors.newSingleThreadExecutor();
    final CompletableFuture<Client> client = new CompletableFuture<>();
    final AtomicReference<String> activeVPN = new AtomicReference<>();
    final CopyOnWriteArrayList<EventListener> listeners = new CopyOnWriteArrayList<>();
    private ConnectivityManager connectivity;
    private volatile Network underlying;
    private volatile String dns = "";
    private SharedPreferences identities;
    private SecretKey stateKey;
    interface EventListener { void onEvent(String name, String payload); }

    @Override public void onCreate() {
        super.onCreate();
        identities = getSharedPreferences("encrypted-identities", MODE_PRIVATE);
        connectivity = getSystemService(ConnectivityManager.class);
        worker.execute(() -> {
            try {
                KeyStore store = KeyStore.getInstance("AndroidKeyStore"); store.load(null);
                if (!store.containsAlias("headscaleclient-state-v1")) {
                    KeyGenerator generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, "AndroidKeyStore");
                    generator.init(new KeyGenParameterSpec.Builder("headscaleclient-state-v1", KeyProperties.PURPOSE_ENCRYPT | KeyProperties.PURPOSE_DECRYPT)
                        .setBlockModes(KeyProperties.BLOCK_MODE_GCM).setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE).build());
                    generator.generateKey();
                }
                stateKey = (SecretKey) store.getKey("headscaleclient-state-v1", null);
                updateNetwork();
                connectivity.registerNetworkCallback(new NetworkRequest.Builder()
                    .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
                    .addCapability(NetworkCapabilities.NET_CAPABILITY_NOT_VPN).build(), new ConnectivityManager.NetworkCallback() {
                        @Override public void onAvailable(Network n) { updateNetwork(); }
                        @Override public void onLost(Network n) { updateNetwork(); }
                        @Override public void onCapabilitiesChanged(Network n, NetworkCapabilities c) { updateNetwork(); }
                        @Override public void onLinkPropertiesChanged(Network n, LinkProperties l) { updateNetwork(); }
                    });
                Client runtime = Engine.start(getFilesDir().getAbsolutePath(), this, this);
                runtime.setVPNState(false, false);
                // The upstream constructor returns before its backend is ready.
                // Do not send an unbuffered RequestVPN to a failed initializer.
                long deadline = SystemClock.elapsedRealtime() + 25000;
                boolean ready = false;
                while (SystemClock.elapsedRealtime() < deadline) {
                    JSONObject result = new JSONObject(runtime.request("GetSnapshot", "[]")).optJSONObject("result");
                    if (result != null && "ready".equals(result.getJSONObject("runtime").optString("daemon"))) { ready = true; break; }
                    Thread.sleep(250);
                }
                if (!ready) throw new Exception("embedded backend readiness timeout");
                client.complete(runtime);
            } catch (Exception e) { client.completeExceptionally(e); }
        });
    }

    Client awaitClient() throws Exception { return client.get(35, TimeUnit.SECONDS); }
    boolean desired() { return getSharedPreferences("lifecycle", MODE_PRIVATE).getBoolean("desired", false); }
    void setDesired(boolean value) { getSharedPreferences("lifecycle", MODE_PRIVATE).edit().putBoolean("desired", value).commit(); }
    @Override public void onEvent(String name, String payload) {
        for (EventListener listener : listeners) listener.onEvent(name, payload);
    }

    private synchronized void updateNetwork() {
        Network chosen = null;
        int best = -1;
        for (Network network : connectivity.getAllNetworks()) {
            NetworkCapabilities c = connectivity.getNetworkCapabilities(network);
            if (c == null || !c.hasCapability(NetworkCapabilities.NET_CAPABILITY_NOT_VPN)
                || !c.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)) continue;
            int score = (c.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED) ? 4 : 0)
                + (c.hasCapability(NetworkCapabilities.NET_CAPABILITY_NOT_METERED) ? 2 : 0);
            if (score > best) { chosen = network; best = score; }
        }
        underlying = chosen;
        LinkProperties links = chosen == null ? null : connectivity.getLinkProperties(chosen);
        StringJoiner servers = new StringJoiner(" ");
        String gateway = "";
        if (links != null) {
            for (java.net.InetAddress address : links.getDnsServers()) servers.add(address.getHostAddress());
            for (RouteInfo route : links.getRoutes()) if (route.isDefaultRoute() && route.getGateway() != null && !route.getGateway().isAnyLocalAddress()) { gateway = route.getGateway().getHostAddress(); break; }
        }
        dns = servers + "\n" + (links != null && links.getDomains() != null ? links.getDomains() : "");
        Libtailscale.onGatewayChanged(gateway);
        Libtailscale.onDNSConfigChanged(links == null || links.getInterfaceName() == null ? "" : links.getInterfaceName());
    }

    @Override public synchronized void encryptToPref(String key, String value) throws Exception {
        if (value.isEmpty()) { if (!identities.edit().remove(key).commit()) throw new Exception("state write failed"); return; }
        Cipher cipher = Cipher.getInstance("AES/GCM/NoPadding"); cipher.init(Cipher.ENCRYPT_MODE, stateKey);
        cipher.updateAAD(key.getBytes(StandardCharsets.UTF_8));
        String encoded = Base64.encodeToString(cipher.getIV(), Base64.NO_WRAP) + ":" +
            Base64.encodeToString(cipher.doFinal(value.getBytes(StandardCharsets.UTF_8)), Base64.NO_WRAP);
        if (!identities.edit().putString(key, encoded).commit()) throw new Exception("state write failed");
    }
    @Override public synchronized String decryptFromPref(String key) throws Exception {
        String encoded = identities.getString(key, ""); if (encoded.isEmpty()) return "";
        String[] parts = encoded.split(":", 2);
        Cipher cipher = Cipher.getInstance("AES/GCM/NoPadding");
        cipher.init(Cipher.DECRYPT_MODE, stateKey, new GCMParameterSpec(128, Base64.decode(parts[0], Base64.NO_WRAP)));
        cipher.updateAAD(key.getBytes(StandardCharsets.UTF_8));
        return new String(cipher.doFinal(Base64.decode(parts[1], Base64.NO_WRAP)), StandardCharsets.UTF_8);
    }
    @Override public String getStateStoreKeysJSON() {
        JSONArray keys = new JSONArray();
        for (String key : identities.getAll().keySet()) if (key.startsWith("statestore-")) keys.put(key.substring(10));
        return keys.toString();
    }
    @Override public void log(String tag, String line) { /* Do not persist login URLs, keys or network metadata in logcat. */ }
    @Override public String getOSVersion() { return Build.VERSION.RELEASE; }
    @Override public long getSDKInt() { return Build.VERSION.SDK_INT; }
    @Override public String getDeviceName() {
        String name = Settings.Global.getString(getContentResolver(), "device_name");
        return name == null || name.trim().isEmpty() ? Build.MANUFACTURER + " " + Build.MODEL : name;
    }
    @Override public String getInstallSource() { return "headscaleclient-android-preview"; }
    @Override public boolean shouldUseGoogleDNSFallback() { return false; }
    @Override public boolean isChromeOS() { return false; }
    @Override public boolean isClientLoggingEnabled() { return false; }
    @Override public String getPlatformDNSConfig() { return dns; }
    @Override public String getInterfacesAsJson() throws Exception {
        JSONArray result = new JSONArray(); Set<String> seen = new HashSet<>();
        for (Network network : connectivity.getAllNetworks()) {
            LinkProperties links = connectivity.getLinkProperties(network);
            if (links == null || links.getInterfaceName() == null || !seen.add(links.getInterfaceName())) continue;
            String name = links.getInterfaceName(); NetworkInterface iface = NetworkInterface.getByName(name);
            if (iface == null) continue;
            int mtu = Build.VERSION.SDK_INT >= 29 ? links.getMtu() : iface.getMTU();
            JSONArray addresses = new JSONArray();
            for (LinkAddress address : links.getLinkAddresses()) addresses.put(new JSONObject()
                .put("ip", address.getAddress().getHostAddress()).put("prefixLen", address.getPrefixLength()));
            result.put(new JSONObject().put("name", name).put("index", iface.getIndex())
                .put("mtu", mtu > 0 ? mtu : 1500).put("up", true)
                .put("loopback", iface.isLoopback()).put("pointToPoint", iface.isPointToPoint())
                .put("multicast", iface.supportsMulticast()).put("broadcast", false).put("addrs", addresses));
        }
        return result.toString();
    }
    @Override public boolean bindSocketToNetwork(int fd) {
        Network network = underlying; if (network == null) return false;
        try (ParcelFileDescriptor duplicate = ParcelFileDescriptor.fromFd(fd)) {
            network.bindSocket(duplicate.getFileDescriptor()); return true;
        } catch (Exception e) { return false; }
    }
    @Override public byte[] getUserCACertsPEM() { return new byte[0]; }
    @Override public String getSyspolicyStringValue(String key) throws Exception { throw new Exception("no such key"); }
    @Override public boolean getSyspolicyBooleanValue(String key) throws Exception { throw new Exception("no such key"); }
    @Override public String getSyspolicyStringArrayJSONValue(String key) throws Exception { throw new Exception("no such key"); }
    @Override public boolean hardwareAttestationKeySupported() { return false; }
    @Override public String hardwareAttestationKeyCreate() throws Exception { throw new Exception("unsupported"); }
    @Override public void hardwareAttestationKeyRelease(String id) throws Exception { throw new Exception("unsupported"); }
    @Override public byte[] hardwareAttestationKeyPublic(String id) throws Exception { throw new Exception("unsupported"); }
    @Override public byte[] hardwareAttestationKeySign(String id, byte[] data) throws Exception { throw new Exception("unsupported"); }
    @Override public void hardwareAttestationKeyLoad(String id) throws Exception { throw new Exception("unsupported"); }
}
