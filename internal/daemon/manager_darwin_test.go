package daemon

import (
	"errors"
	"os"
	"testing"

	"github.com/headscaleclient/headscaleclient/internal/domain"
	"github.com/headscaleclient/headscaleclient/internal/macos"
)

func TestDarwinExternalInstallDetection(t *testing.T) {
	for _, test := range []struct {
		name, marker, installedPath string
		want                        bool
	}{
		{"external mode survives stopped daemon", "external\n", "", true},
		{"managed mode is not external", "managed\n", "", false},
		{"unknown marker is not authority", "unexpected", "", false},
		{"official app", "", "/Applications/Tailscale.app", true},
		{"stopped homebrew", "", "/Library/LaunchDaemons/homebrew.mxcl.tailscale.plist", true},
		{"fresh machine", "", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			read := func(path string) ([]byte, error) {
				if path != macos.InstallModePath {
					t.Fatalf("unexpected read: %s", path)
				}
				return []byte(test.marker), nil
			}
			stat := func(path string) (os.FileInfo, error) {
				if path == test.installedPath {
					return nil, nil
				}
				return nil, os.ErrNotExist
			}
			if got := darwinExternalInstalled(read, stat); got != test.want {
				t.Fatalf("external = %v, want %v", got, test.want)
			}
		})
	}
}

func TestDarwinServiceStateDoesNotTreatLoadedAsRunning(t *testing.T) {
	for _, test := range []struct {
		output string
		err    error
		want   domain.EngineServiceState
	}{
		{"state = running\n", nil, domain.EngineServiceRunning},
		{"state = waiting\nlast exit code = 1", nil, domain.EngineServiceStopped},
		{"", errors.New("not loaded"), domain.EngineServiceStopped},
		{"state = running\n", errors.New("permission denied"), domain.EngineServiceStopped},
	} {
		if got := darwinServiceState(test.output, test.err); got != test.want {
			t.Errorf("state(%q, %v) = %q, want %q", test.output, test.err, got, test.want)
		}
	}
}
