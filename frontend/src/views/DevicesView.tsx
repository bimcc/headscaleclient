import { Copy, Globe2, Search, Send, Server, Smartphone, UserRound, X } from "lucide-react";
import { useMemo, useState } from "react";
import type { PeerDevice, PingResult } from "../lib/contracts";
import { EmptyState, IconButton, StatusBadge } from "../components/ui";
import { useI18n, type Translate } from "../lib/i18n";

type DeviceFilter = "all" | "online" | "offline";

function DeviceIcon({ os }: { os: string }) {
  return os.toLowerCase().includes("android") || os.toLowerCase().includes("ios") ? (
    <Smartphone aria-hidden="true" size={19} />
  ) : (
    <Server aria-hidden="true" size={19} />
  );
}

function deviceDisplayName(device: PeerDevice, t: Translate) {
  return device.name.trim() || device.dnsName.trim() || device.addresses[0] || t("device.unnamed");
}

function primaryAddress(device: PeerDevice) {
  return device.addresses[0] ?? "";
}

function pathLabel(device: PeerDevice, t: Translate) {
  if (device.connectionType === "direct") return t("device.direct");
  if (device.connectionType === "relay") return device.relayRegion ? `DERP · ${device.relayRegion}` : `DERP · ${t("device.unknownRegion")}`;
  return t("device.unavailable");
}

