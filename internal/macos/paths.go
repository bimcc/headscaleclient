// Package macos defines the fixed paths shared by the desktop and launchd service.
package macos

const (
	ServiceLabel     = "io.headscaleclient.tailscaled"
	ServiceRoot      = "/Library/Application Support/BIMCC/HeadscaleClient"
	LaunchDaemonPath = "/Library/LaunchDaemons/" + ServiceLabel + ".plist"
	DaemonPath       = ServiceRoot + "/daemon/tailscaled"
	ControlPath      = ServiceRoot + "/service-control"
	SocketPath       = "/var/run/headscaleclient-tailscaled.socket"
)
