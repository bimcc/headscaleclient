package application

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/headscaleclient/headscaleclient/internal/domain"
	"github.com/headscaleclient/headscaleclient/internal/tailscale"
)

func TestGetSnapshotReturnsConfiguredOfflineState(t *testing.T) {
	t.Parallel()

	daemon := &fakeDaemon{
		snapshotErr: domain.NewError(domain.ErrorDaemonUnavailable, "The local service is not running.").WithRetryable(true),
	}
	store := newFakeStore()
	store.endpoints = []domain.ControlEndpoint{{
		ID: "endpoint-work", Name: "Work", BaseURL: "https://hs.example",
		Provider: domain.ProviderHeadscale,
	}}
	store.settings.LaunchAtLogin = true
	clock := time.Date(2026, 8, 15, 4, 5, 6, 0, time.UTC)
	service := mustService(t, daemon, store, nil,
		WithClock(func() time.Time { return clock }),
		WithDiagnostics("0.1.0", "3.0.0-beta.8", "localapi", "test/amd64"),
	)

	snapshot, err := service.GetSnapshot()
	if err != nil {
		t.Fatalf("GetSnapshot() error: %v", err)
	}
	if snapshot.Source != SnapshotSourceNative {
		t.Fatalf("source = %q, want native", snapshot.Source)
	}
	if snapshot.Runtime.Daemon != domain.DaemonStopped || snapshot.Runtime.Connection != domain.ConnectionStopped {
		t.Fatalf("unexpected offline runtime: %+v", snapshot.Runtime)
	}
	if snapshot.FallbackReason != "The local service is not running." {
		t.Fatalf("fallback reason = %q", snapshot.FallbackReason)
	}
	if len(snapshot.Endpoints) != 1 || snapshot.Endpoints[0].Kind != EndpointKindHeadscale {
		t.Fatalf("stored endpoints were not merged: %+v", snapshot.Endpoints)
	}
	if !snapshot.Settings.LaunchAtLogin || snapshot.Settings.Theme != domain.ThemeSystem {
		t.Fatalf("stored settings were not merged: %+v", snapshot.Settings)
	}
	if snapshot.Diagnostics.AppVersion != "0.1.0" || snapshot.Diagnostics.Platform != "test/amd64" {
		t.Fatalf("diagnostics were not merged: %+v", snapshot.Diagnostics)
	}
	if snapshot.UpdatedAt != "2026-08-15T04:05:06Z" {
		t.Fatalf("updatedAt = %q", snapshot.UpdatedAt)
	}
	if snapshot.Devices == nil || snapshot.Profiles == nil || snapshot.LocalDevice.Addresses == nil {
		t.Fatal("offline snapshot contains nil collections")
	}
}

func TestGetSnapshotIncludesDaemonLifecycleStatus(t *testing.T) {
	t.Parallel()

	lifecycle := &fakeDaemonLifecycle{status: domain.EngineStatus{
		Ownership:        domain.EngineOwnershipManaged,
		Service:          domain.EngineServiceRunning,
		BundledVersion:   "1.102.2",
		PayloadAvailable: true,
	}}
	service := mustService(t, &fakeDaemon{snapshot: healthyDomainSnapshot()}, newFakeStore(), nil, WithDaemonLifecycle(lifecycle))

	snapshot, err := service.GetSnapshot()
	if err != nil {
		t.Fatalf("GetSnapshot() error: %v", err)
	}
	if snapshot.Engine.Ownership != domain.EngineOwnershipManaged || snapshot.Engine.Service != domain.EngineServiceRunning {
		t.Fatalf("engine status = %+v", snapshot.Engine)
	}
	if snapshot.Engine.BundledVersion != "1.102.2" || !snapshot.Engine.PayloadAvailable {
		t.Fatalf("engine payload metadata = %+v", snapshot.Engine)
	}
}

func TestEnsureDaemonTransitionsPreparedPayloadToRunning(t *testing.T) {
	t.Parallel()

	lifecycle := &fakeDaemonLifecycle{status: domain.EngineStatus{
		Ownership:        domain.EngineOwnershipPrepared,
		Service:          domain.EngineServiceMissing,
		BundledVersion:   "1.102.2",
		PayloadAvailable: true,
		CanInstall:       true,
	}}
	service := mustService(t, &fakeDaemon{snapshot: healthyDomainSnapshot()}, newFakeStore(), nil, WithDaemonLifecycle(lifecycle))

	snapshot, err := service.EnsureDaemon()
	if err != nil {
		t.Fatalf("EnsureDaemon() error: %v", err)
	}
	if lifecycle.ensureCalls != 1 {
		t.Fatalf("EnsureInstalled() calls = %d, want 1", lifecycle.ensureCalls)
	}
	if snapshot.Engine.Ownership != domain.EngineOwnershipManaged || snapshot.Engine.Service != domain.EngineServiceRunning {
		t.Fatalf("engine status after install = %+v", snapshot.Engine)
	}
}

func TestEnsureDaemonWaitsForLocalAPIReadiness(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	daemon := &fakeDaemon{snapshotFn: func(context.Context) (domain.AppSnapshot, error) {
		if calls.Add(1) == 1 {
			return domain.AppSnapshot{}, domain.NewError(domain.ErrorDaemonUnavailable, "LocalAPI is not ready.").WithRetryable(true)
		}
		return healthyDomainSnapshot(), nil
	}}
	lifecycle := &fakeDaemonLifecycle{status: domain.EngineStatus{
		Ownership:        domain.EngineOwnershipManaged,
		Service:          domain.EngineServiceStopped,
		PayloadAvailable: true,
		CanStart:         true,
	}}
	service := mustService(t, daemon, newFakeStore(), nil, WithDaemonLifecycle(lifecycle), WithTimeouts(time.Second, time.Second, time.Second))

	snapshot, err := service.EnsureDaemon()
	if err != nil {
		t.Fatalf("EnsureDaemon() error: %v", err)
	}
	if calls.Load() < 2 {
		t.Fatalf("Snapshot() calls = %d, want at least 2", calls.Load())
	}
	if snapshot.Runtime.Daemon != domain.DaemonReady {
		t.Fatalf("runtime daemon = %q, want %q", snapshot.Runtime.Daemon, domain.DaemonReady)
	}
}

