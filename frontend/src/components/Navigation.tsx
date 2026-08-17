import { CircleUserRound, Info, LayoutDashboard, Network, Settings } from "lucide-react";
import { useI18n, type MessageKey, type Translate } from "../lib/i18n";

export type ViewKey = "overview" | "devices" | "networks" | "settings" | "about";

const primaryItems = [
  { id: "overview" as const, label: "nav.overview" as MessageKey, icon: LayoutDashboard },
  { id: "devices" as const, label: "nav.devices" as MessageKey, icon: Network },
  { id: "networks" as const, label: "nav.networksShort" as MessageKey, icon: CircleUserRound },
  { id: "settings" as const, label: "nav.settings" as MessageKey, icon: Settings },
];

const aboutItem = { id: "about" as const, label: "nav.about" as MessageKey, icon: Info };

const viewTitleKeys: Record<ViewKey, MessageKey> = {
  overview: "nav.overview",
  devices: "nav.devices",
  networks: "nav.networks",
  settings: "nav.settings",
  about: "nav.about",
};

export const viewTitle = (view: ViewKey, t: Translate) => t(viewTitleKeys[view]);

export function PrimaryNavigation({
  current,
  daemonReady,
  onChange,
}: {
  current: ViewKey;
  daemonReady: boolean;
  onChange: (view: ViewKey) => void;
}) {
  const { t } = useI18n();
  const renderItem = ({ id, label: labelKey, icon: Icon }: (typeof primaryItems)[number] | typeof aboutItem) => {
    const label = t(labelKey);
    return (
      <button
        key={id}
        type="button"
        className={`navigation-item ${current === id ? "is-active" : ""}`}
        aria-current={current === id ? "page" : undefined}
        title={label}
        onClick={() => onChange(id)}
      >
        <Icon aria-hidden="true" size={19} strokeWidth={1.8} />
        <span>{label}</span>
      </button>
    );
  };

  return (
    <nav className="primary-navigation" aria-label={t("nav.main")}>
      <div className="brand-lockup" aria-label="HeadscaleClient">
        <span className="brand-symbol">
          <Network aria-hidden="true" size={19} strokeWidth={2} />
        </span>
        <span className="brand-name">HeadscaleClient</span>
      </div>
      <div className="navigation-items">
        {primaryItems.map(renderItem)}
      </div>
      <div className="navigation-utility">{renderItem(aboutItem)}</div>
      <div className="navigation-footer">
        <span className={`navigation-health-dot ${daemonReady ? "is-ready" : ""}`} aria-hidden="true" />
        <span>{daemonReady ? t("nav.daemonReady") : t("nav.daemonOffline")}</span>
      </div>
    </nav>
  );
}
