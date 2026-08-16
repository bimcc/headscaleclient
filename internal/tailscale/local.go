package tailscale

import (
	"context"
	"net/netip"

	"tailscale.com/client/local"
	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
)

type localDaemon struct {
	client *local.Client
}

func newLocalDaemon() *localDaemon {
	return &localDaemon{client: &local.Client{}}
}

func (d *localDaemon) Status(ctx context.Context) (*ipnstate.Status, error) {
	return d.client.Status(ctx)
}

func (d *localDaemon) GetPrefs(ctx context.Context) (*ipn.Prefs, error) {
	return d.client.GetPrefs(ctx)
}

func (d *localDaemon) EditPrefs(ctx context.Context, prefs *ipn.MaskedPrefs) (*ipn.Prefs, error) {
	return d.client.EditPrefs(ctx, prefs)
}

func (d *localDaemon) ProfileStatus(ctx context.Context) (ipn.LoginProfile, []ipn.LoginProfile, error) {
	return d.client.ProfileStatus(ctx)
}

func (d *localDaemon) SwitchToEmptyProfile(ctx context.Context) error {
	return d.client.SwitchToEmptyProfile(ctx)
}

func (d *localDaemon) SwitchProfile(ctx context.Context, id ipn.ProfileID) error {
	return d.client.SwitchProfile(ctx, id)
}

func (d *localDaemon) Logout(ctx context.Context) error {
	return d.client.Logout(ctx)
}

func (d *localDaemon) Start(ctx context.Context, options ipn.Options) error {
	return d.client.Start(ctx, options)
}

func (d *localDaemon) StartLoginInteractive(ctx context.Context) error {
	return d.client.StartLoginInteractive(ctx)
}

func (d *localDaemon) Ping(ctx context.Context, address netip.Addr, pingType tailcfg.PingType) (*ipnstate.PingResult, error) {
	return d.client.Ping(ctx, address, pingType)
}

func (d *localDaemon) WatchIPNBus(ctx context.Context, mask ipn.NotifyWatchOpt) (bus, error) {
	return d.client.WatchIPNBus(ctx, mask)
}
