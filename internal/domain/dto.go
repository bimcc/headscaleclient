package domain

type CapabilityState string

const (
	CapabilityUnknown     CapabilityState = "unknown"
	CapabilitySupported   CapabilityState = "supported"
	CapabilityUnsupported CapabilityState = "unsupported"
)

func (s CapabilityState) Valid() bool {
	return s == CapabilityUnknown || s == CapabilitySupported || s == CapabilityUnsupported
}

type Capabilities struct {
	ExitNode     CapabilityState `json:"exitNode"`
	LANAccess    CapabilityState `json:"lanAccess"`
	DNS          CapabilityState `json:"dns"`
	AcceptRoutes CapabilityState `json:"acceptRoutes"`
	ShieldsUp    CapabilityState `json:"shieldsUp"`
	Ping         CapabilityState `json:"ping"`
	Taildrop     CapabilityState `json:"taildrop"`
	Serve        CapabilityState `json:"serve"`
	Funnel       CapabilityState `json:"funnel"`
	TailnetLock  CapabilityState `json:"tailnetLock"`
}

type EndpointSummary struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	BaseURL  string           `json:"baseURL"`
	Provider EndpointProvider `json:"provider"`
}

type ProfileSummary struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	LoginName  string `json:"loginName,omitempty"`
	EndpointID string `json:"endpointID,omitempty"`
	ControlURL string `json:"controlURL,omitempty"`
	Active     bool   `json:"active"`
}

type DeviceIdentity struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	DNSName   string   `json:"dnsName,omitempty"`
	User      string   `json:"user,omitempty"`
	OS        string   `json:"os,omitempty"`
	Addresses []string `json:"addresses"`
}

type PeerSummary struct {
	DeviceIdentity
	Online         bool               `json:"online"`
	LastSeen       string             `json:"lastSeen,omitempty"`
	ConnectionType PeerConnectionType `json:"connectionType"`
	RelayRegion    string             `json:"relayRegion,omitempty"`
	ExitNode       bool               `json:"exitNode"`
	ExitNodeOption bool               `json:"exitNodeOption"`
}

type PeerConnectionType string

const (
	PeerConnectionDirect  PeerConnectionType = "direct"
	PeerConnectionRelay   PeerConnectionType = "relay"
	PeerConnectionOffline PeerConnectionType = "offline"
	PeerConnectionUnknown PeerConnectionType = "unknown"
)

type Preferences struct {
	WantRunning            bool   `json:"wantRunning"`
	CorpDNS                bool   `json:"corpDNS"`
	AcceptRoutes           bool   `json:"acceptRoutes"`
	ShieldsUp              bool   `json:"shieldsUp"`
	ExitNodeID             string `json:"exitNodeID,omitempty"`
	ExitNodeAllowLANAccess bool   `json:"exitNodeAllowLANAccess"`
}

// PreferencePatch deliberately uses pointers so false and empty values remain
// distinguishable from omitted fields.
type PreferencePatch struct {
	WantRunning            *bool   `json:"wantRunning,omitempty"`
	CorpDNS                *bool   `json:"corpDNS,omitempty"`
	AcceptRoutes           *bool   `json:"acceptRoutes,omitempty"`
	ShieldsUp              *bool   `json:"shieldsUp,omitempty"`
	ExitNodeID             *string `json:"exitNodeID,omitempty"`
	ExitNodeAllowLANAccess *bool   `json:"exitNodeAllowLANAccess,omitempty"`
}

type AppSnapshot struct {
	Sequence       uint64           `json:"sequence"`
	State          StateAxes        `json:"state"`
	DisplayState   DisplayState     `json:"displayState"`
	HealthWarnings []string         `json:"healthWarnings"`
	DaemonVersion  string           `json:"daemonVersion,omitempty"`
	ActiveEndpoint *EndpointSummary `json:"activeEndpoint,omitempty"`
	ActiveProfile  *ProfileSummary  `json:"activeProfile,omitempty"`
	LocalDevice    *DeviceIdentity  `json:"localDevice,omitempty"`
	Peers          []PeerSummary    `json:"peers"`
	Preferences    Preferences      `json:"preferences"`
	Capabilities   Capabilities     `json:"capabilities"`
	RefreshedAt    string           `json:"refreshedAt"`
	Problem        *Problem         `json:"problem,omitempty"`
}

type ProfileCollection struct {
	Profiles []ProfileSummary `json:"profiles"`
	ActiveID string           `json:"activeID,omitempty"`
}

type LoginSession struct {
	ID         string `json:"id"`
	EndpointID string `json:"endpointID"`
	StartedAt  string `json:"startedAt"`
}

type SnapshotChangedEvent struct {
	Sequence uint64      `json:"sequence"`
	Snapshot AppSnapshot `json:"snapshot"`
}

type LoginURLEvent struct {
	Sequence  uint64 `json:"sequence"`
	SessionID string `json:"sessionID"`
	URL       string `json:"url"`
}

type LoginFinishedEvent struct {
	Sequence  uint64      `json:"sequence"`
	SessionID string      `json:"sessionID"`
	Snapshot  AppSnapshot `json:"snapshot"`
}

type OperationFailedEvent struct {
	Sequence  uint64  `json:"sequence"`
	Operation string  `json:"operation"`
	Problem   Problem `json:"problem"`
}
