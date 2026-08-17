import { AlertTriangle, Check, ExternalLink, Globe2, LogIn, LogOut, Pencil, Plus, ShieldCheck, Trash2 } from "lucide-react";
import { useEffect, useState, type FormEvent } from "react";
import type {
  Endpoint,
  EndpointInput,
  LoginProfile,
  LoginResult,
} from "../lib/contracts";
import { normalizeEndpointURL } from "../lib/endpointUrl";
import { EmptyState, IconButton, Modal, StatusBadge } from "../components/ui";
import { useI18n, type Translate } from "../lib/i18n";

function endpointKindLabel(kind: Endpoint["kind"], t: Translate) {
  if (kind === "headscale") return "Headscale";
  if (kind === "tailscale") return "Tailscale";
  return t("endpoint.compatible");
}

function endpointStatus(
  endpoint: Endpoint,
  active: boolean,
  accountCount: number,
  t: Translate,
): { label: string; tone: "neutral" | "success" | "warning" | "danger" } {
  if (active) {
    if (endpoint.status === "reachable") return { label: t("endpoint.reachable"), tone: "success" };
    if (endpoint.status === "unreachable") return { label: t("endpoint.unreachable"), tone: "danger" };
    return { label: t("endpoint.statusUnknown"), tone: "warning" };
  }
  if (accountCount > 0) return { label: t("endpoint.checkOnLogin"), tone: "neutral" };
  return { label: t("common.notLoggedIn"), tone: "neutral" };
}

function EndpointDialog({
  endpoint,
  onClose,
  onSave,
}: {
  endpoint: Endpoint | null;
  onClose: () => void;
  onSave: (input: EndpointInput) => Promise<void>;
}) {
  const { t } = useI18n();
  const [name, setName] = useState(endpoint?.name ?? "");
  const [url, setUrl] = useState(endpoint?.url ?? "");
  const [kind, setKind] = useState<Endpoint["kind"]>(endpoint?.kind ?? "headscale");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setError("");
    let normalizedUrl: string;
    try {
      normalizedUrl = normalizeEndpointURL(url);
    } catch {
      setError(t("endpoint.invalidAddress"));
      return;
    }
    setSaving(true);
    try {
      await onSave({
        id: endpoint?.id,
        name: name.trim(),
        url: normalizedUrl,
        kind,
        customCa: false,
      });
      onClose();
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : t("endpoint.saveFailed"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal title={endpoint ? t("endpoint.edit") : t("endpoint.add")} onClose={onClose}>
      <form className="form-stack" onSubmit={submit}>
        <label>
          <span>{t("endpoint.name")}</span>
          <input required value={name} onChange={(event) => setName(event.target.value)} placeholder={t("endpoint.namePlaceholder")} autoFocus />
        </label>
        <label>
          <span>{t("endpoint.address")}</span>
          <input required type="url" value={url} onChange={(event) => setUrl(event.target.value)} placeholder="https://headscale.example.com" />
        </label>
        <label>
          <span>{t("endpoint.type")}</span>
          <select value={kind} onChange={(event) => setKind(event.target.value as Endpoint["kind"])}>
            <option value="headscale">Headscale</option>
            <option value="tailscale">{t("endpoint.tailscaleOfficial")}</option>
            <option value="compatible">{t("endpoint.otherCompatible")}</option>
          </select>
        </label>
        {error && <p className="form-error" role="alert">{error}</p>}
        <div className="form-actions">
          <button className="button primary" type="submit" disabled={saving}>{saving ? t("endpoint.saving") : t("endpoint.saveServer")}</button>
        </div>
      </form>
    </Modal>
  );
}

function DeleteEndpointDialog({
  endpoint,
  onClose,
  onDelete,
}: {
  endpoint: Endpoint;
  onClose: () => void;
  onDelete: (endpointId: string) => Promise<void>;
}) {
  const { t } = useI18n();
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState("");

  const remove = async () => {
    setDeleting(true);
    setError("");
    try {
      await onDelete(endpoint.id);
      onClose();
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : t("endpoint.deleteFailed"));
    } finally {
      setDeleting(false);
    }
  };

  return (
    <Modal title={t("endpoint.deleteTitle")} onClose={onClose}>
      <div className="confirm-dialog">
        <p>{t("endpoint.deleteConfirm", { name: endpoint.name })}</p>
        {error && <p className="form-error" role="alert">{error}</p>}
        <div className="form-actions">
          <button className="button secondary" type="button" disabled={deleting} onClick={onClose}>{t("common.cancel")}</button>
          <button className="button danger" type="button" disabled={deleting} onClick={() => void remove()}>
            <Trash2 aria-hidden="true" size={16} /> {deleting ? t("common.deleting") : t("common.delete")}
          </button>
        </div>
      </div>
    </Modal>
  );
}

