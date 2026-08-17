import { Browser, Events } from "@wailsio/runtime";
import * as generatedBindings from "../../bindings/github.com/headscaleclient/headscaleclient/internal/desktop/backend";
import type {
  AppSettingKey,
  AppSnapshot,
  BackendBindings,
  EndpointInput,
  HeadscaleBackend,
  LanguagePreference,
  LoginResult,
  NavigationTarget,
  PingResult,
  PreferenceKey,
  ThemePreference,
} from "./contracts";
import { validateLoginURL } from "./externalUrl";

type RuntimeWindow = Window & {
  chrome?: { webview?: { postMessage?: unknown } };
  webkit?: { messageHandlers?: { external?: { postMessage?: unknown } } };
  wails?: { invoke?: unknown };
};

const runtimeWindow = window as RuntimeWindow;
const nativeRuntime = Boolean(
  runtimeWindow.chrome?.webview?.postMessage
  || runtimeWindow.webkit?.messageHandlers?.external?.postMessage
  || runtimeWindow.wails?.invoke,
);

const demoSnapshot: AppSnapshot = {
  source: "demo",
  fallbackReason: "未检测到本地服务，当前显示最近一次演示快照。",
  runtime: {
    daemon: "stopped",
    session: "authenticated",
    connection: "stopped",
    control: "reachable",
  },
  healthNotices: [],
  localDevice: {
    id: "local-mbp",
    name: "studio-mbp",
    dnsName: "studio-mbp.mesh.example",
    os: "macOS 15.5",
    addresses: ["100.92.14.7", "fd7a:115c:a1e0::701"],
    clientVersion: "1.84.3",
    connectionType: "offline",
  },
  devices: [
    {
      id: "peer-nas",
      name: "home-nas",
      dnsName: "home-nas.mesh.example",
      owner: "lin@example.com",
      os: "Linux",
      addresses: ["100.92.14.2"],
      online: true,
      lastSeen: "刚刚",
      latencyMs: 8,
      connectionType: "direct",
      exitNodeOption: true,
      tags: ["tag:server"],
    },
    {
      id: "peer-workstation",
      name: "workstation",
      dnsName: "workstation.mesh.example",
      owner: "lin@example.com",
      os: "Windows 11",
      addresses: ["100.92.14.11"],
      online: true,
      lastSeen: "1 分钟前",
      latencyMs: 31,
      connectionType: "relay",
      relayRegion: "Hong Kong",
      exitNodeOption: false,
      tags: [],
    },
    {
      id: "peer-phone",
      name: "pixel-9",
      dnsName: "pixel-9.mesh.example",
      owner: "lin@example.com",
      os: "Android",
      addresses: ["100.92.14.18"],
      online: false,
      lastSeen: "昨天 23:42",
      connectionType: "offline",
      exitNodeOption: false,
      tags: [],
    },
    {
      id: "peer-ci",
      name: "build-runner",
      dnsName: "build-runner.mesh.example",
      owner: "ops@example.com",
      os: "Linux",
      addresses: ["100.92.14.25"],
      online: true,
      lastSeen: "刚刚",
      latencyMs: 67,
      connectionType: "relay",
      relayRegion: "Tokyo",
      exitNodeOption: false,
      tags: ["tag:ci"],
    },
  ],
  endpoints: [
    {
      id: "endpoint-primary",
      name: "团队 Headscale",
      url: "https://hs.example.com",
      kind: "headscale",
      status: "reachable",
      customCa: false,
      builtIn: false,
    },
    {
      id: "endpoint-tailscale",
      name: "Tailscale 官方",
      url: "https://login.tailscale.com",
      kind: "tailscale",
      status: "unchecked",
      customCa: false,
      builtIn: true,
    },
  ],
  profiles: [
    {
      id: "profile-team",
      endpointId: "endpoint-primary",
      account: "lin@example.com",
      displayName: "团队网络",
      active: true,
      state: "ready",
      lastUsedAt: "刚刚",
    },
    {
      id: "profile-personal",
      endpointId: "endpoint-tailscale",
      account: "lin.personal@example.com",
      displayName: "个人 Tailnet",
      active: false,
      state: "login-required",
      lastUsedAt: "2026-08-12",
    },
  ],
  activeEndpointId: "endpoint-primary",
  activeProfileId: "profile-team",
  preferences: {
    exitNodeId: null,
    acceptDns: true,
    acceptRoutes: true,
    allowLanAccess: false,
    shieldsUp: false,
  },
  settings: {
    launchAtLogin: true,
    closeToTray: true,
    notifications: true,
    autoUpdate: true,
    theme: "system",
    language: "zh-CN",
  },
  diagnostics: {
    appVersion: "0.1.0-dev",
    wailsVersion: "3.0.0-beta.8",
    daemonVersion: "未连接",
    localApi: "不可用",
    platform: "Desktop",
  },
  engine: {
    ownership: "prepared",
    service: "missing",
    bundledVersion: "1.102.2",
    payloadAvailable: true,
    canInstall: true,
    canStart: false,
  },
  updatedAt: "2026-08-15T10:30:00+08:00",
};

