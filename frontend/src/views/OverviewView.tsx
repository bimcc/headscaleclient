import {
  AlertTriangle,
  ArrowRight,
  CheckCircle2,
  Copy,
  Laptop,
  Radio,
  ServerOff,
} from "lucide-react";
import type {
  AppSnapshot,
  PeerDevice,
  PreferenceKey,
  RuntimeState,
} from "../lib/contracts";
import { IconButton, SettingRow, StatusBadge, Toggle, type Tone } from "../components/ui";
import { useI18n, type Translate } from "../lib/i18n";

function connectionSummary(runtime: RuntimeState, t: Translate): { label: string; detail: string; tone: Tone } {
  if (["missing", "stopped", "incompatible"].includes(runtime.daemon)) {
    return { label: t("connection.serviceUnavailable"), detail: t("connection.tailscaledStopped"), tone: "danger" };
  }
  if (runtime.session === "login-required" || runtime.session === "none") {
    return { label: t("connection.loginRequired"), detail: t("connection.selectNetworkAccount"), tone: "warning" };
  }
  if (runtime.session === "approval-required") {
    return { label: t("connection.awaitingApproval"), detail: t("connection.deviceNotApproved"), tone: "warning" };
  }
  if (runtime.connection === "starting" || runtime.connection === "stopping") {
    return { label: t("connection.connecting"), detail: t("connection.applying"), tone: "neutral" };
  }
  if (runtime.connection === "degraded") {
    return runtime.control === "unreachable"
      ? { label: t("connection.tunnelConnected"), detail: t("connection.controlLimited"), tone: "warning" }
      : { label: t("connection.tunnelConnected"), detail: t("connection.localWarning"), tone: "warning" };
  }
  if (runtime.connection === "running") {
    if (runtime.control === "unreachable") {
      return { label: t("connection.tunnelConnected"), detail: t("connection.controlLimited"), tone: "warning" };
    }
    return { label: t("connection.connected"), detail: t("connection.tunnelHealthy"), tone: "success" };
  }
  return { label: t("connection.notConnected"), detail: t("connection.tunnelStopped"), tone: "neutral" };
}

function DeviceRow({ device, onCopy }: { device: PeerDevice; onCopy: (value: string) => void }) {
  const { t } = useI18n();
  return (
    <div className="compact-device-row">
      <span className={`presence-dot ${device.online ? "online" : "offline"}`} aria-hidden="true" />
      <div className="device-primary truncate" title={device.name}>
        <strong>{device.name}</strong>
        <span>{device.os}</span>
      </div>
      <span className="mono device-address">{device.addresses[0]}</span>
      <span className="device-path">
        {device.connectionType === "direct"
          ? t("device.direct")
          : device.connectionType === "relay"
            ? `${t("device.relay")} · ${device.relayRegion}`
            : device.lastSeen}
      </span>
      <IconButton label={t("device.copyAddress", { name: device.name })} icon={Copy} onClick={() => onCopy(device.addresses[0])} />
    </div>
  );
}

