package application

import "github.com/headscaleclient/headscaleclient/internal/domain"

type SnapshotSource string

const (
	SnapshotSourceNative SnapshotSource = "native"
	SnapshotSourceDemo   SnapshotSource = "demo"
)

type ConnectionType string

const (
	ConnectionTypeDirect  ConnectionType = "direct"
	ConnectionTypeRelay   ConnectionType = "relay"
	ConnectionTypeOffline ConnectionType = "offline"
	ConnectionTypeUnknown ConnectionType = "unknown"
)

type EndpointKind string

const (
	EndpointKindHeadscale  EndpointKind = "headscale"
	EndpointKindTailscale  EndpointKind = "tailscale"
	EndpointKindCompatible EndpointKind = "compatible"
)

type EndpointStatus string

const (
	EndpointStatusReachable   EndpointStatus = "reachable"
	EndpointStatusUnreachable EndpointStatus = "unreachable"
	EndpointStatusUnchecked   EndpointStatus = "unchecked"
)

type ProfileState string

const (
	ProfileStateReady            ProfileState = "ready"
	ProfileStateLoginRequired    ProfileState = "login-required"
	ProfileStateApprovalRequired ProfileState = "approval-required"
)

type PreferenceKey string

const (
	PreferenceAcceptDNS      PreferenceKey = "acceptDns"
	PreferenceAcceptRoutes   PreferenceKey = "acceptRoutes"
	PreferenceAllowLANAccess PreferenceKey = "allowLanAccess"
	PreferenceShieldsUp      PreferenceKey = "shieldsUp"
)

type AppSettingKey string

const (
	AppSettingLaunchAtLogin AppSettingKey = "launchAtLogin"
	AppSettingCloseToTray   AppSettingKey = "closeToTray"
	AppSettingNotifications AppSettingKey = "notifications"
	AppSettingAutoUpdate    AppSettingKey = "autoUpdate"
)

type RuntimeState struct {
	Daemon     domain.DaemonState     `json:"daemon"`
	Session    domain.SessionState    `json:"session"`
	Connection domain.ConnectionState `json:"connection"`
	Control    domain.ControlState    `json:"control"`
}

type LocalDevice struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	DNSName        string         `json:"dnsName"`
	OS             string         `json:"os"`
	Addresses      []string       `json:"addresses"`
	ClientVersion  string         `json:"clientVersion"`
	ConnectionType ConnectionType `json:"connectionType"`
	RelayRegion    string         `json:"relayRegion,omitempty"`
}

type PeerDevice struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	DNSName        string         `json:"dnsName"`
	Owner          string         `json:"owner"`
	OS             string         `json:"os"`
	Addresses      []string       `json:"addresses"`
	Online         bool           `json:"online"`
	LastSeen       string         `json:"lastSeen"`
	LatencyMS      *int64         `json:"latencyMs,omitempty"`
	ConnectionType ConnectionType `json:"connectionType"`
	RelayRegion    string         `json:"relayRegion,omitempty"`
	ExitNodeOption bool           `json:"exitNodeOption"`
	Tags           []string       `json:"tags"`
}

type Endpoint struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	URL      string         `json:"url"`
	Kind     EndpointKind   `json:"kind"`
	Status   EndpointStatus `json:"status"`
	CustomCA bool           `json:"customCa"`
	BuiltIn  bool           `json:"builtIn"`
}

type LoginProfile struct {
	ID          string       `json:"id"`
	EndpointID  string       `json:"endpointId"`
	Account     string       `json:"account"`
	DisplayName string       `json:"displayName"`
	Active      bool         `json:"active"`
	State       ProfileState `json:"state"`
	LastUsedAt  string       `json:"lastUsedAt"`
}

type QuickPreferences struct {
	ExitNodeID     *string `json:"exitNodeId"`
	AcceptDNS      bool    `json:"acceptDns"`
	AcceptRoutes   bool    `json:"acceptRoutes"`
	AllowLANAccess bool    `json:"allowLanAccess"`
	ShieldsUp      bool    `json:"shieldsUp"`
}

type AppSettings struct {
	LaunchAtLogin bool            `json:"launchAtLogin"`
	CloseToTray   bool            `json:"closeToTray"`
	Notifications bool            `json:"notifications"`
	AutoUpdate    bool            `json:"autoUpdate"`
	Theme         domain.Theme    `json:"theme"`
	Language      domain.Language `json:"language"`
}

type Diagnostics struct {
	AppVersion    string `json:"appVersion"`
	WailsVersion  string `json:"wailsVersion"`
	DaemonVersion string `json:"daemonVersion"`
	LocalAPI      string `json:"localApi"`
	Platform      string `json:"platform"`
}

type EngineStatus struct {
	Ownership        domain.EngineOwnership    `json:"ownership"`
	Service          domain.EngineServiceState `json:"service"`
	BundledVersion   string                    `json:"bundledVersion"`
	PayloadAvailable bool                      `json:"payloadAvailable"`
	CanInstall       bool                      `json:"canInstall"`
	CanStart         bool                      `json:"canStart"`
}

// AppSnapshot deliberately mirrors frontend/src/lib/contracts.ts. Upstream
// Tailscale types must not cross this boundary.
type AppSnapshot struct {
	Source           SnapshotSource   `json:"source"`
	FallbackReason   string           `json:"fallbackReason,omitempty"`
	Runtime          RuntimeState     `json:"runtime"`
	LocalDevice      LocalDevice      `json:"localDevice"`
	Devices          []PeerDevice     `json:"devices"`
	Endpoints        []Endpoint       `json:"endpoints"`
	Profiles         []LoginProfile   `json:"profiles"`
	ActiveEndpointID *string          `json:"activeEndpointId"`
	ActiveProfileID  *string          `json:"activeProfileId"`
	Preferences      QuickPreferences `json:"preferences"`
	Settings         AppSettings      `json:"settings"`
	Diagnostics      Diagnostics      `json:"diagnostics"`
	Engine           EngineStatus     `json:"engine"`
	UpdatedAt        string           `json:"updatedAt"`
}

type EndpointInput struct {
	ID       string       `json:"id,omitempty"`
	Name     string       `json:"name"`
	URL      string       `json:"url"`
	Kind     EndpointKind `json:"kind"`
	CustomCA bool         `json:"customCa"`
}

type LoginResult struct {
	EndpointID string `json:"endpointId"`
	AuthURL    string `json:"authUrl"`
}

type SnapshotChangedEvent struct {
	Sequence uint64      `json:"sequence"`
	Snapshot AppSnapshot `json:"snapshot"`
}

type LoginURLEvent struct {
	Sequence   uint64 `json:"sequence"`
	SessionID  string `json:"sessionId"`
	EndpointID string `json:"endpointId,omitempty"`
	URL        string `json:"url"`
}

type LoginFinishedEvent struct {
	Sequence  uint64      `json:"sequence"`
	SessionID string      `json:"sessionId"`
	Snapshot  AppSnapshot `json:"snapshot"`
}

type OperationFailedEvent struct {
	Sequence  uint64         `json:"sequence"`
	Operation string         `json:"operation"`
	Problem   domain.Problem `json:"problem"`
}
