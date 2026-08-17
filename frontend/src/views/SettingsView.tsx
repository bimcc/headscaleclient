import { Clipboard, ShieldAlert, Wrench } from "lucide-react";
import type {
  AppSettingKey,
  AppSettings,
  Diagnostics,
  EngineStatus,
  LanguagePreference,
  RuntimeState,
  ThemePreference,
} from "../lib/contracts";
import { SettingRow, StatusBadge, Toggle } from "../components/ui";
import { useI18n, type MessageKey } from "../lib/i18n";

const daemonLabelKeys: Record<RuntimeState["daemon"], MessageKey> = {
  unknown: "settings.daemonUnknown",
  missing: "settings.daemonMissing",
  stopped: "settings.daemonStopped",
  ready: "settings.daemonReady",
  unauthorized: "settings.daemonUnauthorized",
  incompatible: "settings.daemonIncompatible",
};

const engineLabelKeys: Record<EngineStatus["ownership"], MessageKey> = {
  unknown: "settings.daemonUnknown",
  managed: "settings.engineManaged",
  external: "settings.engineExternal",
  prepared: "settings.enginePrepared",
  missing: "settings.engineMissing",
};

export function SettingsView({
  settings,
  runtime,
  diagnostics,
  engine,
  busy,
  onSettingChange,
  onThemeChange,
  onLanguageChange,
  onCopyDiagnostics,
  onEnsureDaemon,
}: {
  settings: AppSettings;
  runtime: RuntimeState;
  diagnostics: Diagnostics;
  engine: EngineStatus;
  busy: string | null;
  onSettingChange: (key: AppSettingKey, value: boolean) => void;
  onThemeChange: (theme: ThemePreference) => void;
  onLanguageChange: (language: LanguagePreference) => void;
  onCopyDiagnostics: () => void;
  onEnsureDaemon: () => void;
}) {
  const { t } = useI18n();
  const canEnsureDaemon = engine.canInstall || engine.canStart ||
    (engine.ownership === "managed" && engine.payloadAvailable);
  return (
    <div className="view-stack settings-view">
      <section className="section-block" aria-labelledby="application-settings-title">
        <header className="section-header">
          <div>
            <h2 id="application-settings-title">{t("settings.general")}</h2>
            <p>{t("settings.generalHint")}</p>
          </div>
        </header>
        <div className="setting-list">
          <SettingRow
            title={t("settings.launchAtLogin")}
            description={t("settings.launchAtLoginHint")}
            control={<Toggle label={t("settings.launchAtLogin")} checked={settings.launchAtLogin} disabled={busy === "launchAtLogin"} onChange={(value) => onSettingChange("launchAtLogin", value)} />}
          />
          <SettingRow
            title={t("settings.closeToTray")}
            description={t("settings.closeToTrayHint")}
            control={<Toggle label={t("settings.closeToTray")} checked={settings.closeToTray} disabled={busy === "closeToTray"} onChange={(value) => onSettingChange("closeToTray", value)} />}
          />
          <SettingRow
            title={t("settings.language")}
            description={t("settings.languageHint")}
            control={
              <select aria-label={t("settings.language")} value={settings.language} disabled={busy === "language"} onChange={(event) => onLanguageChange(event.target.value as LanguagePreference)}>
                <option value="zh-CN">{t("settings.chinese")}</option>
                <option value="en-US">{t("settings.english")}</option>
              </select>
            }
          />
          <SettingRow
            title={t("settings.theme")}
            description={t("settings.themeHint")}
            control={
              <select aria-label={t("settings.theme")} value={settings.theme} disabled={busy === "theme"} onChange={(event) => onThemeChange(event.target.value as ThemePreference)}>
                <option value="system">{t("settings.themeSystem")}</option>
                <option value="light">{t("settings.themeLight")}</option>
                <option value="dark">{t("settings.themeDark")}</option>
              </select>
            }
          />
        </div>
      </section>

      <section className="section-block" aria-labelledby="runtime-diagnostics-title">
        <header className="section-header">
          <div>
            <h2 id="runtime-diagnostics-title">{t("settings.runtimeDiagnostics")}</h2>
            <p>{t("settings.runtimeDiagnosticsHint")}</p>
          </div>
          <div className="section-actions">
            <StatusBadge tone={runtime.daemon === "ready" ? "success" : "danger"}>{t(daemonLabelKeys[runtime.daemon])}</StatusBadge>
            <button className="button secondary with-icon" type="button" onClick={onCopyDiagnostics}>
              <Clipboard aria-hidden="true" size={17} /> {t("settings.copySummary")}
            </button>
          </div>
        </header>
        <dl className="diagnostics-grid">
          <div><dt>Daemon</dt><dd>{diagnostics.daemonVersion}</dd></div>
          <div><dt>{t("settings.engineSource")}</dt><dd>{t(engineLabelKeys[engine.ownership])}</dd></div>
          <div><dt>{t("settings.bundledVersion")}</dt><dd>{engine.bundledVersion || t("common.notProvided")}</dd></div>
          <div><dt>LocalAPI</dt><dd>{diagnostics.localApi}</dd></div>
          <div><dt>{t("settings.platform")}</dt><dd>{diagnostics.platform}</dd></div>
        </dl>
        {runtime.daemon !== "ready" && (
          <div className="inline-health-warning">
            <ShieldAlert aria-hidden="true" size={19} />
            <span>{engine.canInstall ? t("settings.installHint") : t("settings.repairHint")}</span>
            {canEnsureDaemon && (
              <button className="button secondary with-icon" type="button" disabled={busy === "daemon"} onClick={onEnsureDaemon}>
                <Wrench aria-hidden="true" size={16} />
                {engine.ownership === "external" ? t("settings.startService") : t("settings.repairService")}
              </button>
            )}
          </div>
        )}
      </section>

    </div>
  );
}
