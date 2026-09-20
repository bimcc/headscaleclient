package bridge

import (
	"github.com/headscaleclient/headscaleclient/internal/domain"
	"slices"
	"strings"
)

// FilterVPNHealth removes only the upstream's normal stopped-state notice when
// neither the platform nor the backend has been asked to connect.
func FilterVPNHealth(notices []domain.HealthNotice, desired, wantRunning bool) []domain.HealthNotice {
	if desired || wantRunning {
		return notices
	}
	return slices.DeleteFunc(notices, func(notice domain.HealthNotice) bool {
		return notice.Code == domain.HealthNoticeTailscaleWarning && strings.TrimSpace(notice.Message) == "Tailscale is stopped."
	})
}

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
