package bridge

import "github.com/headscaleclient/headscaleclient/internal/domain"

// VPNConnectionState never equates a logged-in backend with an established TUN.
func VPNConnectionState(state domain.ConnectionState, desired, established bool) domain.ConnectionState {
	if !desired {
		return domain.ConnectionStopped
	}
	if !established && (state == domain.ConnectionRunning || state == domain.ConnectionDegraded) {
		return domain.ConnectionStarting
	}
	return state
}
