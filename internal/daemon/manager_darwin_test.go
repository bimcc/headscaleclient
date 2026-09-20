package daemon

import (
	"errors"
	"github.com/headscaleclient/headscaleclient/internal/domain"
	"testing"
)

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
