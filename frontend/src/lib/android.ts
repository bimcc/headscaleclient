import type { AppSnapshot, BackendBindings, HeadscaleBackend } from "./contracts";
import { androidErrorMessage } from "./androidError";

interface AndroidHost {
  postMessage(message: string): void;
  onmessage?: (event: { data: string }) => void;
}
declare global { interface Window { HeadscaleAndroid?: AndroidHost } }

const pending = new Map<string, { resolve: (value: unknown) => void; reject: (error: Error) => void; timer: ReturnType<typeof setTimeout> }>();
let sequence = 0;
let installedHost: AndroidHost | undefined;

export function androidCall<T>(method: string, args: unknown[] = []): Promise<T> {
  const host = window.HeadscaleAndroid;
  if (!host) return Promise.reject(new Error("Android bridge unavailable"));
  if (installedHost !== host) {
    installedHost = host;
    host.onmessage = (event) => {
      try {
        const { id, response } = JSON.parse(event.data);
        const request = pending.get(id); if (!request) return;
        clearTimeout(request.timer); pending.delete(id);
        if (response.error) request.reject(new Error(androidErrorMessage(response.problem, response.error))); else request.resolve(response.result);
      } catch { /* A malformed response cannot complete another operation. */ }
    };
  }
  if (pending.size >= 8) return Promise.reject(new Error("Too many pending Android operations"));
  return new Promise<T>((resolve, reject) => {
    const id = `${++sequence}`;
    const timer = setTimeout(() => { pending.delete(id); reject(new Error(document.documentElement.lang.startsWith("zh") ? "操作超时，请检查 VPN 授权和网络后重试。" : "Operation timed out. Check VPN permission and connectivity.")); }, 90000);
    pending.set(id, { resolve: resolve as (value: unknown) => void, reject, timer });
    try { host.postMessage(JSON.stringify({ id, method, args })); }
    catch (error) { clearTimeout(timer); pending.delete(id); reject(error); }
  });
}

export function androidBindings(): BackendBindings {
  return {
    GetSnapshot: () => androidCall("GetSnapshot"), EnsureDaemon: () => androidCall("EnsureDaemon"),
    SetConnection: (value) => androidCall("SetConnection", [value]),
    SetPreference: (key, value) => androidCall("SetPreference", [key, value]),
    SetExitNode: (id) => androidCall("SetExitNode", [id]), PingDevice: (id) => androidCall("PingDevice", [id]),
    SaveEndpoint: (input) => androidCall("SaveEndpoint", [input]), DeleteEndpoint: (id) => androidCall("DeleteEndpoint", [id]),
    SwitchProfile: (id) => androidCall("SwitchProfile", [id]), Logout: () => androidCall("Logout"),
    BeginLogin: (id) => androidCall("BeginLogin", [id]), SetAppSetting: (key, value) => androidCall("SetAppSetting", [key, value]),
    SetTheme: (theme) => androidCall("SetTheme", [theme]), SetLanguage: (language) => androidCall("SetLanguage", [language]),
  };
}

export function androidSubscribe(getSnapshot: () => Promise<AppSnapshot>): HeadscaleBackend["subscribe"] {
  return (onSnapshot, onError) => {
    let active = true;
    let lastSequence = 0;
    const refresh = () => { void getSnapshot().then((s) => { if (active) onSnapshot(s); }).catch((e: Error) => { if (active) onError(e.message); }); };
    const listener = (event: Event) => {
      const { name, payload } = (event as CustomEvent).detail;
      if (name === "android:vpn-state-changed") { refresh(); return; }
      if (typeof payload.sequence === "number") {
        if (payload.sequence <= lastSequence) return;
        const gap = lastSequence !== 0 && payload.sequence > lastSequence + 1;
        lastSequence = payload.sequence;
        if (gap) { refresh(); return; }
      }
      if ((name === "app:snapshot-changed" || name === "app:login-finished") && payload.snapshot) onSnapshot(payload.snapshot);
      if (name === "app:operation-failed") onError(androidErrorMessage(payload.problem));
    };
    window.addEventListener("headscale:native", listener);
    window.addEventListener("headscale:resume", refresh);
    return () => { active = false; window.removeEventListener("headscale:native", listener); window.removeEventListener("headscale:resume", refresh); };
  };
}