func TestGetSnapshotResolvesProfileEndpointsFromControlURLs(t *testing.T) {
	t.Parallel()

	raw := healthyDomainSnapshot()
	raw.ActiveEndpoint = &domain.EndpointSummary{BaseURL: "https://controlplane.tailscale.com"}
	daemon := &fakeDaemon{
		snapshot: raw,
		profiles: domain.ProfileCollection{
			ActiveID: "profile-work",
			Profiles: []domain.ProfileSummary{
				{ID: "profile-work", ControlURL: "https://hs.example", EndpointID: "endpoint-official", Active: true},
				{ID: "profile-official", ControlURL: "https://controlplane.tailscale.com"},
				{ID: "profile-cached"},
			},
		},
	}
	store := newFakeStore()
	store.endpoints = []domain.ControlEndpoint{
		{ID: "endpoint-official", BaseURL: "https://controlplane.tailscale.com", Provider: domain.ProviderTailscale, DaemonProfileIDs: []string{"profile-work"}, BuiltIn: true},
		{ID: "endpoint-work", BaseURL: "https://hs.example", Provider: domain.ProviderHeadscale},
		{ID: "endpoint-cached", BaseURL: "https://cached.example", Provider: domain.ProviderCompatible, DaemonProfileIDs: []string{"profile-cached"}},
	}
	service := mustService(t, daemon, store, nil)

	snapshot, err := service.GetSnapshot()
	if err != nil {
		t.Fatalf("GetSnapshot() error: %v", err)
	}
	if snapshot.ActiveEndpointID == nil || *snapshot.ActiveEndpointID != "endpoint-work" {
		t.Fatalf("active endpoint ID = %v, want endpoint-work", snapshot.ActiveEndpointID)
	}
	want := map[string]string{
		"profile-work":     "endpoint-work",
		"profile-official": "endpoint-official",
		"profile-cached":   "endpoint-cached",
	}
	for _, profile := range snapshot.Profiles {
		if profile.EndpointID != want[profile.ID] {
			t.Errorf("profile %q endpoint = %q, want %q", profile.ID, profile.EndpointID, want[profile.ID])
		}
		delete(want, profile.ID)
	}
	if len(want) != 0 {
		t.Fatalf("profiles missing from snapshot: %v", want)
	}
}

func TestGetSnapshotPreservesPeerConnectionMetadata(t *testing.T) {
	t.Parallel()

	raw := healthyDomainSnapshot()
	raw.Peers = []domain.PeerSummary{
		{DeviceIdentity: domain.DeviceIdentity{ID: "direct", Addresses: []string{}}, Online: true, ConnectionType: domain.PeerConnectionDirect, ExitNodeOption: true},
		{DeviceIdentity: domain.DeviceIdentity{ID: "relay", Addresses: []string{}}, Online: true, ConnectionType: domain.PeerConnectionRelay, RelayRegion: "hkg"},
		{DeviceIdentity: domain.DeviceIdentity{ID: "unknown", Addresses: []string{}}, Online: true, ConnectionType: domain.PeerConnectionUnknown},
		{DeviceIdentity: domain.DeviceIdentity{ID: "offline", Addresses: []string{}}, ConnectionType: domain.PeerConnectionOffline},
	}
	service := mustService(t, &fakeDaemon{snapshot: raw}, newFakeStore(), nil)

	snapshot, err := service.GetSnapshot()
	if err != nil {
		t.Fatalf("GetSnapshot() error: %v", err)
	}
	want := map[string]struct {
		connection ConnectionType
		relay      string
	}{
		"direct":  {connection: ConnectionTypeDirect},
		"relay":   {connection: ConnectionTypeRelay, relay: "hkg"},
		"unknown": {connection: ConnectionTypeUnknown},
		"offline": {connection: ConnectionTypeOffline},
	}
	for _, device := range snapshot.Devices {
		expected, ok := want[device.ID]
		if !ok {
			t.Fatalf("unexpected device %q", device.ID)
		}
		if device.ConnectionType != expected.connection || device.RelayRegion != expected.relay {
			t.Errorf("device %q route = (%q, %q), want (%q, %q)", device.ID, device.ConnectionType, device.RelayRegion, expected.connection, expected.relay)
		}
		if device.ID == "direct" && !device.ExitNodeOption {
			t.Error("eligible exit-node capability was not preserved")
		}
		delete(want, device.ID)
	}
	if len(want) != 0 {
		t.Fatalf("devices missing from snapshot: %v", want)
	}
}

func TestGetSnapshotPreservesDaemonHealthNotices(t *testing.T) {
	t.Parallel()

	raw := healthyDomainSnapshot()
	raw.HealthNotices = []domain.HealthNotice{{
		Code:     domain.HealthNoticeTailscaleWarning,
		Severity: domain.HealthNoticeWarning,
		Message:  "DNS configuration is unavailable.",
	}}
	service := mustService(t, &fakeDaemon{snapshot: raw}, newFakeStore(), nil)

	snapshot, err := service.GetSnapshot()
	if err != nil {
		t.Fatalf("GetSnapshot() error: %v", err)
	}
	if len(snapshot.HealthNotices) != 1 || snapshot.HealthNotices[0] != raw.HealthNotices[0] {
		t.Fatalf("health notices = %#v", snapshot.HealthNotices)
	}
}

func TestSetPreferenceUsesOptionalPatchAndPublishesSnapshot(t *testing.T) {
	t.Parallel()

	daemon := &fakeDaemon{snapshot: healthyDomainSnapshot()}
	store := newFakeStore()
	sink := &recordingSink{}
	service := mustService(t, daemon, store, sink)

	snapshot, err := service.SetPreference(PreferenceAcceptDNS, false)
	if err != nil {
		t.Fatalf("SetPreference() error: %v", err)
	}
	daemon.mu.Lock()
	patches := append([]domain.PreferencePatch(nil), daemon.patches...)
	daemon.mu.Unlock()
	if len(patches) != 1 || patches[0].CorpDNS == nil || *patches[0].CorpDNS {
		t.Fatalf("unexpected patch: %+v", patches)
	}
	if patches[0].AcceptRoutes != nil || patches[0].ShieldsUp != nil || patches[0].WantRunning != nil {
		t.Fatalf("unrequested fields were included in patch: %+v", patches[0])
	}
	if snapshot.Preferences.AcceptDNS {
		t.Fatal("refreshed snapshot did not contain the changed preference")
	}

	events := sink.snapshot()
	if len(events) != 1 || events[0].name != EventSnapshotChanged {
		t.Fatalf("events = %+v, want one snapshot event", events)
	}
	payload, ok := events[0].payload.(SnapshotChangedEvent)
	if !ok || payload.Sequence != 1 {
		t.Fatalf("snapshot event payload = %#v", events[0].payload)
	}
}

func TestSetExitNodeEnablesLANAccessByDefault(t *testing.T) {
	t.Parallel()

	daemon := &fakeDaemon{snapshot: healthyDomainSnapshot()}
	service := mustService(t, daemon, newFakeStore(), nil)
	exitNodeID := "exit-node-1"

	snapshot, err := service.SetExitNode(&exitNodeID)
	if err != nil {
		t.Fatalf("SetExitNode() error: %v", err)
	}
	daemon.mu.Lock()
	patch := daemon.patches[len(daemon.patches)-1]
	daemon.mu.Unlock()
	if patch.ExitNodeID == nil || *patch.ExitNodeID != exitNodeID {
		t.Fatalf("exit node patch = %+v", patch)
	}
	if patch.ExitNodeAllowLANAccess == nil || !*patch.ExitNodeAllowLANAccess {
		t.Fatalf("LAN access was not enabled: %+v", patch)
	}
	if snapshot.Preferences.ExitNodeID == nil || *snapshot.Preferences.ExitNodeID != exitNodeID || !snapshot.Preferences.AllowLANAccess {
		t.Fatalf("snapshot preferences = %+v", snapshot.Preferences)
	}
}