export function createDemoSnapshot(): AppSnapshot {
  return JSON.parse(JSON.stringify(demoSnapshot)) as AppSnapshot;
}

const pause = (duration = 90) =>
  new Promise<void>((resolve) => window.setTimeout(resolve, duration));

class DemoBackend implements HeadscaleBackend {
  private snapshot = createDemoSnapshot();

  constructor(fallbackReason?: string) {
    if (fallbackReason) {
      this.snapshot.fallbackReason = fallbackReason;
    }
  }

  async getSnapshot() {
    await pause(120);
    return this.copy();
  }

  async ensureDaemon() {
    await pause(180);
    this.snapshot.engine.ownership = "managed";
    this.snapshot.engine.service = "running";
    this.snapshot.engine.canInstall = false;
    this.snapshot.engine.canStart = false;
    this.snapshot.runtime.daemon = "ready";
    this.snapshot.runtime.session = "login-required";
    this.snapshot.fallbackReason = undefined;
    this.snapshot.diagnostics.localApi = "tailscaled LocalAPI";
    this.touch();
    return this.copy();
  }

  async setConnection(enabled: boolean) {
    await pause();
    this.snapshot.runtime.connection = enabled ? "running" : "stopped";
    this.snapshot.runtime.daemon = enabled ? "ready" : "stopped";
    this.snapshot.localDevice.connectionType = enabled ? "direct" : "offline";
    this.touch();
    return this.copy();
  }

  async setPreference(key: PreferenceKey, value: boolean) {
    await pause();
    this.snapshot.preferences[key] = value;
    this.touch();
    return this.copy();
  }

  async setExitNode(deviceId: string | null) {
    await pause();
    this.snapshot.preferences.exitNodeId = deviceId;
    this.snapshot.preferences.allowLanAccess = deviceId !== null;
    this.touch();
    return this.copy();
  }

  async pingDevice(deviceId: string): Promise<PingResult> {
    await pause(180);
    const device = this.snapshot.devices.find((item) => item.id === deviceId);
    if (!device || !device.online) {
      throw new Error("设备当前离线，无法 Ping。 ");
    }
    return {
      deviceId,
      latencyMs: device.latencyMs ?? 42,
      via: device.connectionType === "direct" ? "direct" : device.connectionType === "relay" ? "relay" : "unknown",
      relayRegion: device.relayRegion,
    };
  }

