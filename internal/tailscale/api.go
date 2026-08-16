package tailscale

import (
	"context"
	"net/netip"

	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
)

type bus interface {
	Next() (ipn.Notify, error)
	Close() error
}

type daemonAPI interface {
	Status(context.Context) (*ipnstate.Status, error)
	GetPrefs(context.Context) (*ipn.Prefs, error)
	EditPrefs(context.Context, *ipn.MaskedPrefs) (*ipn.Prefs, error)
	ProfileStatus(context.Context) (ipn.LoginProfile, []ipn.LoginProfile, error)
	SwitchToEmptyProfile(context.Context) error
	SwitchProfile(context.Context, ipn.ProfileID) error
	Logout(context.Context) error
	Start(context.Context, ipn.Options) error
	StartLoginInteractive(context.Context) error
	Ping(context.Context, netip.Addr, tailcfg.PingType) (*ipnstate.PingResult, error)
	WatchIPNBus(context.Context, ipn.NotifyWatchOpt) (bus, error)
}
