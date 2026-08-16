//go:build !windows && !linux

package daemon

import (
	"context"
	"os"
	"path/filepath"

	"github.com/headscaleclient/headscaleclient/internal/domain"
)

type platformManager struct{}

func newPlatformManager() Manager {
	return &platformManager{}
}

func (m *platformManager) Inspect(ctx context.Context) (domain.EngineStatus, error) {
	if err := ctx.Err(); err != nil {
		return domain.EngineStatus{}, domain.WrapError(domain.ErrorCancelled, "The network service inspection was cancelled.", err)
	}
	status := domain.EngineStatus{
		Ownership:      domain.EngineOwnershipUnknown,
		Service:        domain.EngineServiceUnknown,
		BundledVersion: BundledVersion,
	}
	executable, err := os.Executable()
	if err == nil {
		payload := filepath.Join(filepath.Dir(executable), "daemon", "tailscaled")
		if info, statErr := os.Stat(payload); statErr == nil && !info.IsDir() {
			status.Ownership = domain.EngineOwnershipPrepared
			status.PayloadAvailable = true
		}
	}
	return status, nil
}

func (m *platformManager) EnsureInstalled(context.Context) error {
	return domain.NewError(domain.ErrorUnsupported, "Managed network-service installation is not available on this platform build.")
}
