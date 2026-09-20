//go:build android

// Package engine binds one embedded Tailscale runtime to the Android host.
package engine

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/headscaleclient/headscaleclient/internal/application"
	"github.com/headscaleclient/headscaleclient/internal/config"
	"github.com/headscaleclient/headscaleclient/internal/domain"
	adapter "github.com/headscaleclient/headscaleclient/internal/tailscale"
	"github.com/headscaleclient/headscaleclient/mobile/bridge"
	"github.com/tailscale/tailscale-android/libtailscale"
	"tailscale.com/client/local"
)

type Events interface{ OnEvent(name, payload string) }

type Client struct {
	service *application.Service
	daemon  *vpnDaemon
	events  Events
}

var startOnce sync.Once
var singleton *Client
var startErr error

// Start is process-scoped. Activity recreation must reuse this same client.
func Start(dataDir string, appContext libtailscale.AppContext, events Events) (*Client, error) {
	startOnce.Do(func() {
		upstream := libtailscale.Start(dataDir, "", false, appContext)
		transport := newTransport(upstream)
		daemon := &vpnDaemon{Adapter: adapter.NewAdapterWithLocalClient(&local.Client{Transport: transport, OmitAuth: true})}
		store, err := config.NewStore(config.WithPath(filepath.Join(dataDir, "client.json")))
		if err != nil {
			startErr = err
			return
		}
		client := &Client{daemon: daemon, events: events}
		service, err := application.NewService(daemon, store,
			application.EventSinkFunc(func(name string, payload any) { client.emit(name, payload) }),
			application.WithDaemonLifecycle(embeddedLifecycle{}),
			application.WithDiagnostics("0.2.1-android.1", "native Android host", "embedded LocalAPI", "android"))
		if err != nil {
			startErr = err
			return
		}
		client.service = service
		singleton = client
		service.Start()
	})
	return singleton, startErr
}

func (c *Client) emit(name string, value any) {
	if c.events == nil {
		return
	}
	data, err := json.Marshal(value)
	if err == nil {
		c.events.OnEvent(name, string(data))
	}
}

// Request is invoked off the Android main thread. Errors are carried separately
// from successful results so a native failure never becomes demo data.
func (c *Client) Request(method, arguments string) string {
	value, err := bridge.Call(c.service, method, arguments)
	if err != nil {
		data, _ := json.Marshal(map[string]any{"error": err.Error()})
		return string(data)
	}
	data, err := json.Marshal(map[string]any{"result": value})
	if err != nil {
		return `{"error":"response encoding failed"}`
	}
	return string(data)
}

// SetVPNState records the Android TUN state separately from login/backend state.
func (c *Client) SetVPNState(desired, established bool) {
	var state uint32
	if desired {
		state |= 1
	}
	if established {
		state |= 2
	}
	c.daemon.vpnState.Store(state)
	c.emit("android:vpn-state-changed", map[string]any{})
}

type vpnDaemon struct {
	*adapter.Adapter
	vpnState atomic.Uint32
}

func (d *vpnDaemon) Snapshot(ctx context.Context) (domain.AppSnapshot, error) {
	snapshot, err := d.Adapter.Snapshot(ctx)
	if err != nil {
		return snapshot, err
	}
	state := d.vpnState.Load()
	snapshot.State.Connection = bridge.VPNConnectionState(snapshot.State.Connection, state&1 != 0, state&2 != 0)
	snapshot.HealthNotices = bridge.FilterVPNHealth(snapshot.HealthNotices, state&1 != 0, snapshot.Preferences.WantRunning)
	snapshot.DisplayState = domain.DeriveDisplayState(snapshot.State)
	return snapshot, nil
}

type embeddedLifecycle struct{}

func (embeddedLifecycle) Inspect(context.Context) (domain.EngineStatus, error) {
	return domain.EngineStatus{Ownership: domain.EngineOwnershipManaged, Service: domain.EngineServiceRunning, BundledVersion: "1.102.2"}, nil
}
func (embeddedLifecycle) EnsureInstalled(context.Context) error { return nil }
