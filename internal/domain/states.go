package domain

// DaemonState describes whether the local tailscaled service can be used.
type DaemonState string

const (
	DaemonUnknown      DaemonState = "unknown"
	DaemonMissing      DaemonState = "missing"
	DaemonStopped      DaemonState = "stopped"
	DaemonReady        DaemonState = "ready"
	DaemonUnauthorized DaemonState = "unauthorized"
	DaemonIncompatible DaemonState = "incompatible"
)

func (s DaemonState) Valid() bool {
	switch s {
	case DaemonUnknown, DaemonMissing, DaemonStopped, DaemonReady, DaemonUnauthorized, DaemonIncompatible:
		return true
	default:
		return false
	}
}

// SessionState describes authentication independently from tunnel state.
type SessionState string

const (
	SessionNone             SessionState = "none"
	SessionLoginRequired    SessionState = "login-required"
	SessionApprovalRequired SessionState = "approval-required"
	SessionAuthenticated    SessionState = "authenticated"
)

func (s SessionState) Valid() bool {
	switch s {
	case SessionNone, SessionLoginRequired, SessionApprovalRequired, SessionAuthenticated:
		return true
	default:
		return false
	}
}

// ConnectionState describes the data-plane lifecycle.
type ConnectionState string

const (
	ConnectionStopped  ConnectionState = "stopped"
	ConnectionStarting ConnectionState = "starting"
	ConnectionRunning  ConnectionState = "running"
	ConnectionStopping ConnectionState = "stopping"
	ConnectionDegraded ConnectionState = "degraded"
)

func (s ConnectionState) Valid() bool {
	switch s {
	case ConnectionStopped, ConnectionStarting, ConnectionRunning, ConnectionStopping, ConnectionDegraded:
		return true
	default:
		return false
	}
}

// ControlState describes control-plane reachability. It does not imply that
// an already established peer tunnel is down.
type ControlState string

const (
	ControlUnknown     ControlState = "unknown"
	ControlReachable   ControlState = "reachable"
	ControlUnreachable ControlState = "unreachable"
)

func (s ControlState) Valid() bool {
	switch s {
	case ControlUnknown, ControlReachable, ControlUnreachable:
		return true
	default:
		return false
	}
}

type StateAxes struct {
	Daemon     DaemonState     `json:"daemon"`
	Session    SessionState    `json:"session"`
	Connection ConnectionState `json:"connection"`
	Control    ControlState    `json:"control"`
}

func (s StateAxes) Valid() bool {
	return s.Daemon.Valid() && s.Session.Valid() && s.Connection.Valid() && s.Control.Valid()
}

// DisplayState is a derived UI summary. Callers should retain StateAxes for
// diagnostics and must not use this value as the source of truth.
type DisplayState string

const (
	DisplayUnknown             DisplayState = "unknown"
	DisplayConnected           DisplayState = "connected"
	DisplayConnecting          DisplayState = "connecting"
	DisplayDisconnected        DisplayState = "disconnected"
	DisplayLoginRequired       DisplayState = "login-required"
	DisplayWaitingForApproval  DisplayState = "waiting-for-approval"
	DisplayServiceUnavailable  DisplayState = "service-unavailable"
	DisplayLimitedConnectivity DisplayState = "limited-connectivity"
)

func DeriveDisplayState(s StateAxes) DisplayState {
	if !s.Valid() || s.Daemon == DaemonUnknown {
		return DisplayUnknown
	}
	if s.Daemon != DaemonReady {
		return DisplayServiceUnavailable
	}
	if s.Session == SessionApprovalRequired {
		return DisplayWaitingForApproval
	}
	if s.Session == SessionNone || s.Session == SessionLoginRequired {
		return DisplayLoginRequired
	}

	switch s.Connection {
	case ConnectionStarting:
		return DisplayConnecting
	case ConnectionRunning:
		return DisplayConnected
	case ConnectionDegraded:
		return DisplayLimitedConnectivity
	case ConnectionStopped, ConnectionStopping:
		return DisplayDisconnected
	default:
		return DisplayUnknown
	}
}
