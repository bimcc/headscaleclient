//go:build windows

package daemon

import (
	"path/filepath"
	"testing"

	"github.com/headscaleclient/headscaleclient/internal/domain"
	"golang.org/x/sys/windows/svc"
)

func TestSameExecutableNormalizesQuotesAndCase(t *testing.T) {
	t.Parallel()

	path := filepath.Join(`C:\Program Files`, "HeadscaleClient", "daemon", "tailscaled.exe")
	quoted := `"C:\PROGRAM FILES\HEADSCALECLIENT\daemon\tailscaled.exe"`
	if !sameExecutable(path, quoted) {
		t.Fatalf("sameExecutable(%q, %q) = false", path, quoted)
	}
	if sameExecutable(path, `C:\Program Files\Tailscale\tailscaled.exe`) {
		t.Fatal("sameExecutable accepted an external service path")
	}
}

func TestMatchesManagedWindowsExecutableIncludesLegacyDefaultPath(t *testing.T) {
	t.Parallel()

	current := `C:\Program Files\BIMCC\HeadscaleClient\daemon\tailscaled.exe`
	legacy := `C:\Program Files\BIMCC., Ltd.\HeadscaleClient\daemon\tailscaled.exe`
	for _, path := range []string{current, legacy} {
		if !matchesManagedWindowsExecutable(`"`+path+`"`, current, legacy) {
			t.Fatalf("managed path %q was not recognized", path)
		}
	}
	if matchesManagedWindowsExecutable(`"C:\Program Files\Tailscale\tailscaled.exe"`, current, legacy) {
		t.Fatal("official Tailscale path was classified as managed")
	}
}

func TestMapWindowsServiceState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state svc.State
		want  domain.EngineServiceState
	}{
		{state: svc.Stopped, want: domain.EngineServiceStopped},
		{state: svc.StartPending, want: domain.EngineServiceStarting},
		{state: svc.Running, want: domain.EngineServiceRunning},
		{state: svc.StopPending, want: domain.EngineServiceStopping},
		{state: svc.Paused, want: domain.EngineServiceUnknown},
	}

	for _, test := range tests {
		if got := mapWindowsServiceState(test.state); got != test.want {
			t.Errorf("mapWindowsServiceState(%v) = %q, want %q", test.state, got, test.want)
		}
	}
}
