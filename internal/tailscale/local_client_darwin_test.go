package tailscale

import (
	"github.com/headscaleclient/headscaleclient/internal/macos"
	"testing"
)

func TestMacManagedClientCannotFallBackToOfficialGUI(t *testing.T) {
	managed := macLocalClient(true)
	if managed.Socket != macos.SocketPath || !managed.UseSocketOnly || !managed.OmitAuth {
		t.Fatalf("managed LocalAPI is not isolated: %+v", managed)
	}
	external := macLocalClient(false)
	if external.Socket != "" || external.UseSocketOnly || external.OmitAuth {
		t.Fatal("GUI-only builds must retain upstream discovery")
	}
}
