package domain

type EndpointProvider string

const (
	ProviderAuto       EndpointProvider = "auto"
	ProviderHeadscale  EndpointProvider = "headscale"
	ProviderTailscale  EndpointProvider = "tailscale"
	ProviderCompatible EndpointProvider = "compatible"
)

func (p EndpointProvider) Valid() bool {
	switch p {
	case ProviderAuto, ProviderHeadscale, ProviderTailscale, ProviderCompatible:
		return true
	default:
		return false
	}
}

type ControlEndpoint struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	BaseURL          string           `json:"baseURL"`
	Provider         EndpointProvider `json:"provider"`
	CustomCARef      string           `json:"customCARef,omitempty"`
	DaemonProfileIDs []string         `json:"daemonProfileIDs"`
	BuiltIn          bool             `json:"builtIn"`
	CreatedAt        string           `json:"createdAt"`
	UpdatedAt        string           `json:"updatedAt"`
}

// ControlEndpointInput is used for both create and update. ID is empty when
// creating a record and required when updating one.
type ControlEndpointInput struct {
	ID               string           `json:"id,omitempty"`
	Name             string           `json:"name"`
	BaseURL          string           `json:"baseURL"`
	Provider         EndpointProvider `json:"provider"`
	CustomCARef      string           `json:"customCARef,omitempty"`
	DaemonProfileIDs []string         `json:"daemonProfileIDs,omitempty"`
}

type Theme string

const (
	ThemeSystem Theme = "system"
	ThemeLight  Theme = "light"
	ThemeDark   Theme = "dark"
)

func (t Theme) Valid() bool {
	switch t {
	case ThemeSystem, ThemeLight, ThemeDark:
		return true
	default:
		return false
	}
}

type UpdateChannel string

const (
	UpdateStable UpdateChannel = "stable"
	UpdateBeta   UpdateChannel = "beta"
)

func (c UpdateChannel) Valid() bool {
	return c == UpdateStable || c == UpdateBeta
}

type Language string

const (
	LanguageChinese Language = "zh-CN"
	LanguageEnglish Language = "en-US"
)

func (l Language) Valid() bool {
	return l == LanguageChinese || l == LanguageEnglish
}

type AppSettings struct {
	Theme                Theme         `json:"theme"`
	Language             Language      `json:"language"`
	CloseToTray          bool          `json:"closeToTray"`
	LaunchAtLogin        bool          `json:"launchAtLogin"`
	NotificationsEnabled bool          `json:"notificationsEnabled"`
	CheckForUpdates      bool          `json:"checkForUpdates"`
	UpdateChannel        UpdateChannel `json:"updateChannel"`
}

func DefaultAppSettings() AppSettings {
	return AppSettings{
		Theme:                ThemeSystem,
		Language:             LanguageChinese,
		CloseToTray:          true,
		NotificationsEnabled: true,
		CheckForUpdates:      true,
		UpdateChannel:        UpdateStable,
	}
}

func (s AppSettings) Valid() bool {
	return s.Theme.Valid() && s.Language.Valid() && s.UpdateChannel.Valid()
}
