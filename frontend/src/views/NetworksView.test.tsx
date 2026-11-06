import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";
import type { Endpoint, LoginProfile } from "../lib/contracts";
import { NetworksView } from "./NetworksView";

const endpoints: Endpoint[] = [
  {
    id: "endpoint-headscale",
    name: "团队 Headscale",
    url: "https://headscale.example.com",
    kind: "headscale",
    status: "unchecked",
    customCa: false,
    builtIn: false,
  },
  {
    id: "endpoint-tailscale",
    name: "Tailscale 官方",
    url: "https://login.tailscale.com",
    kind: "tailscale",
    status: "reachable",
    customCa: false,
    builtIn: true,
  },
];

const profiles: LoginProfile[] = [
  {
    id: "profile-tailscale",
    endpointId: "endpoint-tailscale",
    account: "user@example.com",
    displayName: "个人网络",
    active: true,
    state: "ready",
    lastUsedAt: "刚刚",
  },
];

function renderView(overrides: Partial<ComponentProps<typeof NetworksView>> = {}) {
  const props: ComponentProps<typeof NetworksView> = {
    endpoints,
    profiles,
    activeEndpointId: "endpoint-tailscale",
    activeProfileId: "profile-tailscale",
    onSaveEndpoint: vi.fn(async () => undefined),
    onDeleteEndpoint: vi.fn(async () => undefined),
    onSwitchProfile: vi.fn(async () => undefined),
    onRemoveProfile: vi.fn(async () => undefined),
    onBeginLogin: vi.fn(async (endpointId) => ({
      endpointId,
      authUrl: "https://login.example.com/device",
    })),
    onReauthenticateProfile: vi.fn(async (endpointId) => ({
      endpointId,
      authUrl: "https://login.example.com/device",
    })),
    onOpenURL: vi.fn(async () => undefined),
    ...overrides,
  };
  render(<NetworksView {...props} />);
  return props;
}

describe("NetworksView", () => {
  it("starts login for the exact endpoint row that was selected", async () => {
    const user = userEvent.setup();
    const onBeginLogin = vi.fn(async (endpointId: string) => ({
      endpointId,
      authUrl: "https://login.example.com/device",
    }));
    const props = renderView({ onBeginLogin });

    await user.click(screen.getByRole("option", { name: /团队 Headscale/ }));
    await user.click(screen.getByRole("button", { name: "登录 团队 Headscale" }));

    await waitFor(() => expect(onBeginLogin).toHaveBeenCalledWith("endpoint-headscale"));
    expect(onBeginLogin).not.toHaveBeenCalledWith("endpoint-tailscale");
    expect(props.onOpenURL).toHaveBeenCalledWith("https://login.example.com/device");
    expect(screen.queryByText("未检查")).not.toBeInTheDocument();
    expect(screen.getByText("登录时检查")).toBeInTheDocument();
    expect(screen.getByText(/未配置 OIDC 的 Headscale 会要求管理员批准/)).toBeInTheDocument();
  });

  it("shows only accounts associated with the selected server", async () => {
    const user = userEvent.setup();
    const teamProfile: LoginProfile = {
      id: "profile-headscale",
      endpointId: "endpoint-headscale",
      account: "team@example.com",
      displayName: "团队网络",
      active: false,
      state: "ready",
      lastUsedAt: "昨天",
    };
    renderView({ profiles: [...profiles, teamProfile] });

    expect(screen.getByText("个人网络")).toBeInTheDocument();
    expect(screen.queryByText("团队网络")).not.toBeInTheDocument();

    await user.click(screen.getByRole("option", { name: /团队 Headscale/ }));

    expect(screen.getByText("团队网络")).toBeInTheDocument();
    expect(screen.queryByText("个人网络")).not.toBeInTheDocument();
  });

  it("supports arrow-key navigation between servers", async () => {
    const user = userEvent.setup();
    renderView();

    const tailscale = screen.getByRole("option", { name: /Tailscale 官方/ });
    tailscale.focus();
    await user.keyboard("{ArrowUp}");

    expect(screen.getByRole("option", { name: /团队 Headscale/ })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("heading", { name: "团队 Headscale" })).toBeInTheDocument();
  });

  it("explains reauthentication before removing the active local identity", async () => {
    const user = userEvent.setup();
    const onRemoveProfile = vi.fn(async () => undefined);
    renderView({ onRemoveProfile });

    const profileRow = screen.getByText("个人网络").closest(".profile-row");
    expect(profileRow).not.toBeNull();
    await user.click(within(profileRow as HTMLElement).getByRole("button", { name: "退出并移除" }));

    expect(onRemoveProfile).not.toHaveBeenCalled();
    const dialog = screen.getByRole("dialog", { name: "退出并移除本机登录身份" });
    expect(screen.getByText(/下次使用需重新打开浏览器认证/)).toBeInTheDocument();
    expect(screen.getByText(/若只想暂时停用网络/)).toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "退出并移除" }));

    await waitFor(() => expect(onRemoveProfile).toHaveBeenCalledWith("profile-tailscale"));
  });

  it("offers in-place reauthentication for an active identity that needs login", async () => {
    const user = userEvent.setup();
    const onReauthenticateProfile = vi.fn(async (endpointId: string) => ({
      endpointId,
      authUrl: "https://login.example.com/reauth",
    }));
    const profile = { ...profiles[0]!, state: "login-required" as const };
    renderView({ profiles: [profile], onReauthenticateProfile });

    const profileRow = screen.getByText("个人网络").closest(".profile-row");
    expect(profileRow).not.toBeNull();
    await user.click(within(profileRow as HTMLElement).getByRole("button", { name: "重新登录" }));

    await waitFor(() => expect(onReauthenticateProfile).toHaveBeenCalledWith("endpoint-tailscale", "profile-tailscale"));
    expect(screen.getByRole("status")).toHaveTextContent("login.example.com/reauth");
  });

  it("removes an inactive saved identity without switching into it", async () => {
    const user = userEvent.setup();
    const inactiveProfile: LoginProfile = {
      ...profiles[0]!,
      id: "profile-old",
      displayName: "旧身份",
      account: "old@example.com",
      active: false,
      state: "ready",
    };
    const onRemoveProfile = vi.fn(async () => undefined);
    const onSwitchProfile = vi.fn(async () => undefined);
    renderView({ profiles: [...profiles, inactiveProfile], onRemoveProfile, onSwitchProfile });

    const profileRow = screen.getByText("旧身份").closest(".profile-row");
    expect(profileRow).not.toBeNull();
    await user.click(within(profileRow as HTMLElement).getByRole("button", { name: "移除" }));
    const dialog = screen.getByRole("dialog", { name: "移除本机登录身份" });
    await user.click(within(dialog).getByRole("button", { name: "确认移除" }));

    await waitFor(() => expect(onRemoveProfile).toHaveBeenCalledWith("profile-old"));
    expect(onSwitchProfile).not.toHaveBeenCalled();
  });

  it("shows a persistent endpoint error and restores login after a failed probe", async () => {
    const user = userEvent.setup();
    const onBeginLogin = vi.fn(async () => {
      throw new Error("控制服务器 headscale.example.com 当前不可用（HTTP 503）。");
    });
    renderView({ onBeginLogin });

    await user.click(screen.getByRole("option", { name: /团队 Headscale/ }));
    const loginButton = screen.getByRole("button", { name: "登录 团队 Headscale" });
    await user.click(loginButton);

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("无法连接 团队 Headscale");
    expect(alert).toHaveTextContent("HTTP 503");
    await waitFor(() => expect(loginButton).toBeEnabled());
  });
});