func TestPingDevicePublishesTheMeasuredPath(t *testing.T) {
	t.Parallel()

	raw := healthyDomainSnapshot()
	raw.Peers = []domain.PeerSummary{{
		DeviceIdentity: domain.DeviceIdentity{ID: "peer-1", Addresses: []string{"100.64.0.2"}},
		Online:         true,
		ConnectionType: domain.PeerConnectionRelay,
		RelayRegion:    "hkg",
	}}
	daemon := &fakeDaemon{
		snapshot: raw,
		pingResult: tailscale.PingResult{
			DeviceID:  "peer-1",
			LatencyMS: 24,
			Via:       tailscale.PingViaDirect,
			Endpoint:  "203.0.113.8:41641",
		},
	}
	sink := &recordingSink{}
	service := mustService(t, daemon, newFakeStore(), sink)

	result, err := service.PingDevice("peer-1")
	if err != nil {
		t.Fatalf("PingDevice() error: %v", err)
	}
	if result.Via != tailscale.PingViaDirect {
		t.Fatalf("result route = %q, want direct", result.Via)
	}

	events := sink.snapshot()
	if len(events) != 1 || events[0].name != EventSnapshotChanged {
		t.Fatalf("events = %+v, want one snapshot event", events)
	}
	payload, ok := events[0].payload.(SnapshotChangedEvent)
	if !ok || len(payload.Snapshot.Devices) != 1 {
		t.Fatalf("snapshot event payload = %#v", events[0].payload)
	}
	device := payload.Snapshot.Devices[0]
	if device.ConnectionType != ConnectionTypeDirect || device.RelayRegion != "" {
		t.Fatalf("refreshed path = (%q, %q), want direct", device.ConnectionType, device.RelayRegion)
	}
	if device.LatencyMS == nil || *device.LatencyMS != 24 {
		t.Fatalf("refreshed latency = %v, want 24", device.LatencyMS)
	}
}

func TestSetAppSettingRunsAutostartBeforePersistence(t *testing.T) {
	t.Parallel()

	daemon := &fakeDaemon{snapshot: healthyDomainSnapshot()}
	store := newFakeStore()
	var orderMu sync.Mutex
	var order []string
	store.beforeSaveSettings = func(domain.AppSettings) {
		orderMu.Lock()
		order = append(order, "store")
		orderMu.Unlock()
	}
	autostart := AutostartFunc(func(enabled bool) error {
		if !enabled {
			t.Fatal("autostart was not enabled")
		}
		orderMu.Lock()
		order = append(order, "os")
		orderMu.Unlock()
		return nil
	})
	service := mustService(t, daemon, store, nil, WithAutostart(autostart))

	snapshot, err := service.SetAppSetting(AppSettingLaunchAtLogin, true)
	if err != nil {
		t.Fatalf("SetAppSetting() error: %v", err)
	}
	orderMu.Lock()
	gotOrder := append([]string(nil), order...)
	orderMu.Unlock()
	if len(gotOrder) != 2 || gotOrder[0] != "os" || gotOrder[1] != "store" {
		t.Fatalf("operation order = %v, want [os store]", gotOrder)
	}
	if !snapshot.Settings.LaunchAtLogin {
		t.Fatal("saved launch-at-login setting was not returned")
	}
	store.mu.Lock()
	saved := store.settings
	store.mu.Unlock()
	if !saved.LaunchAtLogin {
		t.Fatal("launch-at-login setting was not persisted")
	}
}

func TestSnapshotUsesActualAutostartState(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	store.settings.LaunchAtLogin = true
	autostart := &statefulAutostart{enabled: false}
	service := mustService(t, &fakeDaemon{snapshot: healthyDomainSnapshot()}, store, nil, WithAutostart(autostart))

	snapshot, err := service.GetSnapshot()
	if err != nil {
		t.Fatalf("GetSnapshot() error: %v", err)
	}
	if snapshot.Settings.LaunchAtLogin {
		t.Fatal("snapshot used stale persisted autostart state")
	}
}

func TestAutostartRollbackUsesActualPreviousState(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	store.settings.LaunchAtLogin = true
	store.saveSettingsErr = domain.NewError(domain.ErrorConfigurationWriteFailed, "save failed")
	autostart := &statefulAutostart{enabled: false}
	service := mustService(t, &fakeDaemon{snapshot: healthyDomainSnapshot()}, store, nil, WithAutostart(autostart))

	if _, err := service.SetAppSetting(AppSettingLaunchAtLogin, true); err == nil {
		t.Fatal("SetAppSetting() succeeded despite persistence failure")
	}
	autostart.mu.Lock()
	enabled := autostart.enabled
	transitions := append([]bool(nil), autostart.transitions...)
	autostart.mu.Unlock()
	if enabled {
		t.Fatal("autostart was not restored to its actual previous state")
	}
	if len(transitions) != 2 || !transitions[0] || transitions[1] {
		t.Fatalf("autostart transitions = %v, want [true false]", transitions)
	}
}

func TestSetCloseToTrayPersistsSetting(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	service := mustService(t, &fakeDaemon{snapshot: healthyDomainSnapshot()}, store, nil)

	snapshot, err := service.SetAppSetting(AppSettingCloseToTray, false)
	if err != nil {
		t.Fatalf("SetAppSetting() error: %v", err)
	}
	if snapshot.Settings.CloseToTray {
		t.Fatal("snapshot still reports close-to-tray enabled")
	}
	store.mu.Lock()
	persisted := store.settings.CloseToTray
	store.mu.Unlock()
	if persisted {
		t.Fatal("close-to-tray setting was not persisted")
	}
}

func TestSetLanguagePersistsAndReturnsLocalizedPreference(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	service := mustService(t, &fakeDaemon{snapshot: healthyDomainSnapshot()}, store, nil)

	snapshot, err := service.SetLanguage(domain.LanguageEnglish)
	if err != nil {
		t.Fatalf("SetLanguage() error: %v", err)
	}
	if snapshot.Settings.Language != domain.LanguageEnglish {
		t.Fatalf("snapshot language = %q, want en-US", snapshot.Settings.Language)
	}
	store.mu.Lock()
	persisted := store.settings.Language
	store.mu.Unlock()
	if persisted != domain.LanguageEnglish {
		t.Fatalf("persisted language = %q, want en-US", persisted)
	}
}

func TestSaveEndpointUpdatesAndPreservesProfileAssociations(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	store.endpoints = []domain.ControlEndpoint{{
		ID: "11111111-1111-4111-8111-111111111111", Name: "Old", BaseURL: "https://old.example",
		Provider: domain.ProviderHeadscale, DaemonProfileIDs: []string{"profile-1"},
	}}
	service := mustService(t, &fakeDaemon{snapshot: healthyDomainSnapshot()}, store, nil)

	snapshot, err := service.SaveEndpoint(EndpointInput{
		ID: "11111111-1111-4111-8111-111111111111", Name: "New", URL: "https://new.example",
		Kind: EndpointKindCompatible,
	})
	if err != nil {
		t.Fatalf("SaveEndpoint() error: %v", err)
	}
	if len(snapshot.Endpoints) != 1 || snapshot.Endpoints[0].Name != "New" {
		t.Fatalf("updated endpoints = %+v", snapshot.Endpoints)
	}
	store.mu.Lock()
	profileIDs := append([]string(nil), store.endpoints[0].DaemonProfileIDs...)
	store.mu.Unlock()
	if len(profileIDs) != 1 || profileIDs[0] != "profile-1" {
		t.Fatalf("profile associations = %v", profileIDs)
	}
}

