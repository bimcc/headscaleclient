package domain

type EngineOwnership string

const (
	EngineOwnershipUnknown  EngineOwnership = "unknown"
	EngineOwnershipManaged  EngineOwnership = "managed"
	EngineOwnershipExternal EngineOwnership = "external"
	EngineOwnershipPrepared EngineOwnership = "prepared"
	EngineOwnershipMissing  EngineOwnership = "missing"
)

func (o EngineOwnership) Valid() bool {
	switch o {
	case EngineOwnershipUnknown, EngineOwnershipManaged, EngineOwnershipExternal, EngineOwnershipPrepared, EngineOwnershipMissing:
		return true
	default:
		return false
	}
}

type EngineServiceState string

const (
	EngineServiceUnknown  EngineServiceState = "unknown"
	EngineServiceMissing  EngineServiceState = "missing"
	EngineServiceStopped  EngineServiceState = "stopped"
	EngineServiceStarting EngineServiceState = "starting"
	EngineServiceRunning  EngineServiceState = "running"
	EngineServiceStopping EngineServiceState = "stopping"
)

func (s EngineServiceState) Valid() bool {
	switch s {
	case EngineServiceUnknown, EngineServiceMissing, EngineServiceStopped, EngineServiceStarting, EngineServiceRunning, EngineServiceStopping:
		return true
	default:
		return false
	}
}

// EngineStatus reports only product-safe daemon lifecycle information. Local
// executable paths and service credentials must never cross the UI boundary.
type EngineStatus struct {
	Ownership        EngineOwnership    `json:"ownership"`
	Service          EngineServiceState `json:"service"`
	BundledVersion   string             `json:"bundledVersion"`
	PayloadAvailable bool               `json:"payloadAvailable"`
	CanInstall       bool               `json:"canInstall"`
	CanStart         bool               `json:"canStart"`
}

func (s EngineStatus) Valid() bool {
	return s.Ownership.Valid() && s.Service.Valid()
}
