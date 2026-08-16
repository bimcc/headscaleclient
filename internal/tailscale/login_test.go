package tailscale

import (
	"context"
	"errors"
	"net/netip"
	"reflect"
	"testing"

	"github.com/headscaleclient/headscaleclient/internal/domain"
	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
)

type loginTestBus struct {
	notify ipn.Notify
}

func (b *loginTestBus) Next() (ipn.Notify, error) { return b.notify, nil }
func (b *loginTestBus) Close() error              { return nil }

type loginTestDaemon struct {
	calls     []string
	prefs     ipn.Prefs
	bus       bus
	logoutErr error
}

func (d *loginTestDaemon) Status(context.Context) (*ipnstate.Status, error) {
	return &ipnstate.Status{}, nil
}
func (d *loginTestDaemon) GetPrefs(context.Context) (*ipn.Prefs, error) {
	d.calls = append(d.calls, "prefs")
	return &d.prefs, nil
}
func (d *loginTestDaemon) EditPrefs(context.Context, *ipn.MaskedPrefs) (*ipn.Prefs, error) {
	return &d.prefs, nil
}
func (d *loginTestDaemon) ProfileStatus(context.Context) (ipn.LoginProfile, []ipn.LoginProfile, error) {
	return ipn.LoginProfile{}, nil, nil
}
func (d *loginTestDaemon) SwitchToEmptyProfile(context.Context) error {
	d.calls = append(d.calls, "empty-profile")
	return nil
}
func (d *loginTestDaemon) SwitchProfile(context.Context, ipn.ProfileID) error { return nil }
func (d *loginTestDaemon) Logout(context.Context) error {
	d.calls = append(d.calls, "logout")
	return d.logoutErr
}
func (d *loginTestDaemon) Start(_ context.Context, options ipn.Options) error {
	d.calls = append(d.calls, "start:"+options.UpdatePrefs.ControlURL)
	return nil
}
func (d *loginTestDaemon) StartLoginInteractive(context.Context) error {
	d.calls = append(d.calls, "login")
	return nil
}
func (d *loginTestDaemon) Ping(context.Context, netip.Addr, tailcfg.PingType) (*ipnstate.PingResult, error) {
	return &ipnstate.PingResult{}, nil
}
func (d *loginTestDaemon) WatchIPNBus(context.Context, ipn.NotifyWatchOpt) (bus, error) {
	d.calls = append(d.calls, "watch")
	return d.bus, nil
}

func TestBeginInteractiveLoginWatchesBeforeStarting(t *testing.T) {
	loginURL := "https://headscale.example/register/device"
	daemon := &loginTestDaemon{bus: &loginTestBus{notify: ipn.Notify{BrowseToURL: &loginURL}}}
	adapter := newAdapterWithDaemon(daemon)

	gotURL, err := adapter.BeginInteractiveLogin(context.Background(), "https://headscale.example")
	if err != nil {
		t.Fatal(err)
	}
	if gotURL != loginURL {
		t.Fatalf("login URL = %q, want %q", gotURL, loginURL)
	}
	wantCalls := []string{
		"empty-profile",
		"watch",
		"prefs",
		"start:https://headscale.example",
		"login",
	}
	if !reflect.DeepEqual(daemon.calls, wantCalls) {
		t.Fatalf("calls = %#v, want %#v", daemon.calls, wantCalls)
	}
}

func TestLogoutUsesDaemonAndClassifiesErrors(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		daemon := &loginTestDaemon{}
		adapter := newAdapterWithDaemon(daemon)

		if err := adapter.Logout(context.Background()); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(daemon.calls, []string{"logout"}) {
			t.Fatalf("calls = %#v", daemon.calls)
		}
	})

	t.Run("error", func(t *testing.T) {
		daemon := &loginTestDaemon{logoutErr: errors.New("connection refused")}
		adapter := newAdapterWithDaemon(daemon)

		err := adapter.Logout(context.Background())
		var appErr *domain.AppError
		if !errors.As(err, &appErr) || appErr.Problem.Code != domain.ErrorDaemonUnavailable {
			t.Fatalf("Logout() error = %#v", err)
		}
	})
}
