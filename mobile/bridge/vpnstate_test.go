package bridge

import (
	"github.com/headscaleclient/headscaleclient/internal/domain"
	"testing"
)

func TestStoppedNoticeIsOnlySuppressedForAnIntentionalDisconnect(t *testing.T) {
	for _, tc := range []struct {
		desired, want bool
		count         int
	}{{false, false, 1}, {true, false, 2}, {false, true, 2}} {
		notices := []domain.HealthNotice{
			{Code: domain.HealthNoticeTailscaleWarning, Message: "Tailscale is stopped."},
			{Code: domain.HealthNoticeTailscaleWarning, Message: "DNS unavailable"},
		}
		got := FilterVPNHealth(notices, tc.desired, tc.want)
		if len(got) != tc.count || got[len(got)-1].Message != "DNS unavailable" {
			t.Fatalf("unexpected health filtering: %+v", got)
		}
	}
}

func TestVPNOwnershipAndTunnelAreRequiredForConnected(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		state                domain.ConnectionState
		desired, established bool
		want                 domain.ConnectionState
	}{
		{"revoked", domain.ConnectionRunning, false, true, domain.ConnectionStopped},
		{"not yet established", domain.ConnectionRunning, true, false, domain.ConnectionStarting},
		{"backend warning but no tunnel", domain.ConnectionDegraded, true, false, domain.ConnectionStarting},
		{"connected", domain.ConnectionRunning, true, true, domain.ConnectionRunning},
		{"backend still starting", domain.ConnectionStarting, true, true, domain.ConnectionStarting},
		{"backend disconnected", domain.ConnectionStopped, true, true, domain.ConnectionStopped},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := VPNConnectionState(tc.state, tc.desired, tc.established); got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}
