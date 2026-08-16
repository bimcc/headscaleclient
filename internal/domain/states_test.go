package domain

import "testing"

func TestDeriveDisplayState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		axes StateAxes
		want DisplayState
	}{
		{"unknown daemon", StateAxes{DaemonUnknown, SessionNone, ConnectionStopped, ControlUnknown}, DisplayUnknown},
		{"missing daemon", StateAxes{DaemonMissing, SessionNone, ConnectionStopped, ControlUnknown}, DisplayServiceUnavailable},
		{"unauthorized daemon", StateAxes{DaemonUnauthorized, SessionAuthenticated, ConnectionStopped, ControlUnknown}, DisplayServiceUnavailable},
		{"login required", StateAxes{DaemonReady, SessionLoginRequired, ConnectionStopped, ControlReachable}, DisplayLoginRequired},
		{"empty session", StateAxes{DaemonReady, SessionNone, ConnectionStopped, ControlReachable}, DisplayLoginRequired},
		{"approval required", StateAxes{DaemonReady, SessionApprovalRequired, ConnectionStarting, ControlReachable}, DisplayWaitingForApproval},
		{"starting", StateAxes{DaemonReady, SessionAuthenticated, ConnectionStarting, ControlReachable}, DisplayConnecting},
		{"running despite control outage", StateAxes{DaemonReady, SessionAuthenticated, ConnectionRunning, ControlUnreachable}, DisplayConnected},
		{"degraded", StateAxes{DaemonReady, SessionAuthenticated, ConnectionDegraded, ControlUnreachable}, DisplayLimitedConnectivity},
		{"stopped", StateAxes{DaemonReady, SessionAuthenticated, ConnectionStopped, ControlReachable}, DisplayDisconnected},
		{"invalid axis", StateAxes{DaemonReady, SessionState("broken"), ConnectionRunning, ControlReachable}, DisplayUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := DeriveDisplayState(tt.axes); got != tt.want {
				t.Fatalf("DeriveDisplayState() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProblemFromErrorDoesNotLeakUnknownErrors(t *testing.T) {
	t.Parallel()

	problem := ProblemFromError(assertionError("secret backend detail"))
	if problem.Code != ErrorInternal || problem.Message != "An unexpected error occurred." {
		t.Fatalf("unexpected problem: %+v", problem)
	}
	if problem.Detail != "" {
		t.Fatalf("unexpected leaked detail: %q", problem.Detail)
	}
}

type assertionError string

func (e assertionError) Error() string { return string(e) }
