package daemon

import (
	"context"

	"github.com/headscaleclient/headscaleclient/internal/domain"
)

const BundledVersion = "1.102.2"

// Manager is the privileged-daemon lifecycle boundary. Implementations expose
// only fixed operations and never accept executable paths or command arguments
// from the frontend.
type Manager interface {
	Inspect(context.Context) (domain.EngineStatus, error)
	EnsureInstalled(context.Context) error
}

func NewManager() Manager {
	return newPlatformManager()
}
