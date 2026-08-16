//go:build linux

package daemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/headscaleclient/headscaleclient/internal/domain"
)

type fakeLinuxRunner struct {
	units map[string]string
	calls []string
}

func (f *fakeLinuxRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	if len(args) >= 2 && args[0] == "show" {
		return []byte(f.units[args[1]]), nil
	}
	if len(args) == 2 && args[0] == "start" {
		unit := args[1]
		f.units[unit] = strings.Replace(f.units[unit], "ActiveState=inactive", "ActiveState=active", 1)
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected command: %s %s", name, strings.Join(args, " "))
}

func TestChooseLinuxUnitPrefersRunningManagedService(t *testing.T) {
	t.Parallel()

	external := linuxUnitStatus{name: linuxExternalUnit, loadState: "loaded", active: "inactive"}
	managed := linuxUnitStatus{name: linuxManagedUnit, loadState: "loaded", active: "active", managed: true}
	got, ok := chooseLinuxUnit(external, managed)
	if !ok || got.name != linuxManagedUnit || !got.managed {
		t.Fatalf("chooseLinuxUnit() = %+v, %v", got, ok)
	}
}

func TestChooseLinuxUnitPrefersRunningExternalService(t *testing.T) {
	t.Parallel()

	external := linuxUnitStatus{name: linuxExternalUnit, loadState: "loaded", active: "active"}
	managed := linuxUnitStatus{name: linuxManagedUnit, loadState: "loaded", active: "active", managed: true}
	got, ok := chooseLinuxUnit(external, managed)
	if !ok || got.name != linuxExternalUnit || got.managed {
		t.Fatalf("chooseLinuxUnit() = %+v, %v", got, ok)
	}
}

func TestParseSystemdPropertiesAndExecutable(t *testing.T) {
	t.Parallel()

	properties := parseSystemdProperties("LoadState=loaded\nActiveState=active\nExecStart={ path=/usr/lib/headscaleclient/daemon/tailscaled ; argv[]=/usr/lib/headscaleclient/daemon/tailscaled --state=test ; }\n")
	if properties["ActiveState"] != "active" {
		t.Fatalf("ActiveState = %q", properties["ActiveState"])
	}
	if got := systemdExecPath(properties["ExecStart"]); got != linuxManagedPayload {
		t.Fatalf("systemdExecPath() = %q", got)
	}
}

func TestMapLinuxServiceState(t *testing.T) {
	t.Parallel()

	tests := map[string]domain.EngineServiceState{
		"active":       domain.EngineServiceRunning,
		"activating":   domain.EngineServiceStarting,
		"deactivating": domain.EngineServiceStopping,
		"inactive":     domain.EngineServiceStopped,
		"failed":       domain.EngineServiceStopped,
		"maintenance":  domain.EngineServiceUnknown,
	}
	for active, want := range tests {
		if got := mapLinuxServiceState(active); got != want {
			t.Errorf("mapLinuxServiceState(%q) = %q, want %q", active, got, want)
		}
	}
}

func TestInspectLinuxManagedService(t *testing.T) {
	t.Parallel()

	payload := filepath.Join(t.TempDir(), "tailscaled")
	if err := os.WriteFile(payload, []byte("test payload"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := &fakeLinuxRunner{units: map[string]string{
		linuxExternalUnit: "LoadState=not-found\nActiveState=inactive\n",
		linuxManagedUnit:  "LoadState=loaded\nActiveState=inactive\nExecStart={ path=" + payload + " ; argv[]=" + payload + " ; }\n",
	}}
	manager := &platformManager{
		runner:       runner,
		systemctl:    "systemctl",
		pkexec:       "pkexec",
		payloadPaths: []string{payload},
		socketPath:   filepath.Join(t.TempDir(), "tailscaled.sock"),
	}
	status, err := manager.Inspect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.Ownership != domain.EngineOwnershipManaged || status.Service != domain.EngineServiceStopped || !status.CanStart {
		t.Fatalf("Inspect() = %+v", status)
	}
}

func TestEnsureInstalledStartsFixedLinuxUnit(t *testing.T) {
	t.Parallel()

	payload := filepath.Join(t.TempDir(), "tailscaled")
	if err := os.WriteFile(payload, []byte("test payload"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := &fakeLinuxRunner{units: map[string]string{
		linuxExternalUnit: "LoadState=not-found\nActiveState=inactive\n",
		linuxManagedUnit:  "LoadState=loaded\nActiveState=inactive\nExecStart={ path=" + payload + " ; argv[]=" + payload + " ; }\n",
	}}
	manager := &platformManager{
		runner:       runner,
		systemctl:    "systemctl",
		pkexec:       "pkexec",
		payloadPaths: []string{payload},
		socketPath:   filepath.Join(t.TempDir(), "tailscaled.sock"),
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.EnsureInstalled(ctx); err != nil {
		t.Fatal(err)
	}
	for _, call := range runner.calls {
		if call == "systemctl start "+linuxManagedUnit {
			return
		}
	}
	t.Fatalf("fixed service start was not called: %v", runner.calls)
}
