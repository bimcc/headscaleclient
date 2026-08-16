package application

import (
	"github.com/headscaleclient/headscaleclient/internal/config"
	"github.com/headscaleclient/headscaleclient/internal/domain"
	"github.com/headscaleclient/headscaleclient/internal/tailscale"
)

// BackendBindings is the Go side of frontend/src/lib/contracts.ts. Keeping the
// assertion here makes accidental method or return-type drift a compile error.
type BackendBindings interface {
	GetSnapshot() (AppSnapshot, error)
	EnsureDaemon() (AppSnapshot, error)
	SetConnection(bool) (AppSnapshot, error)
	SetPreference(PreferenceKey, bool) (AppSnapshot, error)
	SetExitNode(*string) (AppSnapshot, error)
	PingDevice(string) (tailscale.PingResult, error)
	SaveEndpoint(EndpointInput) (AppSnapshot, error)
	DeleteEndpoint(string) (AppSnapshot, error)
	SwitchProfile(string) (AppSnapshot, error)
	Logout() (AppSnapshot, error)
	BeginLogin(string) (LoginResult, error)
	SetAppSetting(AppSettingKey, bool) (AppSnapshot, error)
	SetTheme(domain.Theme) (AppSnapshot, error)
	SetLanguage(domain.Language) (AppSnapshot, error)
}

var (
	_ BackendBindings = (*Service)(nil)
	_ Daemon          = (*tailscale.Adapter)(nil)
	_ Store           = (*config.Store)(nil)
)
