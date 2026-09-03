package desktop

import (
	"strings"
	"testing"

	appservice "github.com/headscaleclient/headscaleclient/internal/application"
	"github.com/headscaleclient/headscaleclient/internal/domain"
)

func TestProjectTrayBuildsQuickStateFromSnapshot(t *testing.T) {
	activeProfileID := "profile-work"
	exitNodeID := "offline-exit"
	snapshot := appservice.AppSnapshot{
		Runtime: appservice.RuntimeState{
			Daemon:     domain.DaemonReady,
			Session:    domain.SessionAuthenticated,
			Connection: domain.ConnectionRunning,
			Control:    domain.ControlReachable,
		},
		LocalDevice: appservice.LocalDevice{
			Name:      "desktop",
			Addresses: []string{"100.64.0.1"},
		},
		Profiles: []appservice.LoginProfile{
			{ID: activeProfileID, EndpointID: "team", Account: "work@example.com", State: appservice.ProfileStateReady},
			{ID: "profile-login", DisplayName: "Needs login", State: appservice.ProfileStateLoginRequired},
		},
		Endpoints:       []appservice.Endpoint{{ID: "team", Name: "Team Headscale"}},
		ActiveProfileID: &activeProfileID,
		Devices: []appservice.PeerDevice{
			{
				ID: "direct", Name: "laptop", Addresses: []string{"100.64.0.2"},
				Online: true, Group: "home", ConnectionType: appservice.ConnectionTypeDirect,
			},
			{
				ID: "relay-exit", Name: "relay", Online: true,
				ConnectionType: appservice.ConnectionTypeRelay, RelayRegion: "Hong Kong",
				ExitNodeOption: true,
			},
			{
				ID: "offline-exit", Name: "offline exit", Online: false,
				ConnectionType: appservice.ConnectionTypeOffline, ExitNodeOption: true,
			},
		},
		Preferences: appservice.QuickPreferences{
			ExitNodeID: exitNodeIDPointer(exitNodeID), AcceptDNS: true, AcceptRoutes: true,
		},
	}

	model := projectTray(snapshot)

	if model.StatusLabel != "已连接" || !model.Connected || !model.CanToggle || !model.CanConfigure {
		t.Fatalf("unexpected connection projection: %+v", model)
	}
	if model.AccountLabel != "work@example.com · Team Headscale" || model.DeviceLabel != "desktop · 100.64.0.1" {
		t.Fatalf("unexpected identity projection: account=%q device=%q", model.AccountLabel, model.DeviceLabel)
	}
	if len(model.Profiles) != 2 || !model.Profiles[0].Active || !model.Profiles[1].Switchable || !strings.Contains(model.Profiles[1].Label, "需登录") {
		t.Fatalf("unexpected profile projection: %+v", model.Profiles)
	}
	if len(model.Devices) != 2 {
		t.Fatalf("online devices = %d, want 2", len(model.Devices))
	}
	if !strings.Contains(model.Devices[0].Label, "直连") || !strings.Contains(model.Devices[1].Label, "中继 Hong Kong") {
		t.Fatalf("unexpected path labels: %+v", model.Devices)
	}
	if model.Devices[0].Group != "home" {
		t.Fatalf("device group = %q, want home", model.Devices[0].Group)
	}
	if len(model.ExitNodes) != 2 || model.ExitNodes[0].ID != "relay-exit" || model.ExitNodes[1].ID != "offline-exit" {
		t.Fatalf("eligible exit nodes were not preserved: %+v", model.ExitNodes)
	}
	if !strings.Contains(model.ExitNodes[1].Label, "离线") {
		t.Fatalf("offline exit node label = %q", model.ExitNodes[1].Label)
	}
	if model.ExitNodeID != exitNodeID || !model.AcceptDNS || !model.AcceptRoutes {
		t.Fatalf("unexpected preference projection: %+v", model)
	}
}

func TestProjectTrayDistinguishesControlSyncWarning(t *testing.T) {
	snapshot := appservice.AppSnapshot{Runtime: appservice.RuntimeState{
		Daemon:     domain.DaemonReady,
		Session:    domain.SessionAuthenticated,
		Connection: domain.ConnectionDegraded,
		Control:    domain.ControlUnreachable,
	}}

	model := projectTray(snapshot)

	if model.StatusLabel != "隧道已连接 · 控制服务器同步受限" || !model.Connected {
		t.Fatalf("unexpected degraded projection: %+v", model)
	}
}

func TestProjectTrayUsesEnglishPreference(t *testing.T) {
	snapshot := appservice.AppSnapshot{
		Runtime: appservice.RuntimeState{
			Daemon: domain.DaemonReady, Session: domain.SessionAuthenticated,
			Connection: domain.ConnectionRunning, Control: domain.ControlReachable,
		},
		Settings: appservice.AppSettings{Language: domain.LanguageEnglish},
		Profiles: []appservice.LoginProfile{{ID: "profile", Account: "user@example.com", State: appservice.ProfileStateLoginRequired}},
	}

	model := projectTray(snapshot)

	if model.Language != domain.LanguageEnglish || model.StatusLabel != "Connected" {
		t.Fatalf("unexpected English tray status: %+v", model)
	}
	if len(model.Profiles) != 1 || !strings.Contains(model.Profiles[0].Label, "Sign-in required") {
		t.Fatalf("unexpected English profile label: %+v", model.Profiles)
	}
}

func TestTruncateMenuLabel(t *testing.T) {
	if got := truncateMenuLabel("  one\n two  ", 20); got != "one two" {
		t.Fatalf("normalized label = %q", got)
	}
	if got := truncateMenuLabel("一二三四五", 4); got != "一二三…" {
		t.Fatalf("truncated label = %q", got)
	}
}

func exitNodeIDPointer(value string) *string {
	return &value
}