export function DevicesView({
  devices,
  networkName,
  accountName,
  onCopy,
  onPing,
}: {
  devices: PeerDevice[];
  networkName: string;
  accountName: string;
  onCopy: (value: string, label?: string) => void;
  onPing: (deviceId: string) => Promise<PingResult>;
}) {
  const { t } = useI18n();
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<DeviceFilter>("all");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [pinging, setPinging] = useState(false);
  const [pingResult, setPingResult] = useState<PingResult | null>(null);

  const filtered = useMemo(() => {
    const normalized = query.trim().toLocaleLowerCase();
    return devices.filter((device) => {
      const matchesFilter =
        filter === "all" || (filter === "online" ? device.online : !device.online);
      const matchesQuery =
        !normalized ||
        [device.name, device.dnsName, device.owner, device.os, ...device.addresses]
          .join(" ")
          .toLocaleLowerCase()
          .includes(normalized);
      return matchesFilter && matchesQuery;
    });
  }, [devices, filter, query]);

  const selected = devices.find((device) => device.id === selectedId) ?? null;

  const pingSelected = async () => {
    if (!selected) return;
    setPinging(true);
    setPingResult(null);
    try {
      setPingResult(await onPing(selected.id));
    } catch {
      setPingResult(null);
    } finally {
      setPinging(false);
    }
  };

  return (
    <div className={`devices-layout ${selected ? "has-detail" : ""}`}>
      <section className="devices-content" aria-labelledby="device-list-title">
        <div className="device-scope" aria-label={t("device.scope")}>
          <div className="device-scope-item">
            <Globe2 aria-hidden="true" size={17} />
            <span><small>{t("device.currentNetwork")}</small><strong className="truncate" title={networkName}>{networkName}</strong></span>
          </div>
          <div className="device-scope-divider" aria-hidden="true" />
          <div className="device-scope-item">
            <UserRound aria-hidden="true" size={17} />
            <span><small>{t("device.currentAccount")}</small><strong className="truncate" title={accountName}>{accountName}</strong></span>
          </div>
        </div>
        <div className="toolbar-row">
          <label className="search-field">
            <Search aria-hidden="true" size={18} />
            <span className="sr-only">{t("device.search")}</span>
            <input
              type="search"
              value={query}
              placeholder={t("device.searchPlaceholder")}
              onChange={(event) => setQuery(event.target.value)}
            />
          </label>
          <div className="segmented-control" aria-label={t("device.filter")}>
            {(["all", "online", "offline"] as const).map((item) => (
              <button
                key={item}
                type="button"
                aria-pressed={filter === item}
                onClick={() => setFilter(item)}
              >
                {item === "all" ? t("device.all") : item === "online" ? t("device.online") : t("device.offline")}
              </button>
            ))}
          </div>
        </div>

        <div className="table-meta">
          <h2 id="device-list-title">{t("device.list")}</h2>
          <span>{filtered.length} / {devices.length}</span>
        </div>

        {filtered.length === 0 ? (
          <EmptyState title={t("device.noMatch")} message={t("device.noMatchHint")} />
        ) : (
          <div className="device-table" role="table" aria-label={t("device.table")}>
            <div className="device-table-header" role="row">
              <span role="columnheader">{t("device.table")}</span>
              <span role="columnheader">{t("device.virtualAddress")}</span>
              <span role="columnheader">{t("device.currentPath")}</span>
              <span role="columnheader">{t("device.lastOnline")}</span>
              <span aria-hidden="true" />
            </div>
            {filtered.map((device) => (
              <button
                className={`device-table-row ${selectedId === device.id ? "is-selected" : ""}`}
                type="button"
                role="row"
                key={device.id}
                onClick={() => {
                  setSelectedId(device.id);
                  setPingResult(null);
                }}
              >
                  <span className="device-cell-main" role="cell">
                  <span className="device-os-icon"><DeviceIcon os={device.os} /></span>
                  <span className="truncate">
                    <strong>{deviceDisplayName(device, t)}</strong>
                    <small>{device.owner || t("device.unknownOwner")}</small>
                  </span>
                </span>
                <span className="device-address-summary" role="cell" title={device.addresses.join("\n")}>
                  <span className="mono truncate">{primaryAddress(device) || t("common.unassigned")}</span>
                  {device.addresses.length > 1 && <small>{t("device.moreAddresses", { count: device.addresses.length - 1 })}</small>}
                </span>
                <span role="cell">
                  <StatusBadge tone={device.online ? "success" : "neutral"}>
                    {device.connectionType === "direct" ? t("device.direct") : device.connectionType === "relay" ? t("device.relay") : t("device.offline")}
                  </StatusBadge>
                </span>
                <span className="muted" role="cell">{device.lastSeen || t("device.neverOnline")}</span>
                <span className="row-chevron" aria-hidden="true">›</span>
              </button>
            ))}
          </div>
        )}
      </section>

      {selected && (
        <aside className="detail-drawer" aria-label={t("device.details", { name: deviceDisplayName(selected, t) })}>
          <header className="drawer-header">
            <div className="drawer-identity">
              <span className="device-os-icon large"><DeviceIcon os={selected.os} /></span>
              <div className="truncate">
                <h2 title={deviceDisplayName(selected, t)}>{deviceDisplayName(selected, t)}</h2>
                <p>{selected.os}</p>
              </div>
            </div>
            <IconButton label={t("device.closeDetails")} icon={X} onClick={() => setSelectedId(null)} />
          </header>

          <div className="drawer-status">
            <span className={`presence-dot ${selected.online ? "online" : "offline"}`} aria-hidden="true" />
            <strong>{selected.online ? t("device.online") : t("device.offline")}</strong>
            <span>{selected.lastSeen || t("device.neverOnline")}</span>
          </div>

          <dl className="detail-list">
            <div>
              <dt>MagicDNS</dt>
              <dd><span className="truncate" title={selected.dnsName}>{selected.dnsName || t("common.unassigned")}</span><IconButton label={t("device.copyMagicDNS")} icon={Copy} disabled={!selected.dnsName} onClick={() => onCopy(selected.dnsName)} /></dd>
            </div>
            {selected.addresses.length > 0 ? selected.addresses.map((address) => (
              <div key={address}>
                <dt>{address.includes(":") ? t("device.ipv6Address") : t("device.ipv4Address")}</dt>
                <dd><span className="mono truncate" title={address}>{address}</span><IconButton label={t("device.copy")} icon={Copy} onClick={() => onCopy(address)} /></dd>
              </div>
            )) : (
              <div><dt>{t("device.virtualAddress")}</dt><dd>{t("common.unassigned")}</dd></div>
            )}
            <div>
              <dt>{t("device.currentPath")}</dt>
              <dd>{pathLabel(selected, t)}</dd>
            </div>
            <div>
              <dt>{t("device.owner")}</dt>
              <dd>{selected.owner || t("device.unknownOwner")}</dd>
            </div>
          </dl>

          {selected.tags.length > 0 && (
            <div className="tag-list" aria-label={t("device.tags")}>
              {selected.tags.map((tag) => <span key={tag}>{tag}</span>)}
            </div>
          )}

          <div className="drawer-actions">
            <button className="button primary with-icon" type="button" disabled={!selected.online || pinging} onClick={pingSelected}>
              <Send aria-hidden="true" size={17} /> {pinging ? t("device.pinging") : "Ping"}
            </button>
            {pingResult && (
              <span className="ping-result" role="status">{t("device.pingResult", { latency: pingResult.latencyMs, path: pingResult.via === "direct" ? t("device.direct") : t("device.relay") })}</span>
            )}
          </div>
        </aside>
      )}
    </div>
  );
}