func TestDeleteEndpointRemovesCustomEndpoint(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	store.endpoints = []domain.ControlEndpoint{{
		ID: "11111111-1111-4111-8111-111111111111", Name: "Custom", BaseURL: "https://custom.example",
		Provider: domain.ProviderHeadscale,
	}}
	service := mustService(t, &fakeDaemon{snapshot: healthyDomainSnapshot()}, store, nil)

	snapshot, err := service.DeleteEndpoint("11111111-1111-4111-8111-111111111111")
	if err != nil {
		t.Fatalf("DeleteEndpoint() error: %v", err)
	}
	if len(snapshot.Endpoints) != 0 {
		t.Fatalf("endpoints after delete = %+v", snapshot.Endpoints)
	}
}

func TestMutationsAreSerialized(t *testing.T) {
	t.Parallel()

	daemon := &serialDaemon{
		fakeDaemon: &fakeDaemon{snapshot: healthyDomainSnapshot()},
		entered:    make(chan bool, 2),
		release:    make(chan struct{}, 2),
	}
	service := mustService(t, daemon, newFakeStore(), nil)
	done := make(chan error, 2)

	go func() {
		_, err := service.SetConnection(true)
		done <- err
	}()
	select {
	case enabled := <-daemon.entered:
		if !enabled {
			t.Fatal("first mutation did not enable the connection")
		}
	case <-time.After(time.Second):
		t.Fatal("first mutation did not start")
	}

	go func() {
		_, err := service.SetConnection(false)
		done <- err
	}()
	waitForMutationWaiter(t, service)
	select {
	case <-daemon.entered:
		t.Fatal("second mutation started before the first one completed")
	default:
	}

	daemon.release <- struct{}{}
	select {
	case enabled := <-daemon.entered:
		if enabled {
			t.Fatal("second mutation did not disable the connection")
		}
	case <-time.After(time.Second):
		t.Fatal("second mutation did not start after the first completed")
	}
	daemon.release <- struct{}{}
	for range 2 {
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("SetConnection() error: %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("mutation did not complete")
		}
	}
}

func TestWatcherPublishesMonotonicEventSequence(t *testing.T) {
	t.Parallel()

	daemon := &fakeDaemon{snapshot: healthyDomainSnapshot()}
	emitted := make(chan struct{})
	daemon.watchFn = func(ctx context.Context, handle func(tailscale.Event)) error {
		handle(tailscale.Event{Kind: tailscale.EventChanged})
		handle(tailscale.Event{Kind: tailscale.EventLoginURL, URL: "https://login.example/session"})
		handle(tailscale.Event{Kind: tailscale.EventLoginFinished})
		close(emitted)
		<-ctx.Done()
		return ctx.Err()
	}
	sink := &recordingSink{}
	service := mustService(t, daemon, newFakeStore(), sink)

	service.Start()
	select {
	case <-emitted:
	case <-time.After(2 * time.Second):
		service.Close()
		t.Fatal("watcher did not emit its events")
	}
	service.Close()

	events := sink.snapshot()
	if len(events) != 4 {
		t.Fatalf("event count = %d, want 4: %+v", len(events), events)
	}
	wantNames := []string{EventSnapshotChanged, EventSnapshotChanged, EventLoginURL, EventLoginFinished}
	for i, event := range events {
		if event.name != wantNames[i] {
			t.Fatalf("event %d name = %q, want %q", i, event.name, wantNames[i])
		}
		if sequenceOf(t, event.payload) != uint64(i+1) {
			t.Fatalf("event %d sequence = %d, want %d", i, sequenceOf(t, event.payload), i+1)
		}
	}
}

func TestWatcherReconnectsAfterStreamFailure(t *testing.T) {
	t.Parallel()

	daemon := &fakeDaemon{snapshot: healthyDomainSnapshot()}
	reconnected := make(chan struct{})
	daemon.watchFn = func(ctx context.Context, _ func(tailscale.Event)) error {
		daemon.mu.Lock()
		daemon.watchCalls++
		call := daemon.watchCalls
		daemon.mu.Unlock()
		if call < 3 {
			return domain.NewError(domain.ErrorDaemonUnavailable, "stream closed").WithRetryable(true)
		}
		if call == 3 {
			close(reconnected)
		}
		<-ctx.Done()
		return ctx.Err()
	}
	service := mustService(t, daemon, newFakeStore(), nil, WithWatchBackoff(time.Millisecond, 2*time.Millisecond))

	service.Start()
	select {
	case <-reconnected:
	case <-time.After(2 * time.Second):
		service.Close()
		t.Fatal("watcher did not reconnect")
	}
	service.Close()

	daemon.mu.Lock()
	calls := daemon.watchCalls
	daemon.mu.Unlock()
	if calls != 3 {
		t.Fatalf("Watch() calls = %d, want 3", calls)
	}
}

func TestBeginLoginReturnsFrontendContractAndURLEvent(t *testing.T) {
	t.Parallel()

	daemon := &fakeDaemon{snapshot: healthyDomainSnapshot(), beginURL: "https://login.example/auth"}
	store := newFakeStore()
	store.endpoints = []domain.ControlEndpoint{{
		ID: "endpoint-1", Name: "Headscale", BaseURL: "https://hs.example", Provider: domain.ProviderHeadscale,
	}}
	sink := &recordingSink{}
	service := mustService(t, daemon, store, sink)

	result, err := service.BeginLogin("endpoint-1")
	if err != nil {
		t.Fatalf("BeginLogin() error: %v", err)
	}
	if result.EndpointID != "endpoint-1" || result.AuthURL != daemon.beginURL {
		t.Fatalf("login result = %+v", result)
	}
	daemon.mu.Lock()
	controlURL := daemon.beginControlURL
	daemon.mu.Unlock()
	if controlURL != "https://hs.example" {
		t.Fatalf("control URL = %q", controlURL)
	}
	events := sink.snapshot()
	if len(events) != 1 || events[0].name != EventLoginURL {
		t.Fatalf("login events = %+v", events)
	}
	payload, ok := events[0].payload.(LoginURLEvent)
	if !ok || payload.SessionID == "" || payload.EndpointID != "endpoint-1" || payload.URL != daemon.beginURL {
		t.Fatalf("login URL event = %#v", events[0].payload)
	}
}

