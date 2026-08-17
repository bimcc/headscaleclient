import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { App } from "./App";
import { createBackend, createDemoSnapshot } from "./lib/backend";

describe("HeadscaleClient shell", () => {
  it("loads the fallback snapshot and filters the device view", async () => {
    const user = userEvent.setup();
    render(<App backendClient={createBackend()} />);

    expect(screen.getByLabelText("正在加载")).toBeInTheDocument();
    expect(await screen.findByRole("heading", { name: "服务不可用" })).toBeInTheDocument();
    expect(screen.getByRole("navigation", { name: "主导航" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "设备" }));
    expect(screen.getByRole("heading", { name: "设备列表" })).toBeInTheDocument();

    await user.type(screen.getByRole("searchbox", { name: "搜索设备" }), "pixel");
    expect(screen.getByText("pixel-9")).toBeInTheDocument();
    expect(screen.queryByText("home-nas")).not.toBeInTheDocument();
  });

  it("renders a recoverable error state when initial loading fails", async () => {
    const brokenBackend = createBackend();
    vi.spyOn(brokenBackend, "getSnapshot").mockRejectedValueOnce(new Error("LocalAPI denied"));

    render(<App backendClient={brokenBackend} />);

    expect(await screen.findByRole("heading", { name: "无法加载客户端状态" })).toBeInTheDocument();
    expect(screen.getByText("LocalAPI denied")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "重试" })).toBeEnabled();
  });

  it("installs the prepared network service from the offline banner", async () => {
    const user = userEvent.setup();
    render(<App backendClient={createBackend()} />);

    expect(await screen.findByRole("button", { name: "安装网络服务" })).toBeEnabled();
    await user.click(screen.getByRole("button", { name: "安装网络服务" }));

    expect(await screen.findByRole("heading", { name: "需要登录" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "安装网络服务" })).not.toBeInTheDocument();
  });

  it("allows close-to-tray behavior to be changed", async () => {
    const user = userEvent.setup();
    render(<App backendClient={createBackend()} />);

    await screen.findByRole("heading", { name: "服务不可用" });
    await user.click(screen.getByRole("button", { name: "设置" }));
    const toggle = screen.getByRole("switch", { name: "关闭到托盘" });
    expect(toggle).toHaveAttribute("aria-checked", "true");

    await user.click(toggle);
    await waitFor(() => expect(toggle).toHaveAttribute("aria-checked", "false"));
  });

  it("groups settings by purpose and keeps product attribution on About", async () => {
    const user = userEvent.setup();
    render(<App backendClient={createBackend()} />);

    await screen.findByRole("heading", { name: "服务不可用" });
    await user.click(screen.getByRole("button", { name: "设置" }));

    expect(screen.getByRole("heading", { name: "常规设置" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "运行与诊断" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "启动并修复" })).toBeEnabled();
    expect(screen.queryByRole("heading", { name: "应用" })).not.toBeInTheDocument();
    expect(screen.queryByText("Powered by BIMCC., Ltd.")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "关于" }));
    expect(screen.getByText("BIMCC., Ltd.")).toBeInTheDocument();
    expect(screen.getByText(/Tailscale Inc\. 及贡献者/)).toBeInTheDocument();
    expect(screen.getByText(/Headscale 开源项目版权归 Juan Font/)).toBeInTheDocument();
  });

  it("defaults to Chinese and switches the application to English", async () => {
    const user = userEvent.setup();
    const backend = createBackend();
    const setLanguage = vi.spyOn(backend, "setLanguage");
    render(<App backendClient={backend} />);

    await screen.findByRole("heading", { name: "服务不可用" });
    expect(screen.getByRole("navigation", { name: "主导航" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "设置" }));
    await user.selectOptions(screen.getByRole("combobox", { name: "语言" }), "en-US");

    await waitFor(() => expect(setLanguage).toHaveBeenCalledWith("en-US"));
    expect(await screen.findByRole("heading", { name: "Settings", level: 1 })).toBeInTheDocument();
    expect(screen.getByRole("navigation", { name: "Main navigation" })).toBeInTheDocument();
    expect(screen.queryByText("Powered by BIMCC., Ltd.")).not.toBeInTheDocument();
    expect(document.documentElement.lang).toBe("en-US");

    await user.click(screen.getByRole("button", { name: "About" }));
    expect(screen.getByRole("heading", { name: "HeadscaleClient" })).toBeInTheDocument();
    expect(screen.getByText("Version 0.1.0-dev")).toBeInTheDocument();
    expect(screen.getByText("BIMCC., Ltd.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Open the official Tailscale website at tailscale.com" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Open the official Headscale website at headscale.net" })).toBeInTheDocument();
  });

  it("opens the device list from the compact online-device count", async () => {
    const user = userEvent.setup();
    render(<App backendClient={createBackend()} />);

    await screen.findByRole("heading", { name: "服务不可用" });
    expect(screen.queryByRole("heading", { name: "在线设备" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "查看在线设备，3 台在线" }));
    expect(screen.getByRole("heading", { name: "设备列表" })).toBeInTheDocument();
  });

  it("enables LAN access only after an exit node is selected", async () => {
    const user = userEvent.setup();
    render(<App backendClient={createBackend()} />);

    await screen.findByRole("heading", { name: "服务不可用" });
    const allowLan = screen.getByRole("switch", { name: "允许局域网访问" });
    expect(allowLan).toBeDisabled();
    expect(allowLan.closest(".setting-row")).toHaveClass("is-nested", "is-disabled");
    expect(screen.getByText("选择出口节点后可用")).toBeInTheDocument();

    await user.selectOptions(screen.getByRole("combobox", { name: "出口节点" }), "peer-nas");
    await waitFor(() => expect(allowLan).toBeEnabled());
    expect(allowLan).toHaveAttribute("aria-checked", "true");
    expect(allowLan.closest(".setting-row")).not.toHaveClass("is-disabled");
    expect(screen.getByText("使用出口节点时仍可访问本地网络")).toBeInTheDocument();
  });

  it("warns when an existing exit-node profile blocks LAN access", async () => {
    const user = userEvent.setup();
    const backend = createBackend();
    const snapshot = createDemoSnapshot();
    snapshot.preferences.exitNodeId = "peer-nas";
    snapshot.preferences.allowLanAccess = false;
    vi.spyOn(backend, "getSnapshot").mockResolvedValue(snapshot);
    const setPreference = vi.spyOn(backend, "setPreference").mockImplementation(async (key, value) => {
      snapshot.preferences[key] = value;
      return snapshot;
    });

    render(<App backendClient={backend} />);

    expect(await screen.findByText("出口节点正在阻止局域网访问")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "允许局域网访问" }));
    await waitFor(() => expect(setPreference).toHaveBeenCalledWith("allowLanAccess", true));
  });

  it("switches saved profiles from the header account menu", async () => {
    const user = userEvent.setup();
    const backend = createBackend();
    const switchProfile = vi.spyOn(backend, "switchProfile");
    render(<App backendClient={backend} />);

    const accountButton = await screen.findByRole("button", { name: "切换账号，当前 lin@example.com" });
    await user.click(accountButton);
    expect(screen.getByRole("menu", { name: "账号切换" })).toBeInTheDocument();

    await user.keyboard("{Escape}");
    expect(screen.queryByRole("menu", { name: "账号切换" })).not.toBeInTheDocument();

    await user.click(accountButton);
    await user.click(screen.getByRole("menuitemradio", { name: /lin\.personal@example\.com/ }));

    await waitFor(() => expect(switchProfile).toHaveBeenCalledWith("profile-personal"));
    expect(await screen.findByRole("button", { name: "切换账号，当前 lin.personal@example.com" })).toBeInTheDocument();
  });

  it("opens detailed account management from the header menu", async () => {
    const user = userEvent.setup();
    render(<App backendClient={createBackend()} />);

    await user.click(await screen.findByRole("button", { name: /切换账号，当前/ }));
    await user.click(screen.getByRole("menuitem", { name: "管理账号与控制服务器" }));

    expect(screen.getByRole("heading", { name: "控制服务器" })).toBeInTheDocument();
  });

  it("keeps server status in the detail heading and account count with accounts", async () => {
    const user = userEvent.setup();
    render(<App backendClient={createBackend()} />);

    await screen.findByRole("heading", { name: "服务不可用" });
    await user.click(screen.getByRole("button", { name: "网络与账号" }));

    expect(screen.getByText("当前网络")).toBeInTheDocument();
    expect(screen.getByText("可达")).toBeInTheDocument();
    expect(screen.queryByText("当前活动网络")).not.toBeInTheDocument();
    expect(within(screen.getByRole("region", { name: "账号" })).getByText("1 个账号")).toBeInTheDocument();
  });

  it("opens the requested detailed view from a native tray event", async () => {
    const backend = createBackend();
    let navigate: Parameters<typeof backend.subscribeNavigation>[0] | undefined;
    vi.spyOn(backend, "subscribeNavigation").mockImplementation((listener) => {
      navigate = listener;
      return () => undefined;
    });
    render(<App backendClient={backend} />);

    await screen.findByRole("heading", { name: "服务不可用" });
    await act(async () => navigate?.("settings"));

    expect(screen.getByRole("heading", { name: "设置", level: 1 })).toBeInTheDocument();
  });

  it("updates the recent path from an accurate ping snapshot", async () => {
    const user = userEvent.setup();
    const backend = createBackend();
    const initial = createDemoSnapshot();
    let publishSnapshot: Parameters<typeof backend.subscribe>[0] | undefined;
    vi.spyOn(backend, "getSnapshot").mockResolvedValue(initial);
    vi.spyOn(backend, "subscribe").mockImplementation((onSnapshot) => {
      publishSnapshot = onSnapshot;
      return () => undefined;
    });
    vi.spyOn(backend, "pingDevice").mockImplementation(async () => {
      const directSnapshot = createDemoSnapshot();
      const workstation = directSnapshot.devices.find((device) => device.id === "peer-workstation")!;
      workstation.connectionType = "direct";
      workstation.relayRegion = undefined;
      publishSnapshot?.(directSnapshot);
      return {
        deviceId: "peer-workstation",
        latencyMs: 24,
        via: "direct",
        endpoint: "203.0.113.8:41641",
      };
    });

    render(<App backendClient={backend} />);

    await screen.findByRole("heading", { name: "服务不可用" });
    await user.click(screen.getByRole("button", { name: "设备" }));
    await user.click(screen.getByRole("row", { name: /workstation/ }));
    expect(screen.getByText("DERP · Hong Kong")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Ping" }));

    expect(await screen.findByText("本次探测 · 24 ms · 直连")).toBeInTheDocument();
    expect(within(screen.getByRole("row", { name: /workstation/ })).getByText("直连")).toBeInTheDocument();
    expect(screen.queryByText("DERP · Hong Kong")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "概览" }));
    expect(screen.queryByText("中继 · Hong Kong")).not.toBeInTheDocument();
  });

  it("does not report a direct path when the probe has no route evidence", async () => {
    const user = userEvent.setup();
    const backend = createBackend();
    vi.spyOn(backend, "pingDevice").mockResolvedValue({
      deviceId: "peer-workstation",
      latencyMs: 50,
      via: "unknown",
    });

    render(<App backendClient={backend} />);

    await screen.findByRole("heading", { name: "服务不可用" });
    await user.click(screen.getByRole("button", { name: "设备" }));
    await user.click(screen.getByRole("row", { name: /workstation/ }));
    await user.click(screen.getByRole("button", { name: "Ping" }));

    expect(await screen.findByText("本次探测 · 50 ms · 路径未知")).toBeInTheDocument();
    expect(within(screen.getByRole("row", { name: /workstation/ })).getByText("中继")).toBeInTheDocument();
  });

  it("keeps the tunnel switch enabled when only the control server is unavailable", async () => {
    const user = userEvent.setup();
    const backend = createBackend();
    const disconnected = createDemoSnapshot();
    const limited = createDemoSnapshot();
    disconnected.runtime.connection = "stopped";
    disconnected.runtime.control = "unreachable";
    limited.runtime.connection = "degraded";
    limited.runtime.control = "unreachable";
    vi.spyOn(backend, "getSnapshot").mockResolvedValue(disconnected);
    vi.spyOn(backend, "setConnection").mockResolvedValue(limited);

    render(<App backendClient={backend} />);

    const toggle = await screen.findByRole("switch", { name: "连接" });
    await user.click(toggle);

    expect(await screen.findByText("隧道已启用，但控制服务器同步受限")).toBeInTheDocument();
    expect(screen.getByRole("switch", { name: "断开连接" })).toHaveAttribute("aria-checked", "true");
  });

  it("does not report the control server as unreachable for an unrelated health warning", async () => {
    const backend = createBackend();
    const degraded = createDemoSnapshot();
    degraded.runtime.daemon = "ready";
    degraded.runtime.connection = "degraded";
    degraded.runtime.control = "reachable";
    vi.spyOn(backend, "getSnapshot").mockResolvedValue(degraded);

    render(<App backendClient={backend} />);

    expect(await screen.findByRole("heading", { name: "隧道已连接" })).toBeInTheDocument();
    expect(screen.getByText("本地网络存在警告")).toBeInTheDocument();
    expect(screen.queryByText("控制服务器暂时不可达")).not.toBeInTheDocument();
    expect(screen.getByRole("switch", { name: "断开连接" })).toHaveAttribute("aria-checked", "true");
  });

  it("edits and deletes a custom control server", async () => {
    const user = userEvent.setup();
    render(<App backendClient={createBackend()} />);

    await screen.findByRole("heading", { name: "服务不可用" });
    await user.click(screen.getByRole("button", { name: "网络与账号" }));
    await user.click(screen.getByRole("button", { name: "编辑 团队 Headscale" }));

    const name = screen.getByRole("textbox", { name: "名称" });
    await user.clear(name);
    await user.type(name, "研发 Headscale");
    await user.click(screen.getByRole("button", { name: "保存服务器" }));
    const deleteButton = await screen.findByRole("button", { name: "删除 研发 Headscale" });
    await user.click(deleteButton);
    await user.click(screen.getByRole("button", { name: "删除" }));
    await waitFor(() => expect(screen.queryByRole("button", { name: "删除 研发 Headscale" })).not.toBeInTheDocument());
  });
});
