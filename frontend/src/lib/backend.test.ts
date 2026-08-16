import { afterEach, describe, expect, it, vi } from "vitest";
import type { BackendBindings } from "./contracts";
import { createBackend, createDemoSnapshot, openExternalURL } from "./backend";

afterEach(() => {
  vi.restoreAllMocks();
});

function createBindings(getSnapshot: BackendBindings["GetSnapshot"]): BackendBindings {
  const snapshot = createDemoSnapshot();
  return {
    GetSnapshot: getSnapshot,
    EnsureDaemon: async () => snapshot,
    SetConnection: async () => snapshot,
    SetPreference: async () => snapshot,
    SetExitNode: async () => snapshot,
    PingDevice: async (deviceId) => ({ deviceId, latencyMs: 1, via: "direct" }),
    SaveEndpoint: async () => snapshot,
    DeleteEndpoint: async () => snapshot,
    SwitchProfile: async () => snapshot,
    Logout: async () => snapshot,
    BeginLogin: async (endpointId) => ({ endpointId, authUrl: "https://login.example.com" }),
    SetAppSetting: async () => snapshot,
    SetTheme: async () => snapshot,
    SetLanguage: async () => snapshot,
  };
}

describe("native backend errors", () => {
  it("preserves binding failures instead of replacing them with demo data", async () => {
    const getSnapshot = vi.fn(async () => {
      throw { message: "native LocalAPI denied" };
    });
    const client = createBackend(createBindings(getSnapshot));

    await expect(client.getSnapshot()).rejects.toThrow("native LocalAPI denied");
    await expect(client.getSnapshot()).rejects.toThrow("native LocalAPI denied");
    expect(getSnapshot).toHaveBeenCalledTimes(2);
  });
});

describe("demo account lifecycle", () => {
  it("removes only the active profile when logging out", async () => {
    const client = createBackend();
    const before = await client.getSnapshot();

    const after = await client.logout();

    expect(after.profiles).toHaveLength(before.profiles.length - 1);
    expect(after.profiles.some((profile) => profile.id === before.activeProfileId)).toBe(false);
    expect(after.activeProfileId).toBeNull();
    expect(after.runtime.session).toBe("login-required");
  });
});

describe("openExternalURL", () => {
  it("rejects an unsafe URL before opening a browser", async () => {
    const open = vi.spyOn(window, "open").mockImplementation(() => null);

    await expect(openExternalURL("https://user:password@example.com/login")).rejects.toThrow();
    expect(open).not.toHaveBeenCalled();
  });

  it("opens the normalized form of a safe URL", async () => {
    const open = vi.spyOn(window, "open").mockImplementation(() => null);

    await openExternalURL("https://login.example.com");

    expect(open).toHaveBeenCalledWith(
      "https://login.example.com/",
      "_blank",
      "noopener,noreferrer",
    );
  });
});