function LogoutDialog({
  profile,
  endpointName,
  onClose,
  onLogout,
}: {
  profile: LoginProfile;
  endpointName: string;
  onClose: () => void;
  onLogout: () => Promise<void>;
}) {
  const { t } = useI18n();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const logout = async () => {
    setBusy(true);
    setError("");
    try {
      await onLogout();
      onClose();
    } catch (logoutError) {
      setError(logoutError instanceof Error ? logoutError.message : t("profile.logoutFailed"));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal title={t("profile.logoutTitle")} onClose={onClose}>
      <div className="confirm-dialog">
        <p>
          {t("profile.logoutConfirm", { name: profile.displayName, endpoint: endpointName })}
        </p>
        <div className="identity-removal-warning">
          <AlertTriangle aria-hidden="true" size={18} />
          <p>{t("profile.logoutReauth")}</p>
        </div>
        {error && <p className="form-error" role="alert">{error}</p>}
        <div className="form-actions">
          <button className="button secondary" type="button" disabled={busy} onClick={onClose}>{t("common.cancel")}</button>
          <button className="button danger" type="button" disabled={busy} onClick={() => void logout()}>
            <LogOut aria-hidden="true" size={16} /> {busy ? t("profile.loggingOut") : t("profile.confirmLogout")}
          </button>
        </div>
      </div>
    </Modal>
  );
}

type NetworksViewProps = {
  endpoints: Endpoint[];
  profiles: LoginProfile[];
  activeEndpointId: string | null;
  activeProfileId: string | null;
  onSaveEndpoint: (input: EndpointInput) => Promise<void>;
  onDeleteEndpoint: (endpointId: string) => Promise<void>;
  onSwitchProfile: (profileId: string) => Promise<void>;
  onLogout: () => Promise<void>;
  onBeginLogin: (endpointId: string) => Promise<LoginResult>;
  onOpenURL: (url: string) => Promise<void>;
};

function profileDisplayName(profile: LoginProfile, t: Translate) {
  return profile.displayName.trim() || profile.account.trim() || t("profile.unnamed");
}

function profileStatusLabel(profile: LoginProfile, active: boolean, t: Translate) {
  if (active && profile.state === "ready") return t("profile.inUse");
  if (profile.state === "ready") return t("profile.loggedIn");
  return profile.state === "login-required" ? t("profile.loginRequired") : t("profile.approvalRequired");
}

export function NetworksView({
  endpoints,
  profiles,
  activeEndpointId,
  activeProfileId,
  onSaveEndpoint,
  onDeleteEndpoint,
  onSwitchProfile,
  onLogout,
  onBeginLogin,
  onOpenURL,
}: NetworksViewProps) {
  const { t } = useI18n();
  const [endpointDialog, setEndpointDialog] = useState<Endpoint | null | undefined>(undefined);
  const [deleteEndpoint, setDeleteEndpoint] = useState<Endpoint | null>(null);
  const [logoutProfile, setLogoutProfile] = useState<LoginProfile | null>(null);
  const [switchingId, setSwitchingId] = useState<string | null>(null);
  const [loginState, setLoginState] = useState<LoginResult | null>(null);
  const [loginFailure, setLoginFailure] = useState<{ endpointId: string; message: string } | null>(null);
  const [loginBusyId, setLoginBusyId] = useState<string | null>(null);
  const [selectedEndpointId, setSelectedEndpointId] = useState<string | null>(activeEndpointId ?? endpoints[0]?.id ?? null);

  useEffect(() => {
    if (selectedEndpointId && endpoints.some((endpoint) => endpoint.id === selectedEndpointId)) return;
    setSelectedEndpointId(activeEndpointId ?? endpoints[0]?.id ?? null);
  }, [activeEndpointId, endpoints, selectedEndpointId]);

  const selectedEndpoint = endpoints.find((endpoint) => endpoint.id === selectedEndpointId) ?? null;
  const selectedProfiles = selectedEndpoint
    ? profiles.filter((profile) => profile.endpointId === selectedEndpoint.id)
    : [];
  const selectedStatus = selectedEndpoint
    ? endpointStatus(selectedEndpoint, selectedEndpoint.id === activeEndpointId, selectedProfiles.length, t)
    : null;

  const selectEndpoint = (endpointId: string) => {
    setSelectedEndpointId(endpointId);
    setLoginState(null);
    setLoginFailure(null);
  };

  const switchProfile = async (profileId: string) => {
    setSwitchingId(profileId);
    try {
      await onSwitchProfile(profileId);
    } catch {
      // The app shell reports backend failures through the toast region.
    } finally {
      setSwitchingId(null);
    }
  };

  const login = async (endpointId: string) => {
    selectEndpoint(endpointId);
    setLoginState(null);
    setLoginFailure(null);
    setLoginBusyId(endpointId);
    try {
      const result = await onBeginLogin(endpointId);
      setLoginState(result);
      void onOpenURL(result.authUrl).catch(() => undefined);
    } catch (loginError) {
      setLoginState(null);
      setLoginFailure({
        endpointId,
        message: loginError instanceof Error ? loginError.message : t("endpoint.loginFailure"),
      });
    } finally {
      setLoginBusyId(null);
    }
  };

  if (endpoints.length === 0) {
    return (
      <div className="view-stack">
        <section className="section-block" aria-labelledby="endpoints-title">
          <header className="section-header">
            <div>
              <h2 id="endpoints-title">{t("endpoint.controlServers")}</h2>
              <p>{t("endpoint.noneSaved")}</p>
            </div>
            <button className="button secondary with-icon" type="button" onClick={() => setEndpointDialog(null)}>
              <Plus aria-hidden="true" size={17} /> {t("endpoint.addShort")}
            </button>
          </header>
          <EmptyState
            title={t("endpoint.noneAdded")}
            message={t("endpoint.addHint")}
            action={<button className="button primary" type="button" onClick={() => setEndpointDialog(null)}>{t("endpoint.addServer")}</button>}
          />
        </section>
        {endpointDialog !== undefined && <EndpointDialog endpoint={endpointDialog} onClose={() => setEndpointDialog(undefined)} onSave={onSaveEndpoint} />}
      </div>
    );
  }

  return (
    <div className="network-management">
      <aside className="endpoint-sidebar" aria-labelledby="endpoints-title">
        <header className="endpoint-sidebar-header">
          <div>
            <h2 id="endpoints-title">{t("endpoint.controlServers")}</h2>
            <p>{t("endpoint.savedCount", { count: endpoints.length })}</p>
          </div>
          <IconButton label={t("endpoint.add")} icon={Plus} onClick={() => setEndpointDialog(null)} />
        </header>
        <div className="endpoint-sidebar-list" role="listbox" aria-label={t("endpoint.list")}>
          {endpoints.map((endpoint, endpointIndex) => {
            const accountCount = profiles.filter((profile) => profile.endpointId === endpoint.id).length;
            const active = endpoint.id === activeEndpointId;
            const status = endpointStatus(endpoint, active, accountCount, t);
            return (
              <button
                className={`endpoint-nav-item ${endpoint.id === selectedEndpointId ? "is-selected" : ""}`}
                type="button"
                role="option"
                aria-selected={endpoint.id === selectedEndpointId}
                key={endpoint.id}
                onClick={() => selectEndpoint(endpoint.id)}
                onKeyDown={(event) => {
                  let nextIndex = endpointIndex;
                  if (event.key === "ArrowDown" || event.key === "ArrowRight") nextIndex = (endpointIndex + 1) % endpoints.length;
                  else if (event.key === "ArrowUp" || event.key === "ArrowLeft") nextIndex = (endpointIndex - 1 + endpoints.length) % endpoints.length;
                  else if (event.key === "Home") nextIndex = 0;
                  else if (event.key === "End") nextIndex = endpoints.length - 1;
                  else return;
                  event.preventDefault();
                  selectEndpoint(endpoints[nextIndex]!.id);
                  event.currentTarget.parentElement?.querySelectorAll<HTMLButtonElement>(".endpoint-nav-item")[nextIndex]?.focus();
                }}
              >
                <span className="endpoint-icon"><Globe2 aria-hidden="true" size={19} /></span>
                <span className="endpoint-nav-copy">
                  <strong className="truncate" title={endpoint.name}>{endpoint.name}</strong>
                  <span className="truncate mono" title={endpoint.url}>{endpoint.url}</span>
                  <span>{endpointKindLabel(endpoint.kind, t)} · {t("endpoint.accountCount", { count: accountCount })}</span>
                </span>
                <span className={`endpoint-health-dot ${status.tone}`} aria-label={status.label} title={status.label} />
              </button>
            );
          })}
        </div>
      </aside>

      {selectedEndpoint && (
        <section className="endpoint-detail" aria-labelledby="selected-endpoint-title">
          <header className="endpoint-detail-header">
            <div className="endpoint-detail-identity">
              <span className="endpoint-icon large"><Globe2 aria-hidden="true" size={21} /></span>
              <div className="truncate">
                <div className="endpoint-detail-title-row">
                  <h2 id="selected-endpoint-title" className="truncate" title={selectedEndpoint.name}>{selectedEndpoint.name}</h2>
                  {selectedEndpoint.id === activeEndpointId && <StatusBadge tone="success">{t("endpoint.currentNetwork")}</StatusBadge>}
                  <StatusBadge tone={selectedEndpoint.id === activeEndpointId ? selectedStatus?.tone : "neutral"}>
                    {selectedEndpoint.id === activeEndpointId ? selectedStatus?.label : t("endpoint.checkOnLogin")}
                  </StatusBadge>
                </div>
                <p className="mono truncate" title={selectedEndpoint.url}>{selectedEndpoint.url}</p>
              </div>
            </div>
            <div className="endpoint-detail-actions">
              <button
                className="button primary with-icon"
                type="button"
                aria-label={`${selectedProfiles.length > 0 ? t("endpoint.addAccount") : t("endpoint.login")} ${selectedEndpoint.name}`}
                disabled={loginBusyId !== null || switchingId !== null}
                onClick={() => void login(selectedEndpoint.id)}
              >
                <LogIn aria-hidden="true" size={16} /> {loginBusyId === selectedEndpoint.id ? t("endpoint.connecting") : selectedProfiles.length > 0 ? t("endpoint.addAccount") : t("endpoint.login")}
              </button>
              {!selectedEndpoint.builtIn && <IconButton label={t("endpoint.editNamed", { name: selectedEndpoint.name })} icon={Pencil} disabled={loginBusyId !== null} onClick={() => setEndpointDialog(selectedEndpoint)} />}
              {!selectedEndpoint.builtIn && <IconButton label={t("endpoint.deleteNamed", { name: selectedEndpoint.name })} icon={Trash2} disabled={loginBusyId !== null} onClick={() => setDeleteEndpoint(selectedEndpoint)} />}
            </div>
          </header>

          {loginFailure?.endpointId === selectedEndpoint.id && (
            <div className="login-flow login-failure" role="alert">
              <AlertTriangle aria-hidden="true" size={21} />
              <div>
                <strong>{t("endpoint.cannotConnect", { name: selectedEndpoint.name })}</strong>
                <span>{loginFailure.message}</span>
              </div>
            </div>
          )}

          {loginState?.endpointId === selectedEndpoint.id && (
            <div className="login-flow" role="status">
              <ShieldCheck aria-hidden="true" size={21} />
              <div className="truncate">
                <strong>{t("endpoint.loginReady", { name: selectedEndpoint.name })}</strong>
                <span>{t("endpoint.loginInstructions")}</span>
                <span className="truncate" title={loginState.authUrl}>{loginState.authUrl}</span>
              </div>
              <button className="button secondary with-icon" type="button" onClick={() => void onOpenURL(loginState.authUrl)}>
                {t("endpoint.openBrowser")} <ExternalLink aria-hidden="true" size={16} />
              </button>
            </div>
          )}

          <section className="endpoint-accounts" aria-labelledby="profiles-title">
            <header className="section-header">
              <div>
                <h2 id="profiles-title">{t("profile.accounts")}</h2>
                <p>{t("profile.scopeHint")}</p>
              </div>
              <StatusBadge tone="neutral">{t("endpoint.accountCount", { count: selectedProfiles.length })}</StatusBadge>
            </header>
            {selectedProfiles.length === 0 ? (
              <EmptyState title={t("profile.none")} message={t("profile.noneHint")} action={<button className="button secondary with-icon" type="button" onClick={() => void login(selectedEndpoint.id)}><LogIn aria-hidden="true" size={16} /> {t("endpoint.login")}</button>} />
            ) : (
              <div className="profile-list">
                {selectedProfiles.map((profile) => {
                  const active = profile.id === activeProfileId;
                  const displayName = profileDisplayName(profile, t);
                  return (
                    <div className="profile-row" key={profile.id}>
                      <span className="profile-avatar">{displayName.slice(0, 1).toUpperCase()}</span>
                      <div className="profile-main truncate">
                        <div>
                          <strong className="truncate" title={displayName}>{displayName}</strong>
                          {active && <Check aria-label={t("account.current")} size={17} />}
                        </div>
                        <span className="truncate" title={profile.account}>{profile.account || t("profile.loginNameMissing")}</span>
                      </div>
                      <StatusBadge tone={profile.state === "ready" ? (active ? "success" : "neutral") : "warning"}>{profileStatusLabel(profile, active, t)}</StatusBadge>
                      {active ? (
                        <button className="button secondary profile-action logout-action with-icon" type="button" disabled={loginBusyId !== null} onClick={() => setLogoutProfile(profile)}>
                          <LogOut aria-hidden="true" size={15} /> {t("profile.logout")}
                        </button>
                      ) : (
                        <button className="button secondary profile-action" type="button" disabled={switchingId !== null || loginBusyId !== null} onClick={() => void switchProfile(profile.id)}>
                          {switchingId === profile.id ? t("profile.switching") : t("profile.switch")}
                        </button>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </section>
        </section>
      )}

      {endpointDialog !== undefined && <EndpointDialog endpoint={endpointDialog} onClose={() => setEndpointDialog(undefined)} onSave={onSaveEndpoint} />}
      {deleteEndpoint && <DeleteEndpointDialog endpoint={deleteEndpoint} onClose={() => setDeleteEndpoint(null)} onDelete={onDeleteEndpoint} />}
      {logoutProfile && (
        <LogoutDialog
          profile={logoutProfile}
          endpointName={endpoints.find((endpoint) => endpoint.id === logoutProfile.endpointId)?.name ?? t("profile.currentEndpoint")}
          onClose={() => setLogoutProfile(null)}
          onLogout={onLogout}
        />
      )}
    </div>
  );
}
