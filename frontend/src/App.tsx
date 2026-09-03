import { RefreshCw } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { AccountMenu } from "./components/AccountMenu";
import { PrimaryNavigation, viewTitle, type ViewKey } from "./components/Navigation";
import { ErrorState, LoadingState } from "./components/ui";
import { backend as defaultBackend, openExternalURL } from "./lib/backend";
import type {
  AppSettingKey,
  AppSnapshot,
  EndpointInput,
  HeadscaleBackend,
  LanguagePreference,
  PreferenceKey,
  ThemePreference,
} from "./lib/contracts";
import { createTranslator, I18nProvider, type Translate } from "./lib/i18n";
import { AboutView } from "./views/AboutView";
import { DevicesView } from "./views/DevicesView";
import { NetworksView } from "./views/NetworksView";
import { OverviewView } from "./views/OverviewView";
import { SettingsView } from "./views/SettingsView";

function connectionChangeMessage(snapshot: AppSnapshot, enabled: boolean, t: Translate): string {
  if (!enabled) return t("connection.disconnected");
  if (snapshot.runtime.control === "unreachable") {
    return t("connection.tunnelControlLimited");
  }
  if (snapshot.runtime.connection === "degraded") {
    return t("connection.tunnelLocalWarning");
  }
  if (snapshot.runtime.connection === "starting") return t("connection.connecting");
  return t("connection.connected");
}

