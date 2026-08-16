package desktop

const EventNavigate = "app:navigate"

type NavigationTarget string

const (
	NavigateOverview NavigationTarget = "overview"
	NavigateDevices  NavigationTarget = "devices"
	NavigateNetworks NavigationTarget = "networks"
	NavigateSettings NavigationTarget = "settings"
)

type NavigateEvent struct {
	View NavigationTarget `json:"view"`
}
