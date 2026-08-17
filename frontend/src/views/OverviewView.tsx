import {
  AlertTriangle,
  CheckCircle2,
  Copy,
  Laptop,
  Monitor,
  Radio,
  ServerOff,
} from "lucide-react";
import type {
  AppSnapshot,
  PreferenceKey,
  RuntimeState,
} from "../lib/contracts";
import { IconButton, SettingRow, StatusBadge, Toggle, type Tone } from "../components/ui";
import { useI18n, type Translate } from "../lib/i18n";

function connectionSummary(runtime: RuntimeState, warningCount: number, t: Translate): { label: string; detail: string; tone: Tone } {
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
      : {
          label: t("connection.tunnelConnected"),
          detail: warningCount > 0 ? t("connection.warningCount", { count: warningCount }) : t("connection.localWarning"),
          tone: "warning",
        };
  }
  if (runtime.connection === "running") {
    if (runtime.control === "unreachable") {
      return { label: t("connection.tunnelConnected"), detail: t("connection.controlLimited"), tone: "warning" };
    }
    return { label: t("connection.connected"), detail: t("connection.tunnelHealthy"), tone: "success" };
  }
  return { label: t("connection.notConnected"), detail: t("connection.tunnelStopped"), tone: "neutral" };
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
  const healthWarnings = snapshot.healthWarnings ?? [];
  const status = connectionSummary(snapshot.runtime, healthWarnings.length, t);
  const endpoint = snapshot.endpoints.find((item) => item.id === snapshot.activeEndpointId);
  const profile = snapshot.profiles.find((item) => item.id === snapshot.activeProfileId);
  const endpointName = endpoint?.name ?? t("network.noneSelected");
  const accountName = profile?.account ?? t("account.noneSelected");
  const onlineDeviceCount = snapshot.devices.filter((item) => item.online).length;
  const exitNodeOptions = snapshot.devices.filter((device) => device.exitNodeOption);
  const selectedExitNodeMissing = Boolean(snapshot.preferences.exitNodeId) &&
    !exitNodeOptions.some((device) => device.id === snapshot.preferences.exitNodeId);
  const exitNodeDescription = exitNodeOptions.length === 0
    ? t("overview.exitNodeNoneApproved")
    : exitNodeOptions.some((device) => device.online)
      ? t("overview.exitNodeDescription")
      : t("overview.exitNodeAllOffline");
  const isTunnelRunning = ["running", "degraded"].includes(snapshot.runtime.connection);
  const canEnsureDaemon = snapshot.engine.canInstall || snapshot.engine.canStart ||
    (snapshot.engine.ownership === "managed" && snapshot.engine.payloadAvailable);
  const daemonActionLabel = snapshot.engine.ownership === "managed"
    ? t("daemon.repair")
    : snapshot.engine.canInstall ? t("daemon.install") : t("daemon.start");

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
          <p className="truncate" title={`${endpointName} · ${accountName}`}>
            {endpointName} · {accountName}
          </p>
        </div>
        <button
          className="online-device-shortcut"
          type="button"
          title={t("overview.viewOnlineDevices", { count: onlineDeviceCount })}
          aria-label={t("overview.viewOnlineDevices", { count: onlineDeviceCount })}
          onClick={onShowDevices}
        >
          <Monitor aria-hidden="true" size={17} />
          <span className="online-device-copy" aria-hidden="true">
            <span className="online-device-count">{onlineDeviceCount}</span>
            <span className="online-device-suffix"> {t("overview.onlineSuffix")}</span>
          </span>
        </button>
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

      {healthWarnings.length > 0 && (
        <div className="persistent-alert health-warning-alert" role="alert">
          <AlertTriangle aria-hidden="true" size={19} />
          <div>
            <strong>{t("overview.healthWarningTitle", { count: healthWarnings.length })}</strong>
            <span>{t("overview.healthWarningSource")}</span>
            <ul className="health-warning-list">
              {healthWarnings.map((warning, index) => <li key={`${index}-${warning}`}>{warning}</li>)}
            </ul>
          </div>
        </div>
      )}

      {snapshot.preferences.exitNodeId && !snapshot.preferences.allowLanAccess && (
        <div className="persistent-alert" role="alert">
          <AlertTriangle aria-hidden="true" size={19} />
          <div>
            <strong>{t("overview.lanBlockedTitle")}</strong>
            <span>{t("overview.lanBlockedDescription")}</span>
          </div>
          <button
            className="button secondary daemon-action"
            type="button"
            disabled={busy === "allowLanAccess"}
            onClick={() => onPreferenceChange("allowLanAccess", true)}
          >
            {t("overview.enableLAN")}
          </button>
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
            <h2 id="preferences-title">{t("overview.networkSettings")}</h2>
            <p>{t("overview.networkSettingsScope", { network: endpointName, account: accountName })}</p>
          </div>
        </header>
        <div className="setting-list">
          <div className="setting-group">
            <SettingRow
              title={t("overview.exitNode")}
              description={exitNodeDescription}
              control={
                <select
                  aria-label={t("overview.exitNode")}
                  value={snapshot.preferences.exitNodeId ?? ""}
                  disabled={busy === "exitNode"}
                  onChange={(event) => onExitNodeChange(event.target.value || null)}
                >
                  <option value="">{t("overview.doNotUse")}</option>
                  {exitNodeOptions.length === 0 && (
                    <option value="__no-approved-exit-node" disabled>{t("overview.exitNodeNoneOption")}</option>
                  )}
                  {selectedExitNodeMissing && (
                    <option value={snapshot.preferences.exitNodeId ?? ""} disabled>{t("overview.exitNodeUnavailable")}</option>
                  )}
                  {exitNodeOptions.map((device) => (
                    <option key={device.id} value={device.id} disabled={!device.online}>
                      {device.online ? device.name : t("overview.exitNodeOffline", { name: device.name })}
                    </option>
                  ))}
                </select>
              }
            />
            <SettingRow
              title={t("overview.allowLAN")}
              description={snapshot.preferences.exitNodeId ? t("overview.allowLANDescription") : t("overview.allowLANRequiresExitNode")}
              nested
              disabled={!snapshot.preferences.exitNodeId}
              control={<Toggle label={t("overview.allowLAN")} checked={snapshot.preferences.allowLanAccess} disabled={busy === "allowLanAccess" || !snapshot.preferences.exitNodeId} onChange={(value) => onPreferenceChange("allowLanAccess", value)} />}
            />
          </div>
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

    </div>
  );
}