export function App({ backendClient = defaultBackend }: { backendClient?: HeadscaleBackend }) {
  const [view, setView] = useState<ViewKey>("overview");
  const [snapshot, setSnapshot] = useState<AppSnapshot | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [toast, setToast] = useState<string | null>(null);
  const [reloadKey, setReloadKey] = useState(0);
  const language = snapshot?.settings.language ?? "zh-CN";
  const t = useMemo(() => createTranslator(language), [language]);

  useEffect(() => {
    let active = true;
    setLoadError(null);
    backendClient
      .getSnapshot()
      .then((nextSnapshot) => {
        if (active) setSnapshot(nextSnapshot);
      })
      .catch((error: unknown) => {
        if (active) {
          setLoadError(error instanceof Error ? error.message : "未知错误");
        }
      });
    return () => {
      active = false;
    };
  }, [backendClient, reloadKey]);

  useEffect(
    () => backendClient.subscribe(setSnapshot, setToast),
    [backendClient],
  );

  useEffect(
    () => backendClient.subscribeNavigation(setView),
    [backendClient],
  );

  useEffect(() => {
    if (!snapshot) return;
    document.documentElement.dataset.theme = snapshot.settings.theme;
    document.documentElement.lang = snapshot.settings.language;
  }, [snapshot]);

  useEffect(() => {
    if (!toast) return;
    const timer = window.setTimeout(() => setToast(null), 3200);
    return () => window.clearTimeout(timer);
  }, [toast]);

  const activeEndpoint = useMemo(
    () => snapshot?.endpoints.find((item) => item.id === snapshot.activeEndpointId),
    [snapshot],
  );
  const activeProfile = useMemo(
    () => snapshot?.profiles.find((item) => item.id === snapshot.activeProfileId),
    [snapshot],
  );

  const applySnapshotAction = async (
    key: string,
    action: () => Promise<AppSnapshot>,
    successMessage?: string,
  ) => {
    setBusy(key);
    try {
      const nextSnapshot = await action();
      setSnapshot(nextSnapshot);
      if (successMessage) setToast(successMessage);
      return nextSnapshot;
    } catch (error) {
      setToast(error instanceof Error ? error.message : t("common.operationFailed"));
      throw error;
    } finally {
      setBusy(null);
    }
  };

  const copyText = async (value: string, label = t("common.copied")) => {
    try {
      await navigator.clipboard.writeText(value);
      setToast(label);
    } catch {
      setToast(t("common.copyFailed"));
    }
  };

  if (!snapshot && !loadError) return <I18nProvider language={language}><LoadingState /></I18nProvider>;
  if (loadError) return <I18nProvider language={language}><ErrorState message={loadError} onRetry={() => setReloadKey((value) => value + 1)} /></I18nProvider>;
  if (!snapshot) return null;

  const setPreference = (key: PreferenceKey, value: boolean) =>
    void applySnapshotAction(key, () => backendClient.setPreference(key, value)).catch(() => undefined);
  const setAppSetting = (key: AppSettingKey, value: boolean) =>
    void applySnapshotAction(key, () => backendClient.setAppSetting(key, value)).catch(() => undefined);
  const setTheme = (theme: ThemePreference) =>
    void applySnapshotAction("theme", () => backendClient.setTheme(theme)).catch(() => undefined);
  const setLanguage = (nextLanguage: LanguagePreference) =>
    void applySnapshotAction("language", () => backendClient.setLanguage(nextLanguage)).catch(() => undefined);
  const saveEndpoint = (input: EndpointInput) =>
    applySnapshotAction("endpoint", () => backendClient.saveEndpoint(input), t("network.serverSaved"));
  const deleteEndpoint = (endpointId: string) =>
    applySnapshotAction("endpoint", () => backendClient.deleteEndpoint(endpointId), t("network.serverDeleted"));
  const switchProfile = (profileId: string) =>
    applySnapshotAction("profile", () => backendClient.switchProfile(profileId), t("account.switched"));
  const refreshDevices = () =>
    applySnapshotAction("devices-refresh", () => backendClient.getSnapshot(), t("device.refreshed")).then(() => undefined);

  return (
    <I18nProvider language={language}>
      <div className="app-shell">
        <PrimaryNavigation
          current={view}
          daemonReady={snapshot.runtime.daemon === "ready"}
          onChange={setView}
        />

      <main className="app-content">
        <header className="app-header">
          <div>
            <h1>{viewTitle(view, t)}</h1>
            <span className="header-context truncate" title={`${activeEndpoint?.name ?? t("network.noneSelected")} · ${activeProfile?.account ?? t("account.noneSelected")}`}>
              {activeEndpoint?.name ?? t("network.noneSelected")} · {activeProfile?.account ?? t("account.noneSelected")}
            </span>
          </div>
          <AccountMenu
            profiles={snapshot.profiles}
            endpoints={snapshot.endpoints}
            activeProfileId={snapshot.activeProfileId}
            switching={busy === "profile"}
            onSwitch={async (profileId) => { await switchProfile(profileId); }}
            onManage={() => setView("networks")}
          />
        </header>

        <div className="page-body">
          {view === "overview" && (
            <OverviewView
              snapshot={snapshot}
              busy={busy}
              onConnectionChange={(enabled) => void applySnapshotAction("connection", () => backendClient.setConnection(enabled))
                .then((nextSnapshot) => setToast(connectionChangeMessage(nextSnapshot, enabled, t)))
                .catch(() => undefined)}
              onEnsureDaemon={() => void applySnapshotAction("daemon", () => backendClient.ensureDaemon(), t("daemon.ready")).catch(() => undefined)}
              onPreferenceChange={setPreference}
              onExitNodeChange={(deviceId) => void applySnapshotAction("exitNode", () => backendClient.setExitNode(deviceId)).catch(() => undefined)}
              onCopy={copyText}
              onShowDevices={() => setView("devices")}
            />
          )}
          {view === "devices" && (
            <DevicesView
              key={`${snapshot.activeEndpointId ?? "none"}:${snapshot.activeProfileId ?? "none"}`}
              devices={snapshot.devices}
              networkName={activeEndpoint?.name ?? t("network.noneSelected")}
              accountName={activeProfile?.account ?? t("common.notLoggedIn")}
              refreshing={busy === "devices-refresh"}
              onCopy={copyText}
              onRefresh={refreshDevices}
              onPing={async (deviceId) => {
                try {
                  return await backendClient.pingDevice(deviceId);
                } catch (error) {
                  setToast(error instanceof Error ? error.message : t("device.pingFailed"));
                  throw error;
                }
              }}
            />
          )}
          {view === "networks" && (
            <NetworksView
              endpoints={snapshot.endpoints}
              profiles={snapshot.profiles}
              activeEndpointId={snapshot.activeEndpointId}
              activeProfileId={snapshot.activeProfileId}
              onSaveEndpoint={async (input) => { await saveEndpoint(input); }}
              onDeleteEndpoint={async (endpointId) => { await deleteEndpoint(endpointId); }}
              onSwitchProfile={async (profileId) => { await switchProfile(profileId); }}
              onLogout={async () => { await applySnapshotAction("logout", () => backendClient.logout(), t("account.loggedOut")); }}
              onBeginLogin={async (endpointId) => {
                try {
                  return await backendClient.beginLogin(endpointId);
                } catch (error) {
                  setToast(error instanceof Error ? error.message : t("endpoint.loginFailure"));
                  throw error;
                }
              }}
              onOpenURL={openExternalURL}
            />
          )}
          {view === "settings" && (
            <SettingsView
              settings={snapshot.settings}
              runtime={snapshot.runtime}
              diagnostics={snapshot.diagnostics}
              engine={snapshot.engine}
              busy={busy}
              onSettingChange={setAppSetting}
              onThemeChange={setTheme}
              onLanguageChange={setLanguage}
              onCopyDiagnostics={() => void copyText(JSON.stringify({
                runtime: snapshot.runtime,
                healthNotices: snapshot.healthNotices,
                engine: snapshot.engine,
                diagnostics: snapshot.diagnostics,
                preferences: snapshot.preferences,
                activeEndpointId: snapshot.activeEndpointId,
                activeProfileId: snapshot.activeProfileId,
              }, null, 2), t("settings.summaryCopied"))}
              onEnsureDaemon={() => void applySnapshotAction("daemon", () => backendClient.ensureDaemon(), t("daemon.ready")).catch(() => undefined)}
            />
          )}
          {view === "about" && (
            <AboutView version={snapshot.diagnostics.appVersion} onOpenURL={openExternalURL} />
          )}
        </div>
      </main>

      <div className="connection-announcer sr-only" aria-live="polite">
        {snapshot.runtime.connection}
      </div>

      {toast && (
        <div className="toast-region" role="status" aria-live="polite">
          <RefreshCw aria-hidden="true" size={17} />
          <span>{toast}</span>
        </div>
      )}
      </div>
    </I18nProvider>
  );
}

export default App;
