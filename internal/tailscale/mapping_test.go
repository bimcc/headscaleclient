package tailscale

import (
	"net/netip"
	"testing"

	"github.com/headscaleclient/headscaleclient/internal/domain"
	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
	"tailscale.com/types/views"
)

func TestMapSnapshotNormalizesOfficialControlURL(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name       string
		controlURL string
	}{
		{name: "empty default", controlURL: ""},
		{name: "legacy login host", controlURL: "https://login.tailscale.com"},
		{name: "canonical host with case and slash", controlURL: "HTTPS://CONTROLPLANE.TAILSCALE.COM/"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			snapshot := mapSnapshot(
				&ipnstate.Status{BackendState: "NeedsLogin"},
				&ipn.Prefs{ControlURL: test.controlURL},
			)
			if snapshot.ActiveEndpoint == nil {
				t.Fatal("active endpoint is nil")
			}
			if got := snapshot.ActiveEndpoint.BaseURL; got != ipn.DefaultControlURL {
				t.Fatalf("active endpoint URL = %q, want %q", got, ipn.DefaultControlURL)
			}
			if got := snapshot.ActiveEndpoint.Provider; got != domain.ProviderTailscale {
				t.Fatalf("active endpoint provider = %q, want %q", got, domain.ProviderTailscale)
			}
		})
	}
}

func TestMapDeviceProvidesStableDisplayNameFallback(t *testing.T) {
	t.Parallel()

	fromControlServerName := mapDevice(&ipnstate.Status{MagicDNSSuffix: "bimcc.internal"}, &ipnstate.PeerStatus{
		ID:       tailcfg.StableNodeID("node-renamed"),
		HostName: "bimcc",
		DNSName:  "bimcc-188.bimcc.internal.",
	})
	if fromControlServerName.Name != "bimcc-188" {
		t.Fatalf("control-server display name = %q", fromControlServerName.Name)
	}

	fromAddress := mapDevice(&ipnstate.Status{}, &ipnstate.PeerStatus{
		ID:           tailcfg.StableNodeID("node-address"),
		TailscaleIPs: []netip.Addr{netip.MustParseAddr("100.64.0.8")},
	})
	if fromAddress.Name != "100.64.0.8" {
		t.Fatalf("address fallback name = %q", fromAddress.Name)
	}

	fromID := mapDevice(&ipnstate.Status{}, &ipnstate.PeerStatus{ID: tailcfg.StableNodeID("node-id")})
	if fromID.Name != "node-id" {
		t.Fatalf("ID fallback name = %q", fromID.Name)
	}
}

func TestMapDeviceProvidesGroupAndACLTags(t *testing.T) {
	t.Parallel()

	status := &ipnstate.Status{
		MagicDNSSuffix: "mesh.example",
		User: map[tailcfg.UserID]tailcfg.UserProfile{
			42: {LoginName: "home", DisplayName: "Liao Xiaofeng"},
		},
	}
	peer := &ipnstate.PeerStatus{
		ID:     tailcfg.StableNodeID("node-1"),
		UserID: 42,
		Tags:   new(views.SliceOf([]string{"tag:server", "tag:dev"})),
	}

	device := mapDevice(status, peer)
	if device.User != "Liao Xiaofeng" || device.Group != "home" {
		t.Fatalf("owner/group = (%q, %q), want (%q, %q)", device.User, device.Group, "Liao Xiaofeng", "home")
	}
	if got := device.Tags; len(got) != 2 || got[0] != "tag:server" || got[1] != "tag:dev" {
		t.Fatalf("tags = %#v", got)
	}
}

func TestMapSnapshotHandlesNilStatus(t *testing.T) {
	t.Parallel()

	snapshot := mapSnapshot(nil, &ipn.Prefs{})
	if snapshot.State.Daemon != domain.DaemonUnknown {
		t.Fatalf("daemon state = %q, want %q", snapshot.State.Daemon, domain.DaemonUnknown)
	}
	if snapshot.DisplayState != domain.DisplayUnknown {
		t.Fatalf("display state = %q, want %q", snapshot.DisplayState, domain.DisplayUnknown)
	}
	if snapshot.Peers == nil || len(snapshot.Peers) != 0 {
		t.Fatalf("peers = %#v, want an empty non-nil slice", snapshot.Peers)
	}
}