func TestBeginLoginReportsUnreachableEndpointBeforeChangingDaemon(t *testing.T) {
	t.Parallel()

	daemon := &fakeDaemon{snapshot: healthyDomainSnapshot(), beginURL: "https://login.example/auth"}
	store := newFakeStore()
	store.endpoints = []domain.ControlEndpoint{{
		ID: "endpoint-1", Name: "Headscale", BaseURL: "https://unavailable.example", Provider: domain.ProviderHeadscale,
	}}
	sink := &recordingSink{}
	service := mustService(t, daemon, store, sink, WithEndpointProbe(EndpointProbeFunc(func(_ context.Context, controlURL string) error {
		if controlURL != "https://unavailable.example" {
			t.Fatalf("probe URL = %q", controlURL)
		}
		return domain.NewError(domain.ErrorControlUnavailable, "控制服务器当前不可用。").WithRetryable(true)
	})))

	result, err := service.BeginLogin("endpoint-1")
	assertApplicationErrorCode(t, err, domain.ErrorControlUnavailable)
	if result != (LoginResult{}) {
		t.Fatalf("login result = %+v", result)
	}
	daemon.mu.Lock()
	beginControlURL := daemon.beginControlURL
	daemon.mu.Unlock()
	if beginControlURL != "" {
		t.Fatalf("daemon login started with %q", beginControlURL)
	}
	if pending := service.currentPendingLogin(); pending != nil {
		t.Fatalf("failed probe left pending login: %+v", pending)
	}
	events := sink.snapshot()
	if len(events) != 1 || events[0].name != EventOperationFailed {
		t.Fatalf("events = %+v", events)
	}
	payload := events[0].payload.(OperationFailedEvent)
	if payload.Operation != "BeginLogin" || payload.Problem.Code != domain.ErrorControlUnavailable {
		t.Fatalf("failure event = %+v", payload)
	}
}

func TestLogoutRefreshesSnapshotAfterRemovingActiveProfile(t *testing.T) {
	t.Parallel()

	daemon := &fakeDaemon{
		snapshot: healthyDomainSnapshot(),
		profiles: domain.ProfileCollection{
			ActiveID: "profile-1",
			Profiles: []domain.ProfileSummary{{
				ID: "profile-1", Name: "Team", LoginName: "user@example.com", ControlURL: "https://hs.example", Active: true,
			}},
		},
	}
	store := newFakeStore()
	store.endpoints = []domain.ControlEndpoint{{
		ID: "endpoint-1", Name: "Headscale", BaseURL: "https://hs.example", Provider: domain.ProviderHeadscale,
	}}
	service := mustService(t, daemon, store, &recordingSink{})

	snapshot, err := service.Logout()
	if err != nil {
		t.Fatalf("Logout() error: %v", err)
	}
	if !daemon.logoutCalled {
		t.Fatal("Logout() did not call the daemon")
	}
	if len(snapshot.Profiles) != 0 || snapshot.ActiveProfileID != nil {
		t.Fatalf("profiles after logout = %#v, active = %#v", snapshot.Profiles, snapshot.ActiveProfileID)
	}
}

func TestBeginLoginRejectsUnsafeURLWithoutBroadcast(t *testing.T) {
	t.Parallel()

	daemon := &fakeDaemon{snapshot: healthyDomainSnapshot(), beginURL: "file:///tmp/unsafe-login"}
	store := newFakeStore()
	store.endpoints = []domain.ControlEndpoint{{
		ID: "endpoint-1", Name: "Headscale", BaseURL: "https://hs.example", Provider: domain.ProviderHeadscale,
	}}
	sink := &recordingSink{}
	service := mustService(t, daemon, store, sink)

	result, err := service.BeginLogin("endpoint-1")
	assertApplicationErrorCode(t, err, domain.ErrorDaemonIncompatible)
	if result != (LoginResult{}) {
		t.Fatalf("unsafe login returned a result: %+v", result)
	}
	for _, event := range sink.snapshot() {
		if event.name == EventLoginURL {
			t.Fatalf("unsafe login URL was broadcast: %#v", event.payload)
		}
	}
	if pending := service.currentPendingLogin(); pending != nil {
		t.Fatalf("unsafe login left a pending session: %+v", pending)
	}
}

func TestWatcherRejectsUnsafeLoginURLWithoutBroadcast(t *testing.T) {
	t.Parallel()

	daemon := &fakeDaemon{snapshot: healthyDomainSnapshot()}
	handled := make(chan struct{})
	daemon.watchFn = func(ctx context.Context, handle func(tailscale.Event)) error {
		handle(tailscale.Event{Kind: tailscale.EventLoginURL, URL: "javascript:alert(1)"})
		close(handled)
		<-ctx.Done()
		return ctx.Err()
	}
	sink := &recordingSink{}
	service := mustService(t, daemon, newFakeStore(), sink)

	service.Start()
	select {
	case <-handled:
	case <-time.After(time.Second):
		t.Fatal("watcher did not handle the login URL")
	}
	service.Close()

	foundFailure := false
	for _, event := range sink.snapshot() {
		if event.name == EventLoginURL {
			t.Fatalf("unsafe watcher login URL was broadcast: %#v", event.payload)
		}
		if event.name == EventOperationFailed {
			payload := event.payload.(OperationFailedEvent)
			if payload.Operation == "LoginURL" && payload.Problem.Code == domain.ErrorDaemonIncompatible {
				foundFailure = true
			}
		}
	}
	if !foundFailure {
		t.Fatal("unsafe watcher URL did not produce a structured failure")
	}
}