  async saveEndpoint(input: EndpointInput) {
    await pause();
    const existing = input.id
      ? this.snapshot.endpoints.find((endpoint) => endpoint.id === input.id)
      : undefined;
    if (input.id && !existing) throw new Error("找不到所选控制服务器。");
    if (existing) {
      if (existing.builtIn) throw new Error("内置控制服务器不能编辑。");
      existing.name = input.name;
      existing.url = input.url;
      existing.kind = input.kind;
      existing.customCa = input.customCa;
      existing.status = "unchecked";
      this.touch();
      return this.copy();
    }
    const endpointId = `endpoint-${Date.now()}`;
    this.snapshot.endpoints.push({
      id: endpointId,
      name: input.name,
      url: input.url,
      kind: input.kind,
      customCa: input.customCa,
      status: "unchecked",
      builtIn: false,
    });
    this.touch();
    return this.copy();
  }

  async deleteEndpoint(endpointId: string) {
    await pause();
    const index = this.snapshot.endpoints.findIndex((endpoint) => endpoint.id === endpointId);
    if (index < 0) throw new Error("找不到所选控制服务器。");
    if (this.snapshot.endpoints[index].builtIn) throw new Error("内置控制服务器不能删除。");
    this.snapshot.endpoints.splice(index, 1);
    this.touch();
    return this.copy();
  }

  async switchProfile(profileId: string) {
    await pause();
    const selected = this.snapshot.profiles.find((profile) => profile.id === profileId);
    if (!selected) {
      throw new Error("找不到所选账号。 ");
    }
    this.snapshot.profiles.forEach((profile) => {
      profile.active = profile.id === profileId;
    });
    this.snapshot.activeProfileId = profileId;
    this.snapshot.activeEndpointId = selected.endpointId;
    this.snapshot.runtime.session =
      selected.state === "ready" ? "authenticated" : selected.state;
    this.snapshot.runtime.connection = "stopped";
    this.touch();
    return this.copy();
  }

  async logout() {
    await pause();
    const activeProfileId = this.snapshot.activeProfileId;
    if (!activeProfileId) {
      throw new Error("当前没有已登录账号。");
    }
    this.snapshot.profiles = this.snapshot.profiles.filter(
      (profile) => profile.id !== activeProfileId,
    );
    this.snapshot.activeProfileId = null;
    this.snapshot.activeEndpointId = null;
    this.snapshot.runtime.session = "login-required";
    this.snapshot.runtime.connection = "stopped";
    this.snapshot.runtime.control = "unknown";
    this.snapshot.localDevice.connectionType = "offline";
    this.touch();
    return this.copy();
  }

  async beginLogin(endpointId: string): Promise<LoginResult> {
    await pause();
    const endpoint = this.snapshot.endpoints.find((item) => item.id === endpointId);
    if (!endpoint) {
      throw new Error("找不到所选控制服务器。 ");
    }
    return {
      endpointId,
      authUrl: `${endpoint.url}/register/demo-device`,
    };
  }

  async setAppSetting(key: AppSettingKey, value: boolean) {
    await pause();
    this.snapshot.settings[key] = value;
    this.touch();
    return this.copy();
  }

  async setTheme(theme: ThemePreference) {
    await pause();
    this.snapshot.settings.theme = theme;
    this.touch();
    return this.copy();
  }

  async setLanguage(language: LanguagePreference) {
    await pause();
    this.snapshot.settings.language = language;
    this.touch();
    return this.copy();
  }

  subscribe() {
    return () => undefined;
  }

  subscribeNavigation() {
    return () => undefined;
  }

  private touch() {
    this.snapshot.updatedAt = new Date().toISOString();
  }

  private copy() {
    return JSON.parse(JSON.stringify(this.snapshot)) as AppSnapshot;
  }
}

class NativeBackend implements HeadscaleBackend {
  constructor(private readonly bindings: BackendBindings) {}

