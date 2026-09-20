package tailscale

import (
	"os"

	"github.com/headscaleclient/headscaleclient/internal/macos"
	"tailscale.com/client/local"
)

func newLocalClient() *local.Client {
	// A managed install must never fall through to a concurrently installed
	// official GUI's TCP LocalAPI, including while our daemon is stopped.
	_, err := os.Lstat(macos.LaunchDaemonPath)
	return macLocalClient(!os.IsNotExist(err))
}

func macLocalClient(managed bool) *local.Client {
	if managed {
		return &local.Client{Socket: macos.SocketPath, UseSocketOnly: true, OmitAuth: true}
	}
	return &local.Client{}
}
