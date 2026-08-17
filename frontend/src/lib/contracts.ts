export type DaemonState =
  | "unknown"
  | "missing"
  | "stopped"
  | "ready"
  | "unauthorized"
  | "incompatible";

export type SessionState =
  | "none"
  | "login-required"
  | "approval-required"
  | "authenticated";

export type ConnectionState =
  | "stopped"
  | "starting"
  | "running"
  | "stopping"
  | "degraded";

export type ControlState = "unknown" | "reachable" | "unreachable";
export type EndpointKind = "headscale" | "tailscale" | "compatible";
export type ThemePreference = "system" | "light" | "dark";
export type LanguagePreference = "zh-CN" | "en-US";
export type PreferenceKey =
  | "acceptDns"
  | "acceptRoutes"
  | "allowLanAccess"
  | "shieldsUp";
export type AppSettingKey =
  | "launchAtLogin"
  | "closeToTray"
  | "notifications"
  | "autoUpdate";

export interface RuntimeState {
  daemon: DaemonState;
  session: SessionState;
  connection: ConnectionState;
  control: ControlState;
}

export interface HealthNotice {
  code: "routes-not-accepted" | "tailscale-warning";
  severity: "info" | "warning";
  message: string;
}

export interface LocalDevice {
  id: string;
  name: string;
  dnsName: string;
  os: string;
  addresses: string[];
  clientVersion: string;
  connectionType: "direct" | "relay" | "offline" | "unknown";
  relayRegion?: string;
}

export interface PeerDevice {
  id: string;
  name: string;
  dnsName: string;
  owner: string;
  os: string;
  addresses: string[];
  online: boolean;
  lastSeen: string;
  latencyMs?: number;
  connectionType: "direct" | "relay" | "offline" | "unknown";
  relayRegion?: string;
  exitNodeOption: boolean;
  tags: string[];
}

export interface Endpoint {
  id: string;
  name: string;
  url: string;
  kind: EndpointKind;
  status: "reachable" | "unreachable" | "unchecked";
  customCa: boolean;
  builtIn: boolean;
}

export interface LoginProfile {
  id: string;
  endpointId: string;
  account: string;
  displayName: string;
  active: boolean;
  state: "ready" | "login-required" | "approval-required";
  lastUsedAt: string;
}

export interface QuickPreferences {
  exitNodeId: string | null;
  acceptDns: boolean;
  acceptRoutes: boolean;
  allowLanAccess: boolean;
  shieldsUp: boolean;
}

export interface AppSettings {
  launchAtLogin: boolean;
  closeToTray: boolean;
  notifications: boolean;
  autoUpdate: boolean;
  theme: ThemePreference;
  language: LanguagePreference;
}

export interface Diagnostics {
  appVersion: string;
  wailsVersion: string;
  daemonVersion: string;
  localApi: string;
  platform: string;
}

export interface EngineStatus {
  ownership: "unknown" | "managed" | "external" | "prepared" | "missing";
  service: "unknown" | "missing" | "stopped" | "starting" | "running" | "stopping";
  bundledVersion: string;
  payloadAvailable: boolean;
  canInstall: boolean;
  canStart: boolean;
}

export interface AppSnapshot {
  source: "native" | "demo";
  fallbackReason?: string;
  runtime: RuntimeState;
  healthNotices: HealthNotice[];
  localDevice: LocalDevice;
  devices: PeerDevice[];
  endpoints: Endpoint[];
  profiles: LoginProfile[];
  activeEndpointId: string | null;
  activeProfileId: string | null;
  preferences: QuickPreferences;
  settings: AppSettings;
  diagnostics: Diagnostics;
  engine: EngineStatus;
  updatedAt: string;
}

export interface EndpointInput {
  id?: string;
  name: string;
  url: string;
  kind: EndpointKind;
  customCa: boolean;
}

export interface PingResult {
  deviceId: string;
  latencyMs: number;
  via: "direct" | "relay" | "unknown";
  relayRegion?: string;
  endpoint?: string;
}

export interface LoginResult {
  endpointId: string;
  authUrl: string;
}

export type NavigationTarget = "overview" | "devices" | "networks" | "settings";

export interface HeadscaleBackend {
  getSnapshot(): Promise<AppSnapshot>;
  ensureDaemon(): Promise<AppSnapshot>;
  setConnection(enabled: boolean): Promise<AppSnapshot>;
  setPreference(key: PreferenceKey, value: boolean): Promise<AppSnapshot>;
  setExitNode(deviceId: string | null): Promise<AppSnapshot>;
  pingDevice(deviceId: string): Promise<PingResult>;
  saveEndpoint(input: EndpointInput): Promise<AppSnapshot>;
  deleteEndpoint(endpointId: string): Promise<AppSnapshot>;
  switchProfile(profileId: string): Promise<AppSnapshot>;
  logout(): Promise<AppSnapshot>;
  beginLogin(endpointId: string): Promise<LoginResult>;
  setAppSetting(key: AppSettingKey, value: boolean): Promise<AppSnapshot>;
  setTheme(theme: ThemePreference): Promise<AppSnapshot>;
  setLanguage(language: LanguagePreference): Promise<AppSnapshot>;
  subscribe(
    onSnapshot: (snapshot: AppSnapshot) => void,
    onError: (message: string) => void,
  ): () => void;
  subscribeNavigation(onNavigate: (target: NavigationTarget) => void): () => void;
}

export interface BackendBindings {
  GetSnapshot(): Promise<AppSnapshot>;
  EnsureDaemon(): Promise<AppSnapshot>;
  SetConnection(enabled: boolean): Promise<AppSnapshot>;
  SetPreference(key: PreferenceKey, value: boolean): Promise<AppSnapshot>;
  SetExitNode(deviceId: string | null): Promise<AppSnapshot>;
  PingDevice(deviceId: string): Promise<PingResult>;
  SaveEndpoint(input: EndpointInput): Promise<AppSnapshot>;
  DeleteEndpoint(endpointId: string): Promise<AppSnapshot>;
  SwitchProfile(profileId: string): Promise<AppSnapshot>;
  Logout(): Promise<AppSnapshot>;
  BeginLogin(endpointId: string): Promise<LoginResult>;
  SetAppSetting(key: AppSettingKey, value: boolean): Promise<AppSnapshot>;
  SetTheme(theme: ThemePreference): Promise<AppSnapshot>;
  SetLanguage(language: LanguagePreference): Promise<AppSnapshot>;
}
