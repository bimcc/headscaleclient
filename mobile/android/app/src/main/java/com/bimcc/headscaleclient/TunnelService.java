package com.bimcc.headscaleclient;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.Intent;
import android.net.IpPrefix;
import android.net.VpnService;
import android.os.Build;
import android.system.OsConstants;
import java.net.InetAddress;
import java.util.UUID;
import java.util.concurrent.atomic.AtomicBoolean;
import libtailscale.IPNService;
import libtailscale.Libtailscale;
import libtailscale.VPNServiceBuilder;

public final class TunnelService extends VpnService implements IPNService {
    static final String DISCONNECT = "com.bimcc.headscaleclient.DISCONNECT";
    private final String instanceId = UUID.randomUUID().toString();
    private final AtomicBoolean attached = new AtomicBoolean();
    private ClientApplication app;
    private final android.os.Handler main = new android.os.Handler(android.os.Looper.getMainLooper());
    @Override public void onCreate() {
        super.onCreate(); app = (ClientApplication) getApplication();
        getSystemService(NotificationManager.class).createNotificationChannel(new NotificationChannel("vpn", getString(R.string.vpn_channel), NotificationManager.IMPORTANCE_LOW));
    }
    @Override public int onStartCommand(Intent intent, int flags, int startId) {
        if ((intent != null && DISCONNECT.equals(intent.getAction())) || !app.desired() || prepare(this) != null) {
            disconnectVPN(); return START_NOT_STICKY;
        }
        startForeground(1, notification(false));
        app.activeVPN.set(instanceId);
        app.vpnWorker.execute(() -> {
            try {
                if (!instanceId.equals(app.activeVPN.get()) || !app.desired()) return;
                if (attached.compareAndSet(false, true)) {
                    app.awaitClient().setVPNState(true, false);
                    Libtailscale.requestVPN(this);
                }
                // A sticky restart restores only previously authorized intent.
                if (intent == null) app.awaitClient().request("SetConnection", "[true]");
            } catch (Exception e) { disconnectVPN(); }
        });
        return START_STICKY;
    }
    private Notification notification(boolean connected) {
        PendingIntent open = PendingIntent.getActivity(this, 0, new Intent(this, MainActivity.class), PendingIntent.FLAG_IMMUTABLE | PendingIntent.FLAG_UPDATE_CURRENT);
        PendingIntent stop = PendingIntent.getService(this, 1, new Intent(this, TunnelService.class).setAction(DISCONNECT), PendingIntent.FLAG_IMMUTABLE | PendingIntent.FLAG_UPDATE_CURRENT);
        return new Notification.Builder(this, "vpn").setSmallIcon(android.R.drawable.ic_lock_lock)
            .setContentTitle("HeadscaleClient").setContentText(getString(connected ? R.string.vpn_connected : R.string.vpn_starting))
            .setContentIntent(open).setOngoing(true).addAction(new Notification.Action.Builder(null, getString(R.string.disconnect), stop).build()).build();
    }
    @Override public String id() { return instanceId; }
    @Override public VPNServiceBuilder newBuilder() {
        Builder builder = new Builder().setSession("HeadscaleClient").setConfigureIntent(PendingIntent.getActivity(this, 0,
            new Intent(this, MainActivity.class), PendingIntent.FLAG_IMMUTABLE | PendingIntent.FLAG_UPDATE_CURRENT));
        builder.allowFamily(OsConstants.AF_INET); builder.allowFamily(OsConstants.AF_INET6);
        if (Build.VERSION.SDK_INT >= 29) builder.setMetered(false);
        builder.setUnderlyingNetworks(null);
        return new VPNServiceBuilder() {
            public void setMTU(int mtu) { builder.setMtu(mtu); }
            public void addDNSServer(String address) { builder.addDnsServer(address); }
            public void addSearchDomain(String domain) { builder.addSearchDomain(domain); }
            public void addRoute(String address, int prefix) { builder.addRoute(address, prefix); }
            public void excludeRoute(String address, int prefix) throws Exception {
                if (Build.VERSION.SDK_INT >= 33) builder.excludeRoute(new IpPrefix(InetAddress.getByName(address), prefix));
                else throw new Exception("route exclusion requires upstream prefix subtraction on this Android version");
            }
            public void addAddress(String address, int prefix) { builder.addAddress(address, prefix); }
            public libtailscale.ParcelFileDescriptor establish() throws Exception {
                android.os.ParcelFileDescriptor tunnel = builder.establish();
                if (tunnel == null) throw new Exception("VPN permission revoked");
                return () -> tunnel.detachFd();
            }
        };
    }
    @Override public void updateVpnStatus(boolean connected) {
        app.vpnWorker.execute(() -> {
            if (!instanceId.equals(app.activeVPN.get())) return;
            try { app.awaitClient().setVPNState(app.desired(), connected); } catch (Exception ignored) {}
        });
        main.post(() -> { if (app.desired() && instanceId.equals(app.activeVPN.get())) getSystemService(NotificationManager.class).notify(1, notification(connected)); });
    }
    @Override public void close() { main.post(this::stopSelf); }
    @Override public void disconnectVPN() {
        String owner = app.activeVPN.get();
        if (owner != null && !instanceId.equals(owner)) { close(); return; }
        app.setDesired(false);
        app.vpnWorker.execute(() -> {
            if (app.desired()) return;
            String current = app.activeVPN.get();
            if (current != null && !instanceId.equals(current)) return;
            try { app.awaitClient().setVPNState(false, false); app.awaitClient().request("SetConnection", "[false]"); }
            catch (Exception ignored) {}
        });
        close();
    }
    @Override public void onRevoke() { disconnectVPN(); super.onRevoke(); }
    @Override public void onDestroy() {
        app.activeVPN.compareAndSet(instanceId, null);
        app.vpnWorker.execute(() -> {
            if (app.activeVPN.get() == null) {
                try { app.awaitClient().setVPNState(false, false); } catch (Exception ignored) {}
            }
            if (attached.compareAndSet(true, false)) Libtailscale.serviceDisconnect(this);
        });
        stopForeground(STOP_FOREGROUND_REMOVE); super.onDestroy();
    }
}