func TestMapSnapshotPreservesHealthWarningDetails(t *testing.T) {
	t.Parallel()

	snapshot := mapSnapshot(&ipnstate.Status{
		BackendState: "Running",
		Health:       []string{" UDP is unavailable; using relays. ", ""},
	}, &ipn.Prefs{})
	if len(snapshot.HealthNotices) != 1 ||
		snapshot.HealthNotices[0].Code != domain.HealthNoticeTailscaleWarning ||
		snapshot.HealthNotices[0].Severity != domain.HealthNoticeWarning ||
		snapshot.HealthNotices[0].Message != "UDP is unavailable; using relays." {
		t.Fatalf("health notices = %#v", snapshot.HealthNotices)
	}
}

func TestMapSnapshotClassifiesDisabledAcceptRoutesAsInformation(t *testing.T) {
	t.Parallel()

	message := "Some peers are advertising routes but --accept-routes is false"
	snapshot := mapSnapshot(&ipnstate.Status{BackendState: "Running", Health: []string{message}}, &ipn.Prefs{})
	if len(snapshot.HealthNotices) != 1 ||
		snapshot.HealthNotices[0].Code != domain.HealthNoticeRoutesNotAccepted ||
		snapshot.HealthNotices[0].Severity != domain.HealthNoticeInfo ||
		snapshot.HealthNotices[0].Message != message {
		t.Fatalf("health notices = %#v", snapshot.HealthNotices)
	}
}

func TestMapProfilePreservesNormalizedControlURL(t *testing.T) {
	t.Parallel()

	profile := mapProfile(ipn.LoginProfile{
		ID:         ipn.ProfileID("profile-1"),
		ControlURL: "https://headscale.example/control/",
	}, true)
	if profile.ControlURL != "https://headscale.example/control" {
		t.Fatalf("profile control URL = %q", profile.ControlURL)
	}
	if !profile.Active {
		t.Fatal("profile was not marked active")
	}

	official := mapProfile(ipn.LoginProfile{ID: ipn.ProfileID("official")}, false)
	if official.ControlURL != ipn.DefaultControlURL {
		t.Fatalf("official profile control URL = %q, want %q", official.ControlURL, ipn.DefaultControlURL)
	}
}

func TestMapPeerConnection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		peer        *ipnstate.PeerStatus
		connection  domain.PeerConnectionType
		relayRegion string
	}{
		{name: "nil", connection: domain.PeerConnectionOffline},
		{name: "offline wins over stale route", peer: &ipnstate.PeerStatus{CurAddr: "192.0.2.1:41641", Relay: "hkg"}, connection: domain.PeerConnectionOffline},
		{name: "direct", peer: &ipnstate.PeerStatus{Online: true, CurAddr: "192.0.2.1:41641", Relay: "hkg"}, connection: domain.PeerConnectionDirect},
		{name: "derp relay", peer: &ipnstate.PeerStatus{Online: true, Relay: "hkg"}, connection: domain.PeerConnectionRelay, relayRegion: "hkg"},
		{name: "peer relay", peer: &ipnstate.PeerStatus{Online: true, PeerRelay: "100.64.0.2:1234:vni:1"}, connection: domain.PeerConnectionRelay},
		{name: "online without route", peer: &ipnstate.PeerStatus{Online: true}, connection: domain.PeerConnectionUnknown},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			connection, relayRegion := mapPeerConnection(test.peer)
			if connection != test.connection || relayRegion != test.relayRegion {
				t.Fatalf("mapPeerConnection() = (%q, %q), want (%q, %q)", connection, relayRegion, test.connection, test.relayRegion)
			}
		})
	}
}
