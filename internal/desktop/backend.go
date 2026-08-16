package desktop

import (
	"context"

	appservice "github.com/headscaleclient/headscaleclient/internal/application"
	"github.com/headscaleclient/headscaleclient/internal/domain"
	"github.com/headscaleclient/headscaleclient/internal/tailscale"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Backend is the intentionally narrow Wails binding surface. Lifecycle and
// orchestration stay on the internal application service.
type Backend struct {
	service *appservice.Service
}

var _ appservice.BackendBindings = (*Backend)(nil)

func NewBackend(service *appservice.Service) *Backend {
	return &Backend{service: service}
}

func (b *Backend) ServiceStartup(context.Context, application.ServiceOptions) error {
	b.service.Start()
	return nil
}

func (b *Backend) ServiceShutdown() error {
	b.service.Close()
	return nil
}

func (b *Backend) GetSnapshot() (appservice.AppSnapshot, error) {
	return b.service.GetSnapshot()
}

func (b *Backend) EnsureDaemon() (appservice.AppSnapshot, error) {
	return b.service.EnsureDaemon()
}

func (b *Backend) SetConnection(enabled bool) (appservice.AppSnapshot, error) {
	return b.service.SetConnection(enabled)
}

func (b *Backend) SetPreference(key appservice.PreferenceKey, value bool) (appservice.AppSnapshot, error) {
	return b.service.SetPreference(key, value)
}

func (b *Backend) SetExitNode(deviceID *string) (appservice.AppSnapshot, error) {
	return b.service.SetExitNode(deviceID)
}

func (b *Backend) PingDevice(deviceID string) (tailscale.PingResult, error) {
	return b.service.PingDevice(deviceID)
}

func (b *Backend) SaveEndpoint(input appservice.EndpointInput) (appservice.AppSnapshot, error) {
	return b.service.SaveEndpoint(input)
}

func (b *Backend) DeleteEndpoint(endpointID string) (appservice.AppSnapshot, error) {
	return b.service.DeleteEndpoint(endpointID)
}

func (b *Backend) SwitchProfile(profileID string) (appservice.AppSnapshot, error) {
	return b.service.SwitchProfile(profileID)
}

func (b *Backend) Logout() (appservice.AppSnapshot, error) {
	return b.service.Logout()
}

func (b *Backend) BeginLogin(endpointID string) (appservice.LoginResult, error) {
	return b.service.BeginLogin(endpointID)
}

func (b *Backend) SetAppSetting(key appservice.AppSettingKey, value bool) (appservice.AppSnapshot, error) {
	return b.service.SetAppSetting(key, value)
}

func (b *Backend) SetTheme(theme domain.Theme) (appservice.AppSnapshot, error) {
	return b.service.SetTheme(theme)
}

func (b *Backend) SetLanguage(language domain.Language) (appservice.AppSnapshot, error) {
	return b.service.SetLanguage(language)
}