func TestWatcherRefreshCannotPublishStaleStateAfterMutation(t *testing.T) {
	t.Parallel()

	base := &fakeDaemon{snapshot: healthyDomainSnapshot()}
	daemon := &interleavingDaemon{
		fakeDaemon:      base,
		firstSnapshot:   make(chan struct{}),
		releaseSnapshot: make(chan struct{}),
		mutationEntered: make(chan struct{}, 1),
	}
	sink := &recordingSink{}
	service := mustService(t, daemon, newFakeStore(), sink)

	refreshDone := make(chan struct{})
	go func() {
		service.refreshAndPublish(service.serviceCtx, "WatchRefresh")
		close(refreshDone)
	}()
	select {
	case <-daemon.firstSnapshot:
	case <-time.After(time.Second):
		t.Fatal("watcher refresh did not begin")
	}

	mutationAttempted := make(chan struct{})
	mutationDone := make(chan error, 1)
	go func() {
		close(mutationAttempted)
		_, err := service.SetConnection(false)
		mutationDone <- err
	}()
	<-mutationAttempted
	waitForMutationWaiter(t, service)
	select {
	case <-daemon.mutationEntered:
		t.Fatal("mutation entered the daemon while the older refresh was in flight")
	default:
	}

	close(daemon.releaseSnapshot)
	select {
	case <-refreshDone:
	case <-time.After(time.Second):
		t.Fatal("watcher refresh did not finish")
	}
	select {
	case <-daemon.mutationEntered:
	case <-time.After(time.Second):
		t.Fatal("mutation did not run after the watcher refresh")
	}
	select {
	case err := <-mutationDone:
		if err != nil {
			t.Fatalf("SetConnection() error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("mutation did not finish")
	}

	events := sink.snapshot()
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2: %+v", len(events), events)
	}
	first := events[0].payload.(SnapshotChangedEvent)
	second := events[1].payload.(SnapshotChangedEvent)
	if first.Sequence != 1 || second.Sequence != 2 {
		t.Fatalf("snapshot sequences = %d, %d", first.Sequence, second.Sequence)
	}
	if first.Snapshot.Runtime.Connection != domain.ConnectionRunning || second.Snapshot.Runtime.Connection != domain.ConnectionStopped {
		t.Fatalf("snapshot order is stale: first=%s second=%s", first.Snapshot.Runtime.Connection, second.Snapshot.Runtime.Connection)
	}
}

func TestBeginLoginRejectsConcurrentSession(t *testing.T) {
	t.Parallel()

	entered := make(chan struct{})
	release := make(chan struct{})
	daemon := &fakeDaemon{snapshot: healthyDomainSnapshot()}
	daemon.beginFn = func(ctx context.Context, _ string) (string, error) {
		close(entered)
		select {
		case <-release:
			return "https://login.example/auth", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	store := newFakeStore()
	store.endpoints = []domain.ControlEndpoint{{
		ID: "endpoint-1", Name: "Headscale", BaseURL: "https://hs.example", Provider: domain.ProviderHeadscale,
	}}
	service := mustService(t, daemon, store, nil)
	firstDone := make(chan error, 1)
	go func() {
		_, err := service.BeginLogin("endpoint-1")
		firstDone <- err
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("first login did not start")
	}

	_, err := service.BeginLogin("endpoint-1")
	assertApplicationErrorCode(t, err, domain.ErrorConflict)
	close(release)
	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf("first BeginLogin() error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("first login did not finish")
	}
}

func TestPendingLoginExpiresAndAllowsAnotherSession(t *testing.T) {
	t.Parallel()

	daemon := &fakeDaemon{snapshot: healthyDomainSnapshot(), beginURL: "https://login.example/auth"}
	store := newFakeStore()
	store.endpoints = []domain.ControlEndpoint{{
		ID: "endpoint-1", Name: "Headscale", BaseURL: "https://hs.example", Provider: domain.ProviderHeadscale,
	}}
	service := mustService(t, daemon, store, nil, WithPendingLoginTTL(10*time.Millisecond))

	if _, err := service.BeginLogin("endpoint-1"); err != nil {
		t.Fatalf("first BeginLogin() error: %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for service.currentPendingLogin() != nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if pending := service.currentPendingLogin(); pending != nil {
		t.Fatalf("pending login did not expire: %+v", pending)
	}
	if _, err := service.BeginLogin("endpoint-1"); err != nil {
		t.Fatalf("second BeginLogin() error after expiry: %v", err)
	}
}

func TestLoginFinishedDoesNotAssociateMismatchedControlEndpoint(t *testing.T) {
	t.Parallel()

	raw := healthyDomainSnapshot()
	raw.ActiveProfile = &domain.ProfileSummary{ID: "profile-1", Name: "User", Active: true}
	raw.ActiveEndpoint = &domain.EndpointSummary{BaseURL: "https://hs.example/tenant"}
	daemon := &fakeDaemon{
		snapshot: raw,
		profiles: domain.ProfileCollection{
			ActiveID: "profile-1",
			Profiles: []domain.ProfileSummary{{ID: "profile-1", Name: "User", Active: true}},
		},
		beginURL: "https://login.example/auth",
	}
	store := newFakeStore()
	store.endpoints = []domain.ControlEndpoint{{
		ID: "endpoint-1", Name: "Headscale", BaseURL: "https://hs.example/Tenant", Provider: domain.ProviderHeadscale,
	}}
	sink := &recordingSink{}
	service := mustService(t, daemon, store, sink)
	if _, err := service.BeginLogin("endpoint-1"); err != nil {
		t.Fatalf("BeginLogin() error: %v", err)
	}

	service.handleDaemonEvent(service.serviceCtx, tailscale.Event{Kind: tailscale.EventLoginFinished})
	store.mu.Lock()
	profileIDs := append([]string(nil), store.endpoints[0].DaemonProfileIDs...)
	store.mu.Unlock()
	if len(profileIDs) != 0 {
		t.Fatalf("mismatched endpoint was associated: %v", profileIDs)
	}
	foundConflict := false
	for _, event := range sink.snapshot() {
		if event.name == EventOperationFailed {
			payload := event.payload.(OperationFailedEvent)
			if payload.Operation == "FinishLogin" && payload.Problem.Code == domain.ErrorConflict {
				foundConflict = true
			}
		}
	}
	if !foundConflict {
		t.Fatal("mismatched control endpoint did not produce a conflict")
	}
	if pending := service.currentPendingLogin(); pending != nil {
		t.Fatalf("completed login remained pending: %+v", pending)
	}
}

func TestLoginFinishedAssociatesVerifiedProfileAndEndpoint(t *testing.T) {
	t.Parallel()

	raw := healthyDomainSnapshot()
	raw.ActiveProfile = &domain.ProfileSummary{
		ID: "profile-1", Name: "User", ControlURL: "https://hs.example", Active: true,
	}
	raw.ActiveEndpoint = &domain.EndpointSummary{BaseURL: "https://hs.example"}
	daemon := &fakeDaemon{
		snapshot: raw,
		profiles: domain.ProfileCollection{
			ActiveID: "profile-1",
			Profiles: []domain.ProfileSummary{{
				ID: "profile-1", Name: "User", ControlURL: "https://hs.example", Active: true,
			}},
		},
		beginURL: "https://login.example/auth",
	}
	store := newFakeStore()
	store.endpoints = []domain.ControlEndpoint{{
		ID: "endpoint-1", Name: "Headscale", BaseURL: "https://hs.example", Provider: domain.ProviderHeadscale,
	}}
	service := mustService(t, daemon, store, nil)
	if _, err := service.BeginLogin("endpoint-1"); err != nil {
		t.Fatalf("BeginLogin() error: %v", err)
	}

	service.handleDaemonEvent(service.serviceCtx, tailscale.Event{Kind: tailscale.EventLoginFinished})
	store.mu.Lock()
	profileIDs := append([]string(nil), store.endpoints[0].DaemonProfileIDs...)
	store.mu.Unlock()
	if len(profileIDs) != 1 || profileIDs[0] != "profile-1" {
		t.Fatalf("verified profile association = %v, want [profile-1]", profileIDs)
	}
}

func TestCloseCancelsActiveLoginAndMutationWaitingForGate(t *testing.T) {
	t.Parallel()

	loginEntered := make(chan struct{})
	daemon := &fakeDaemon{snapshot: healthyDomainSnapshot()}
	daemon.beginFn = func(ctx context.Context, _ string) (string, error) {
		close(loginEntered)
		<-ctx.Done()
		return "", ctx.Err()
	}
	store := newFakeStore()
	store.endpoints = []domain.ControlEndpoint{{
		ID: "endpoint-1", Name: "Headscale", BaseURL: "https://hs.example", Provider: domain.ProviderHeadscale,
	}}
	service := mustService(t, daemon, store, nil)
	loginDone := make(chan error, 1)
	go func() {
		_, err := service.BeginLogin("endpoint-1")
		loginDone <- err
	}()
	select {
	case <-loginEntered:
	case <-time.After(time.Second):
		t.Fatal("login did not start")
	}

	mutationDone := make(chan error, 1)
	go func() {
		_, err := service.SetConnection(false)
		mutationDone <- err
	}()
	waitForMutationWaiter(t, service)
	service.Close()
	for name, done := range map[string]<-chan error{"login": loginDone, "mutation": mutationDone} {
		select {
		case err := <-done:
			assertApplicationErrorCode(t, err, domain.ErrorCancelled)
		case <-time.After(time.Second):
			t.Fatalf("%s operation was not cancelled by Close", name)
		}
	}
}

func TestWatcherInitialEventThenEOFStillIncreasesBackoff(t *testing.T) {
	t.Parallel()

	daemon := &fakeDaemon{snapshot: healthyDomainSnapshot()}
	fakeNow := time.Unix(0, 0)
	daemon.watchFn = func(_ context.Context, handle func(tailscale.Event)) error {
		handle(tailscale.Event{Kind: tailscale.EventChanged})
		fakeNow = fakeNow.Add(time.Second)
		return domain.NewError(domain.ErrorDaemonUnavailable, "EOF").WithRetryable(true)
	}
	waitsRecorded := make(chan struct{})
	var waits []time.Duration
	testClock := func(service *Service) error {
		service.watchNow = func() time.Time { return fakeNow }
		service.waitFor = func(_ context.Context, duration time.Duration) bool {
			waits = append(waits, duration)
			if len(waits) == 3 {
				close(waitsRecorded)
				return false
			}
			return true
		}
		return nil
	}
	service := mustService(t, daemon, newFakeStore(), nil,
		WithWatchBackoff(time.Second, 8*time.Second),
		WithWatchStability(10*time.Second),
		testClock,
	)
	service.Start()
	select {
	case <-waitsRecorded:
	case <-time.After(time.Second):
		t.Fatal("watcher did not record backoff waits")
	}
	service.Close()
	want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}
	if len(waits) != len(want) {
		t.Fatalf("waits = %v, want %v", waits, want)
	}
	for i := range want {
		if waits[i] != want[i] {
			t.Fatalf("waits = %v, want %v", waits, want)
		}
	}
}

type fakeDaemon struct {
	mu sync.Mutex

	snapshot        domain.AppSnapshot
	snapshotErr     error
	snapshotFn      func(context.Context) (domain.AppSnapshot, error)
	profiles        domain.ProfileCollection
	profilesErr     error
	patches         []domain.PreferencePatch
	running         []bool
	beginURL        string
	beginErr        error
	beginFn         func(context.Context, string) (string, error)
	beginControlURL string
	logoutCalled    bool
	logoutErr       error
	pingResult      tailscale.PingResult
	pingErr         error
	watchFn         func(context.Context, func(tailscale.Event)) error
	watchCalls      int
}

type fakeDaemonLifecycle struct {
	status      domain.EngineStatus
	inspectErr  error
	ensureErr   error
	ensureCalls int
}

func (l *fakeDaemonLifecycle) Inspect(context.Context) (domain.EngineStatus, error) {
	return l.status, l.inspectErr
}

func (l *fakeDaemonLifecycle) EnsureInstalled(context.Context) error {
	l.ensureCalls++
	if l.ensureErr != nil {
		return l.ensureErr
	}
	l.status.Ownership = domain.EngineOwnershipManaged
	l.status.Service = domain.EngineServiceRunning
	l.status.CanInstall = false
	l.status.CanStart = false
	return nil
}

type serialDaemon struct {
	*fakeDaemon
	entered chan bool
	release chan struct{}
}

type interleavingDaemon struct {
	*fakeDaemon

	snapshotMu      sync.Mutex
	snapshotCalls   int
	firstSnapshot   chan struct{}
	releaseSnapshot chan struct{}
	mutationEntered chan struct{}
}

func (d *interleavingDaemon) Snapshot(ctx context.Context) (domain.AppSnapshot, error) {
	d.snapshotMu.Lock()
	d.snapshotCalls++
	call := d.snapshotCalls
	d.snapshotMu.Unlock()

	d.fakeDaemon.mu.Lock()
	snapshot := d.fakeDaemon.snapshot
	err := d.fakeDaemon.snapshotErr
	d.fakeDaemon.mu.Unlock()
	if call == 1 {
		close(d.firstSnapshot)
		select {
		case <-d.releaseSnapshot:
		case <-ctx.Done():
			return domain.AppSnapshot{}, ctx.Err()
		}
	}
	return snapshot, err
}

func (d *interleavingDaemon) SetRunning(ctx context.Context, enabled bool) (domain.Preferences, error) {
	select {
	case d.mutationEntered <- struct{}{}:
	default:
	}
	return d.fakeDaemon.SetRunning(ctx, enabled)
}

func (d *serialDaemon) SetRunning(ctx context.Context, enabled bool) (domain.Preferences, error) {
	select {
	case d.entered <- enabled:
	case <-ctx.Done():
		return domain.Preferences{}, ctx.Err()
	}
	select {
	case <-d.release:
	case <-ctx.Done():
		return domain.Preferences{}, ctx.Err()
	}
	return d.fakeDaemon.SetRunning(ctx, enabled)
}

func (d *fakeDaemon) Snapshot(ctx context.Context) (domain.AppSnapshot, error) {
	d.mu.Lock()
	snapshotFn := d.snapshotFn
	if snapshotFn != nil {
		d.mu.Unlock()
		return snapshotFn(ctx)
	}
	defer d.mu.Unlock()
	return d.snapshot, d.snapshotErr
}

func (d *fakeDaemon) SetRunning(_ context.Context, running bool) (domain.Preferences, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.running = append(d.running, running)
	d.snapshot.Preferences.WantRunning = running
	if running {
		d.snapshot.State.Connection = domain.ConnectionRunning
	} else {
		d.snapshot.State.Connection = domain.ConnectionStopped
	}
	return d.snapshot.Preferences, nil
}

func (d *fakeDaemon) PatchPreferences(_ context.Context, patch domain.PreferencePatch) (domain.Preferences, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.patches = append(d.patches, patch)
	if patch.WantRunning != nil {
		d.snapshot.Preferences.WantRunning = *patch.WantRunning
	}
	if patch.CorpDNS != nil {
		d.snapshot.Preferences.CorpDNS = *patch.CorpDNS
	}
	if patch.AcceptRoutes != nil {
		d.snapshot.Preferences.AcceptRoutes = *patch.AcceptRoutes
	}
	if patch.ShieldsUp != nil {
		d.snapshot.Preferences.ShieldsUp = *patch.ShieldsUp
	}
	if patch.ExitNodeID != nil {
		d.snapshot.Preferences.ExitNodeID = *patch.ExitNodeID
	}
	if patch.ExitNodeAllowLANAccess != nil {
		d.snapshot.Preferences.ExitNodeAllowLANAccess = *patch.ExitNodeAllowLANAccess
	}
	return d.snapshot.Preferences, nil
}

func (d *fakeDaemon) Profiles(context.Context) (domain.ProfileCollection, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.profiles, d.profilesErr
}

func (d *fakeDaemon) SwitchProfile(_ context.Context, profileID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.profiles.ActiveID = profileID
	for i := range d.profiles.Profiles {
		d.profiles.Profiles[i].Active = d.profiles.Profiles[i].ID == profileID
	}
	return nil
}

func (d *fakeDaemon) Logout(context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.logoutCalled = true
	if d.logoutErr != nil {
		return d.logoutErr
	}
	d.profiles.ActiveID = ""
	d.profiles.Profiles = nil
	d.snapshot.ActiveProfile = nil
	d.snapshot.State.Session = domain.SessionLoginRequired
	d.snapshot.State.Connection = domain.ConnectionStopped
	return nil
}

func (d *fakeDaemon) BeginInteractiveLogin(ctx context.Context, controlURL string) (string, error) {
	d.mu.Lock()
	d.beginControlURL = controlURL
	beginFn := d.beginFn
	if beginFn != nil {
		d.mu.Unlock()
		return beginFn(ctx, controlURL)
	}
	defer d.mu.Unlock()
	return d.beginURL, d.beginErr
}

func (d *fakeDaemon) PingDevice(context.Context, string) (tailscale.PingResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.pingResult, d.pingErr
}

func (d *fakeDaemon) Watch(ctx context.Context, handle func(tailscale.Event)) error {
	d.mu.Lock()
	watchFn := d.watchFn
	d.mu.Unlock()
	if watchFn == nil {
		<-ctx.Done()
		return ctx.Err()
	}
	return watchFn(ctx, handle)
}

type statefulAutostart struct {
	mu          sync.Mutex
	enabled     bool
	transitions []bool
}

func (a *statefulAutostart) Enable() error {
	a.setEnabled(true)
	return nil
}

func (a *statefulAutostart) Disable() error {
	a.setEnabled(false)
	return nil
}

func (a *statefulAutostart) IsEnabled() (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.enabled, nil
}

func (a *statefulAutostart) setEnabled(enabled bool) {
	a.mu.Lock()
	a.enabled = enabled
	a.transitions = append(a.transitions, enabled)
	a.mu.Unlock()
}

type fakeStore struct {
	mu sync.Mutex

	endpoints          []domain.ControlEndpoint
	settings           domain.AppSettings
	saveSettingsErr    error
	beforeSaveSettings func(domain.AppSettings)
}

func newFakeStore() *fakeStore {
	return &fakeStore{settings: domain.DefaultAppSettings()}
}

func (s *fakeStore) ListEndpoints(context.Context) ([]domain.ControlEndpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]domain.ControlEndpoint(nil), s.endpoints...), nil
}

func (s *fakeStore) SaveEndpoint(_ context.Context, input domain.ControlEndpointInput) (domain.ControlEndpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.endpoints {
		if input.ID != "" && s.endpoints[i].ID == input.ID {
			s.endpoints[i].Name = input.Name
			s.endpoints[i].BaseURL = input.BaseURL
			s.endpoints[i].Provider = input.Provider
			s.endpoints[i].CustomCARef = input.CustomCARef
			s.endpoints[i].DaemonProfileIDs = append([]string(nil), input.DaemonProfileIDs...)
			return s.endpoints[i], nil
		}
	}
	endpoint := domain.ControlEndpoint{
		ID: "created-endpoint", Name: input.Name, BaseURL: input.BaseURL,
		Provider: input.Provider, CustomCARef: input.CustomCARef,
		DaemonProfileIDs: append([]string(nil), input.DaemonProfileIDs...),
	}
	s.endpoints = append(s.endpoints, endpoint)
	return endpoint, nil
}

func (s *fakeStore) DeleteEndpoint(_ context.Context, endpointID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, endpoint := range s.endpoints {
		if endpoint.ID != endpointID {
			continue
		}
		if endpoint.BuiltIn {
			return domain.NewError(domain.ErrorPreconditionFailed, "built-in endpoint")
		}
		s.endpoints = append(s.endpoints[:i], s.endpoints[i+1:]...)
		return nil
	}
	return domain.NewError(domain.ErrorNotFound, "endpoint not found")
}

func (s *fakeStore) GetSettings(context.Context) (domain.AppSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.settings, nil
}

func (s *fakeStore) SaveSettings(_ context.Context, settings domain.AppSettings) (domain.AppSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.beforeSaveSettings != nil {
		s.beforeSaveSettings(settings)
	}
	if s.saveSettingsErr != nil {
		return domain.AppSettings{}, s.saveSettingsErr
	}
	s.settings = settings
	return settings, nil
}

type recordedEvent struct {
	name    string
	payload any
}

type recordingSink struct {
	mu     sync.Mutex
	events []recordedEvent
}

func (s *recordingSink) Emit(name string, payload any) {
	s.mu.Lock()
	s.events = append(s.events, recordedEvent{name: name, payload: payload})
	s.mu.Unlock()
}

func (s *recordingSink) snapshot() []recordedEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]recordedEvent(nil), s.events...)
}