  getSnapshot = () => this.invoke(this.bindings.GetSnapshot());
  ensureDaemon = () => this.invoke(this.bindings.EnsureDaemon());
  setConnection = (enabled: boolean) => this.invoke(this.bindings.SetConnection(enabled));
  setPreference = (key: PreferenceKey, value: boolean) =>
    this.invoke(this.bindings.SetPreference(key, value));
  setExitNode = (deviceId: string | null) => this.invoke(this.bindings.SetExitNode(deviceId));
  pingDevice = (deviceId: string) => this.invoke(this.bindings.PingDevice(deviceId));
  saveEndpoint = (input: EndpointInput) => this.invoke(this.bindings.SaveEndpoint(input));
  deleteEndpoint = (endpointId: string) => this.invoke(this.bindings.DeleteEndpoint(endpointId));
  switchProfile = (profileId: string) => this.invoke(this.bindings.SwitchProfile(profileId));
  logout = () => this.invoke(this.bindings.Logout());
  beginLogin = (endpointId: string) => this.invoke(this.bindings.BeginLogin(endpointId));
  setAppSetting = (key: AppSettingKey, value: boolean) =>
    this.invoke(this.bindings.SetAppSetting(key, value));
  setTheme = (theme: ThemePreference) => this.invoke(this.bindings.SetTheme(theme));
  setLanguage = (language: LanguagePreference) => this.invoke(this.bindings.SetLanguage(language));

  subscribe(onSnapshot: (snapshot: AppSnapshot) => void, onError: (message: string) => void) {
    let lastSequence = 0;
    const accept = (sequence: number, snapshot?: AppSnapshot) => {
      const gap = lastSequence !== 0 && sequence > lastSequence + 1;
      if (sequence > lastSequence) lastSequence = sequence;
      if (gap) {
        void this.getSnapshot().then(onSnapshot).catch((error) => onError(toError(error).message));
      } else if (snapshot) {
        onSnapshot(snapshot);
      }
    };

    const offSnapshot = Events.On("app:snapshot-changed", (event) => {
      const payload = event.data as { sequence: number; snapshot: AppSnapshot };
      accept(payload.sequence, payload.snapshot);
    });
    const offLoginURL = Events.On("app:login-url", (event) => {
      accept((event.data as { sequence: number }).sequence);
    });
    const offLoginFinished = Events.On("app:login-finished", (event) => {
      const payload = event.data as { sequence: number; snapshot: AppSnapshot };
      accept(payload.sequence, payload.snapshot);
    });
    const offFailure = Events.On("app:operation-failed", (event) => {
      const payload = event.data as { sequence: number; problem: { message: string } };
      accept(payload.sequence);
      onError(payload.problem.message);
    });

    return () => {
      offSnapshot();
      offLoginURL();
      offLoginFinished();
      offFailure();
    };
  }

  subscribeNavigation(onNavigate: (target: NavigationTarget) => void) {
    return Events.On("app:navigate", (event) => {
      const target = (event.data as { view?: unknown }).view;
      if (target === "overview" || target === "devices" || target === "networks" || target === "settings") {
        onNavigate(target);
      }
    });
  }

  private invoke<T>(call: Promise<T>): Promise<T> {
    return call.catch((error) => Promise.reject(toError(error)));
  }
}

export function createBackend(bindings?: BackendBindings): HeadscaleBackend {
  return bindings ? new NativeBackend(bindings) : new DemoBackend();
}

function toError(value: unknown): Error {
  if (value instanceof Error) return value;
  if (typeof value === "object" && value !== null && "message" in value) {
    return new Error(String(value.message));
  }
  if (typeof value === "string") {
    try {
      const parsed = JSON.parse(value) as { message?: unknown };
      if (parsed.message) return new Error(String(parsed.message));
    } catch {
      // Keep the original string when it is not structured JSON.
    }
    return new Error(value);
  }
  return new Error("本地服务响应异常");
}

export async function openExternalURL(url: string) {
  const safeURL = validateLoginURL(url);
  if (nativeRuntime) {
    await Browser.OpenURL(safeURL);
    return;
  }
  window.open(safeURL, "_blank", "noopener,noreferrer");
}

const bindings = nativeRuntime
  ? (generatedBindings as unknown as BackendBindings)
  : undefined;

export const backend = createBackend(bindings);
