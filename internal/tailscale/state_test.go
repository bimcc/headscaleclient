package tailscale

import (
	"testing"

	"github.com/headscaleclient/headscaleclient/internal/domain"
	"tailscale.com/ipn/ipnstate"
)

func TestMapState(t *testing.T) {
	tests := []struct {
		name    string
		status  ipnstate.Status
		daemon  domain.DaemonState
		session domain.SessionState
		conn    domain.ConnectionState
		display domain.DisplayState
	}{
		{name: "other user", status: ipnstate.Status{BackendState: "InUseOtherUser"}, daemon: domain.DaemonUnauthorized, session: domain.SessionNone, conn: domain.ConnectionStopped, display: domain.DisplayServiceUnavailable},
		{name: "no state", status: ipnstate.Status{BackendState: "NoState"}, daemon: domain.DaemonReady, session: domain.SessionNone, conn: domain.ConnectionStopped, display: domain.DisplayLoginRequired},
		{name: "unknown backend", status: ipnstate.Status{BackendState: "FutureState", HaveNodeKey: true}, daemon: domain.DaemonIncompatible, session: domain.SessionNone, conn: domain.ConnectionStopped, display: domain.DisplayServiceUnavailable},
		{name: "login", status: ipnstate.Status{BackendState: "NeedsLogin"}, daemon: domain.DaemonReady, session: domain.SessionLoginRequired, conn: domain.ConnectionStopped, display: domain.DisplayLoginRequired},
		{name: "approval", status: ipnstate.Status{BackendState: "NeedsMachineAuth"}, daemon: domain.DaemonReady, session: domain.SessionApprovalRequired, conn: domain.ConnectionStopped, display: domain.DisplayWaitingForApproval},
		{name: "running", status: ipnstate.Status{BackendState: "Running"}, daemon: domain.DaemonReady, session: domain.SessionAuthenticated, conn: domain.ConnectionRunning, display: domain.DisplayConnected},
		{name: "route information", status: ipnstate.Status{BackendState: "Running", Health: []string{"Some peers are advertising routes but --accept-routes is false"}}, daemon: domain.DaemonReady, session: domain.SessionAuthenticated, conn: domain.ConnectionRunning, display: domain.DisplayConnected},
		{name: "degraded", status: ipnstate.Status{BackendState: "Running", Health: []string{"control server unreachable"}}, daemon: domain.DaemonReady, session: domain.SessionAuthenticated, conn: domain.ConnectionDegraded, display: domain.DisplayLimitedConnectivity},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := mapState(&test.status)
			if state.Daemon != test.daemon || state.Session != test.session || state.Connection != test.conn {
				t.Fatalf("got daemon=%s session=%s connection=%s", state.Daemon, state.Session, state.Connection)
			}
			if got := domain.DeriveDisplayState(state); got != test.display {
				t.Fatalf("display=%s, want %s", got, test.display)
			}
		})
	}
}