func mustService(t *testing.T, daemon Daemon, store Store, sink EventSink, options ...Option) *Service {
	t.Helper()
	options = append([]Option{WithEndpointProbe(EndpointProbeFunc(func(context.Context, string) error {
		return nil
	}))}, options...)
	service, err := NewService(daemon, store, sink, options...)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	t.Cleanup(service.Close)
	return service
}

func healthyDomainSnapshot() domain.AppSnapshot {
	return domain.AppSnapshot{
		State: domain.StateAxes{
			Daemon: domain.DaemonReady, Session: domain.SessionAuthenticated,
			Connection: domain.ConnectionRunning, Control: domain.ControlReachable,
		},
		DaemonVersion: "1.102.2",
		LocalDevice: &domain.DeviceIdentity{
			ID: "self", Name: "laptop", DNSName: "laptop.example", OS: "test", Addresses: []string{"100.64.0.1"},
		},
		Peers:       []domain.PeerSummary{},
		Preferences: domain.Preferences{CorpDNS: true, AcceptRoutes: true},
		RefreshedAt: "2026-08-15T04:05:06Z",
	}
}

func sequenceOf(t *testing.T, payload any) uint64 {
	t.Helper()
	switch value := payload.(type) {
	case SnapshotChangedEvent:
		return value.Sequence
	case LoginURLEvent:
		return value.Sequence
	case LoginFinishedEvent:
		return value.Sequence
	case OperationFailedEvent:
		return value.Sequence
	default:
		t.Fatalf("unexpected event payload type %T", payload)
		return 0
	}
}

func assertApplicationErrorCode(t *testing.T, err error, want domain.ErrorCode) {
	t.Helper()
	var appError *domain.AppError
	if !errors.As(err, &appError) {
		t.Fatalf("error %T is not *domain.AppError: %v", err, err)
	}
	if appError.Problem.Code != want {
		t.Fatalf("error code = %q, want %q", appError.Problem.Code, want)
	}
}

func waitForMutationWaiter(t *testing.T, service *Service) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for service.mutationWaiters.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if service.mutationWaiters.Load() == 0 {
		t.Fatal("operation did not begin waiting for the mutation gate")
	}
}