export function OverviewView({
  snapshot,
  busy,
  onConnectionChange,
  onEnsureDaemon,
  onPreferenceChange,
  onExitNodeChange,
  onCopy,
  onShowDevices,
}: {
  snapshot: AppSnapshot;
  busy: string | null;
  onConnectionChange: (enabled: boolean) => void;
  onEnsureDaemon: () => void;
  onPreferenceChange: (key: PreferenceKey, value: boolean) => void;
  onExitNodeChange: (deviceId: string | null) => void;
  onCopy: (value: string, label?: string) => void;
  onShowDevices: () => void;
}) {
  const { t } = useI18n();
  const status = connectionSummary(snapshot.runtime, t);
  const endpoint = snapshot.endpoints.find((item) => item.id === snapshot.activeEndpointId);
  const profile = snapshot.profiles.find((item) => item.id === snapshot.activeProfileId);
  const recentDevices = snapshot.devices.filter((item) => item.online).slice(0, 5);
  const isTunnelRunning = ["running", "degraded"].includes(snapshot.runtime.connection);
  const canEnsureDaemon = snapshot.engine.canInstall || snapshot.engine.canStart;
  const daemonActionLabel = snapshot.engine.canInstall ? t("daemon.install") : t("daemon.start");

  return (
    <div className="view-stack">
      <section className="connection-strip" aria-labelledby="connection-title">
        <div className={`connection-icon tone-${status.tone}`}>
          {status.tone === "danger" ? (
            <ServerOff aria-hidden="true" size={21} />
          ) : isTunnelRunning ? (
            <CheckCircle2 aria-hidden="true" size={21} />
          ) : (
            <Radio aria-hidden="true" size={21} />
          )}
        </div>
        <div className="connection-copy">
          <div className="connection-heading">
            <h2 id="connection-title">{status.label}</h2>
            <StatusBadge tone={status.tone}>{status.detail}</StatusBadge>
          </div>
          <p className="truncate" title={`${endpoint?.name ?? t("network.noneSelected")} · ${profile?.account ?? t("account.noneSelected")}`}>
            {endpoint?.name ?? t("network.noneSelected")} · {profile?.account ?? t("account.noneSelected")}
          </p>
        </div>
        <Toggle
          label={isTunnelRunning ? t("connection.disconnect") : t("connection.connect")}
          checked={isTunnelRunning}
          disabled={busy === "connection"}
          onChange={onConnectionChange}
        />
      </section>

      {snapshot.fallbackReason && (
        <div className="persistent-alert" role="status">
          <AlertTriangle aria-hidden="true" size={19} />
          <div>
            <strong>{t("daemon.unavailable")}</strong>
            <span>{snapshot.fallbackReason}</span>
          </div>
          {canEnsureDaemon && (
            <button className="button secondary daemon-action" type="button" disabled={busy === "daemon"} onClick={onEnsureDaemon}>
              {daemonActionLabel}
            </button>
          )}
        </div>
      )}

      <section className="section-block" aria-labelledby="local-device-title">
        <header className="section-header">
          <div>
            <h2 id="local-device-title">{t("overview.localDevice")}</h2>
            <p>{snapshot.localDevice.os} · {t("overview.clientVersion", { version: snapshot.localDevice.clientVersion })}</p>
          </div>
          <Laptop aria-hidden="true" size={20} />
        </header>
        <dl className="facts-grid">
          <div>
            <dt>{t("overview.deviceName")}</dt>
            <dd>
              <span className="truncate" title={snapshot.localDevice.name}>{snapshot.localDevice.name}</span>
              <IconButton label={t("overview.copyDeviceName")} icon={Copy} onClick={() => onCopy(snapshot.localDevice.name)} />
            </dd>
          </div>
          <div>
            <dt>MagicDNS</dt>
            <dd>
              <span className="truncate" title={snapshot.localDevice.dnsName}>{snapshot.localDevice.dnsName}</span>
              <IconButton label={t("overview.copyMagicDNS")} icon={Copy} onClick={() => onCopy(snapshot.localDevice.dnsName)} />
            </dd>
          </div>
          {snapshot.localDevice.addresses.map((address, index) => (
            <div key={address}>
              <dt>{index === 0 ? "IPv4" : "IPv6"}</dt>
              <dd>
                <span className="mono truncate" title={address}>{address}</span>
                <IconButton label={t("overview.copyAddress", { type: index === 0 ? "IPv4" : "IPv6" })} icon={Copy} onClick={() => onCopy(address)} />
              </dd>
            </div>
          ))}
        </dl>
      </section>

      <section className="section-block" aria-labelledby="preferences-title">
        <header className="section-header">
          <div>
            <h2 id="preferences-title">{t("overview.preferences")}</h2>
            <p>{t("overview.currentAccountScope")}</p>
          </div>
        </header>
        <div className="setting-list">
          <SettingRow
            title={t("overview.exitNode")}
            description={t("overview.exitNodeDescription")}
            control={
              <select
                aria-label={t("overview.exitNode")}
                value={snapshot.preferences.exitNodeId ?? ""}
                disabled={busy === "exitNode"}
                onChange={(event) => onExitNodeChange(event.target.value || null)}
              >
                <option value="">{t("overview.doNotUse")}</option>
                {snapshot.devices.filter((device) => device.online && device.exitNodeOption).map((device) => (
                  <option key={device.id} value={device.id}>{device.name}</option>
                ))}
              </select>
            }
          />
          <SettingRow
            title={t("overview.allowLAN")}
            description={t("overview.allowLANDescription")}
            control={<Toggle label={t("overview.allowLAN")} checked={snapshot.preferences.allowLanAccess} disabled={busy === "allowLanAccess"} onChange={(value) => onPreferenceChange("allowLanAccess", value)} />}
          />
          <SettingRow
            title={t("overview.magicDNS")}
            description={t("overview.magicDNSDescription")}
            control={<Toggle label={t("overview.magicDNS")} checked={snapshot.preferences.acceptDns} disabled={busy === "acceptDns"} onChange={(value) => onPreferenceChange("acceptDns", value)} />}
          />
          <SettingRow
            title={t("overview.acceptRoutes")}
            description={t("overview.acceptRoutesDescription")}
            control={<Toggle label={t("overview.acceptRoutes")} checked={snapshot.preferences.acceptRoutes} disabled={busy === "acceptRoutes"} onChange={(value) => onPreferenceChange("acceptRoutes", value)} />}
          />
          <SettingRow
            title={t("overview.shieldsUp")}
            description={t("overview.shieldsUpDescription")}
            control={<Toggle label={t("overview.shieldsUp")} checked={snapshot.preferences.shieldsUp} disabled={busy === "shieldsUp"} onChange={(value) => onPreferenceChange("shieldsUp", value)} />}
          />
        </div>
      </section>

      <section className="section-block" aria-labelledby="recent-title">
        <header className="section-header">
          <div>
            <h2 id="recent-title">{t("overview.onlineDevices")}</h2>
            <p>{t("overview.reachableCount", { count: recentDevices.length })}</p>
          </div>
          <button className="button quiet with-icon" type="button" onClick={onShowDevices}>
            {t("overview.viewAll")} <ArrowRight aria-hidden="true" size={17} />
          </button>
        </header>
        <div className="compact-device-list">
          {recentDevices.map((device) => <DeviceRow key={device.id} device={device} onCopy={onCopy} />)}
        </div>
      </section>
    </div>
  );
}
