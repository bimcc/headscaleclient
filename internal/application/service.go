package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/headscaleclient/headscaleclient/internal/domain"
	"github.com/headscaleclient/headscaleclient/internal/tailscale"
)

const (
	EventSnapshotChanged = "app:snapshot-changed"
	EventLoginURL        = "app:login-url"
	EventLoginFinished   = "app:login-finished"
	EventOperationFailed = "app:operation-failed"

	DefaultQueryTimeout    = 5 * time.Second
	DefaultMutationTimeout = 45 * time.Second
	DefaultLoginTimeout    = 30 * time.Second
	DefaultPendingLoginTTL = 10 * time.Minute
	DefaultWatchStableTime = 30 * time.Second
)

type Daemon interface {
	Snapshot(context.Context) (domain.AppSnapshot, error)
	SetRunning(context.Context, bool) (domain.Preferences, error)
	PatchPreferences(context.Context, domain.PreferencePatch) (domain.Preferences, error)
	Profiles(context.Context) (domain.ProfileCollection, error)
	SwitchProfile(context.Context, string) error
	Logout(context.Context) error
	BeginInteractiveLogin(context.Context, string) (string, error)
	PingDevice(context.Context, string) (tailscale.PingResult, error)
	Watch(context.Context, func(tailscale.Event)) error
}

type Store interface {
	ListEndpoints(context.Context) ([]domain.ControlEndpoint, error)
	SaveEndpoint(context.Context, domain.ControlEndpointInput) (domain.ControlEndpoint, error)
	DeleteEndpoint(context.Context, string) error
	GetSettings(context.Context) (domain.AppSettings, error)
	SaveSettings(context.Context, domain.AppSettings) (domain.AppSettings, error)
}

type DaemonLifecycle interface {
	Inspect(context.Context) (domain.EngineStatus, error)
	EnsureInstalled(context.Context) error
}

type unavailableDaemonLifecycle struct{}

func (unavailableDaemonLifecycle) Inspect(context.Context) (domain.EngineStatus, error) {
	return domain.EngineStatus{
		Ownership: domain.EngineOwnershipUnknown,
		Service:   domain.EngineServiceUnknown,
	}, nil
}

func (unavailableDaemonLifecycle) EnsureInstalled(context.Context) error {
	return domain.NewError(domain.ErrorUnsupported, "Managed network-service installation is unavailable in this build.")
}

type EventSink interface {
	Emit(string, any)
}

type Autostart interface {
	Enable() error
	Disable() error
}

type AutostartStatus interface {
	IsEnabled() (bool, error)
}

type AutostartFunc func(bool) error

func (f AutostartFunc) Enable() error {
	if f == nil {
		return nil
	}
	return f(true)
}

func (f AutostartFunc) Disable() error {
	if f == nil {
		return nil
	}
	return f(false)
}

type EventSinkFunc func(string, any)

func (f EventSinkFunc) Emit(name string, payload any) {
	if f != nil {
		f(name, payload)
	}
}

type Option func(*Service) error

func WithTimeouts(query, mutation, login time.Duration) Option {
	return func(service *Service) error {
		if query <= 0 || mutation <= 0 || login <= 0 {
			return domain.NewError(domain.ErrorInvalidArgument, "Application timeouts must be positive.")
		}
		service.queryTimeout = query
		service.mutationTimeout = mutation
		service.loginTimeout = login
		return nil
	}
}

func WithWatchBackoff(initial, maximum time.Duration) Option {
	return func(service *Service) error {
		if initial <= 0 || maximum < initial {
			return domain.NewError(domain.ErrorInvalidArgument, "Watcher backoff values are invalid.")
		}
		service.watchBackoffInitial = initial
		service.watchBackoffMaximum = maximum
		return nil
	}
}

func WithWatchStability(duration time.Duration) Option {
	return func(service *Service) error {
		if duration <= 0 {
			return domain.NewError(domain.ErrorInvalidArgument, "Watcher stability duration must be positive.")
		}
		service.watchStableTime = duration
		return nil
	}
}

func WithPendingLoginTTL(duration time.Duration) Option {
	return func(service *Service) error {
		if duration <= 0 {
			return domain.NewError(domain.ErrorInvalidArgument, "Pending login lifetime must be positive.")
		}
		service.pendingLoginTTL = duration
		return nil
	}
}

func WithDiagnostics(appVersion, wailsVersion, localAPI, platform string) Option {
	return func(service *Service) error {
		if strings.TrimSpace(appVersion) != "" {
			service.appVersion = strings.TrimSpace(appVersion)
		}
		if strings.TrimSpace(wailsVersion) != "" {
			service.wailsVersion = strings.TrimSpace(wailsVersion)
		}
		if strings.TrimSpace(localAPI) != "" {
			service.localAPI = strings.TrimSpace(localAPI)
		}
		if strings.TrimSpace(platform) != "" {
			service.platform = strings.TrimSpace(platform)
		}
		return nil
	}
}

func WithClock(clock func() time.Time) Option {
	return func(service *Service) error {
		if clock == nil {
			return domain.NewError(domain.ErrorInvalidArgument, "Application clock cannot be nil.")
		}
		service.now = clock
		return nil
	}
}

func WithAutostart(autostart Autostart) Option {
	return func(service *Service) error {
		if autostart == nil {
			return domain.NewError(domain.ErrorInvalidArgument, "Autostart integration cannot be nil.")
		}
		service.autostart = autostart
		return nil
	}
}

func WithDaemonLifecycle(lifecycle DaemonLifecycle) Option {
	return func(service *Service) error {
		if lifecycle == nil {
			return domain.NewError(domain.ErrorInvalidArgument, "Daemon lifecycle integration cannot be nil.")
		}
		service.daemonLifecycle = lifecycle
		return nil
	}
}

func WithEndpointProbe(probe EndpointProbe) Option {
	return func(service *Service) error {
		if probe == nil {
			return domain.NewError(domain.ErrorInvalidArgument, "Endpoint probe cannot be nil.")
		}
		service.endpointProbe = probe
		return nil
	}
}

type pendingLogin struct {
	sessionID  string
	endpointID string
	lastURL    string
	expiresAt  time.Time
	timer      *time.Timer
}

type Service struct {
	daemon          Daemon
	daemonLifecycle DaemonLifecycle
	store           Store
	sink            EventSink
	autostart       Autostart
	endpointProbe   EndpointProbe

	queryTimeout    time.Duration
	mutationTimeout time.Duration
	loginTimeout    time.Duration

	watchBackoffInitial time.Duration
	watchBackoffMaximum time.Duration
	watchStableTime     time.Duration
	pendingLoginTTL     time.Duration

	appVersion   string
	wailsVersion string
	localAPI     string
	platform     string
	now          func() time.Time
	watchNow     func() time.Time
	pendingNow   func() time.Time
	waitFor      func(context.Context, time.Duration) bool

	mutationGate    chan struct{}
	mutationWaiters atomic.Int32
	eventMu         sync.Mutex
	sequence        uint64

	pendingMu sync.Mutex
	pending   *pendingLogin

	serviceCtx    context.Context
	cancelService context.CancelFunc
	closeOnce     sync.Once
	lifecycleMu   sync.Mutex
	watchDone     chan struct{}
}

func NewService(daemon Daemon, store Store, sink EventSink, options ...Option) (*Service, error) {
	if daemon == nil {
		return nil, domain.NewError(domain.ErrorInvalidArgument, "Daemon adapter cannot be nil.")
	}
	if store == nil {
		return nil, domain.NewError(domain.ErrorInvalidArgument, "Configuration store cannot be nil.")
	}
	if sink == nil {
		sink = EventSinkFunc(nil)
	}

	serviceCtx, cancelService := context.WithCancel(context.Background())
	service := &Service{
		daemon:              daemon,
		store:               store,
		sink:                sink,
		autostart:           AutostartFunc(nil),
		daemonLifecycle:     unavailableDaemonLifecycle{},
		endpointProbe:       newHTTPEndpointProbe(),
		queryTimeout:        DefaultQueryTimeout,
		mutationTimeout:     DefaultMutationTimeout,
		loginTimeout:        DefaultLoginTimeout,
		watchBackoffInitial: 250 * time.Millisecond,
		watchBackoffMaximum: 10 * time.Second,
		watchStableTime:     DefaultWatchStableTime,
		pendingLoginTTL:     DefaultPendingLoginTTL,
		appVersion:          "dev",
		wailsVersion:        "3.0.0-beta.8",
		localAPI:            "tailscaled LocalAPI",
		platform:            runtime.GOOS + "/" + runtime.GOARCH,
		now:                 time.Now,
		watchNow:            time.Now,
		pendingNow:          time.Now,
		waitFor:             waitForContext,
		mutationGate:        make(chan struct{}, 1),
		serviceCtx:          serviceCtx,
		cancelService:       cancelService,
	}
	for _, option := range options {
		if option == nil {
			cancelService()
			return nil, domain.NewError(domain.ErrorInvalidArgument, "Application option cannot be nil.")
		}
		if err := option(service); err != nil {
			cancelService()
			return nil, err
		}
	}
	return service, nil
}

func (s *Service) GetSnapshot() (AppSnapshot, error) {
	ctx, cancel := context.WithTimeout(s.serviceCtx, s.queryTimeout)
	defer cancel()
	snapshot, err := s.composeSnapshot(ctx)
	if err != nil {
		err = normalizeError(err, "The application state could not be loaded.")
		s.emitOperationFailed("GetSnapshot", err)
		return AppSnapshot{}, err
	}
	return snapshot, nil
}

func (s *Service) EnsureDaemon() (AppSnapshot, error) {
	return s.mutate("EnsureDaemon", func(ctx context.Context) error {
		return s.ensureDaemonReady(ctx)
	})
}

func (s *Service) SetConnection(enabled bool) (AppSnapshot, error) {
	return s.mutate("SetConnection", func(ctx context.Context) error {
		if enabled {
			if err := s.ensureDaemonAvailable(ctx); err != nil {
				return err
			}
		}
		_, err := s.daemon.SetRunning(ctx, enabled)
		return err
	})
}

func (s *Service) SetPreference(key PreferenceKey, value bool) (AppSnapshot, error) {
	return s.mutate("SetPreference", func(ctx context.Context) error {
		patch := domain.PreferencePatch{}
		switch key {
		case PreferenceAcceptDNS:
			patch.CorpDNS = &value
		case PreferenceAcceptRoutes:
			patch.AcceptRoutes = &value
		case PreferenceAllowLANAccess:
			patch.ExitNodeAllowLANAccess = &value
		case PreferenceShieldsUp:
			patch.ShieldsUp = &value
		default:
			return domain.NewError(domain.ErrorInvalidArgument, "Preference key is invalid.").WithDetail("key")
		}
		_, err := s.daemon.PatchPreferences(ctx, patch)
		return err
	})
}

func (s *Service) SetExitNode(deviceID *string) (AppSnapshot, error) {
	return s.mutate("SetExitNode", func(ctx context.Context) error {
		value := ""
		if deviceID != nil {
			value = strings.TrimSpace(*deviceID)
		}
		allowLANAccess := value != ""
		_, err := s.daemon.PatchPreferences(ctx, domain.PreferencePatch{
			ExitNodeID:             &value,
			ExitNodeAllowLANAccess: &allowLANAccess,
		})
		return err
	})
}

func (s *Service) PingDevice(deviceID string) (tailscale.PingResult, error) {
	ctx, cancel := context.WithTimeout(s.serviceCtx, s.queryTimeout)
	defer cancel()
	result, err := s.daemon.PingDevice(ctx, deviceID)
	if err != nil {
		err = normalizeError(err, "Could not ping the selected device.")
		s.emitOperationFailed("PingDevice", err)
		return tailscale.PingResult{}, err
	}
	s.refreshAndPublishWith(s.serviceCtx, "PingRefresh", func(snapshot *AppSnapshot) {
		applyPingResult(snapshot, result)
	})
	return result, nil
}

func applyPingResult(snapshot *AppSnapshot, result tailscale.PingResult) {
	for i := range snapshot.Devices {
		device := &snapshot.Devices[i]
		if device.ID != result.DeviceID {
			continue
		}
		latency := result.LatencyMS
		device.LatencyMS = &latency
		switch result.Via {
		case tailscale.PingViaDirect:
			device.ConnectionType = ConnectionTypeDirect
			device.RelayRegion = ""
		case tailscale.PingViaRelay:
			device.ConnectionType = ConnectionTypeRelay
			if result.RelayRegion != "" {
				device.RelayRegion = result.RelayRegion
			}
		}
		return
	}
}

func (s *Service) SaveEndpoint(input EndpointInput) (AppSnapshot, error) {
	return s.mutate("SaveEndpoint", func(ctx context.Context) error {
		if input.CustomCA {
			return domain.NewError(domain.ErrorUnsupported, "Custom CA selection is not available in this build.").WithDetail("customCa")
		}
		provider, err := providerFromKind(input.Kind)
		if err != nil {
			return err
		}
		profileIDs := []string{}
		if strings.TrimSpace(input.ID) != "" {
			existing, findErr := s.findEndpoint(ctx, input.ID)
			if findErr != nil {
				return findErr
			}
			profileIDs = nonNilStrings(existing.DaemonProfileIDs)
		}
		_, err = s.store.SaveEndpoint(ctx, domain.ControlEndpointInput{
			ID:               input.ID,
			Name:             input.Name,
			BaseURL:          input.URL,
			Provider:         provider,
			DaemonProfileIDs: profileIDs,
		})
		return err
	})
}

func (s *Service) DeleteEndpoint(endpointID string) (AppSnapshot, error) {
	return s.mutate("DeleteEndpoint", func(ctx context.Context) error {
		return s.store.DeleteEndpoint(ctx, strings.TrimSpace(endpointID))
	})
}

func (s *Service) SwitchProfile(profileID string) (AppSnapshot, error) {
	return s.mutate("SwitchProfile", func(ctx context.Context) error {
		return s.daemon.SwitchProfile(ctx, profileID)
	})
}

func (s *Service) Logout() (AppSnapshot, error) {
	return s.mutate("Logout", func(ctx context.Context) error {
		return s.daemon.Logout(ctx)
	})
}

func (s *Service) BeginLogin(endpointID string) (LoginResult, error) {
	ctx, cancel := context.WithTimeout(s.serviceCtx, s.loginTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		err = normalizeError(err, "Could not start login.")
		s.emitOperationFailed("BeginLogin", err)
		return LoginResult{}, err
	}

	pending, err := s.reservePendingLogin(endpointID)
	if err != nil {
		err = normalizeError(err, "Could not start login.")
		s.emitOperationFailed("BeginLogin", err)
		return LoginResult{}, err
	}
	if err := s.acquireMutation(ctx); err != nil {
		s.clearPendingLogin(pending.sessionID)
		err = normalizeError(err, "Could not start login.")
		s.emitOperationFailed("BeginLogin", err)
		return LoginResult{}, err
	}
	defer s.releaseMutation()
	if err := s.ensureDaemonAvailable(ctx); err != nil {
		s.clearPendingLogin(pending.sessionID)
		err = normalizeError(err, "Could not start the local network service.")
		s.emitOperationFailed("BeginLogin", err)
		return LoginResult{}, err
	}

	endpoint, err := s.findEndpoint(ctx, pending.endpointID)
	if err != nil {
		s.clearPendingLogin(pending.sessionID)
		err = normalizeError(err, "Could not start login.")
		s.emitOperationFailed("BeginLogin", err)
		return LoginResult{}, err
	}
	if endpoint.CustomCARef != "" {
		s.clearPendingLogin(pending.sessionID)
		err = domain.NewError(domain.ErrorUnsupported, "Custom CA login is not available in this build.")
		s.emitOperationFailed("BeginLogin", err)
		return LoginResult{}, err
	}
	if err := s.endpointProbe.Probe(ctx, endpoint.BaseURL); err != nil {
		s.clearPendingLogin(pending.sessionID)
		err = normalizeError(err, "无法连接控制服务器。")
		s.emitOperationFailed("BeginLogin", err)
		return LoginResult{}, err
	}

	authURL, err := s.daemon.BeginInteractiveLogin(ctx, endpoint.BaseURL)
	if ctxErr := ctx.Err(); ctxErr != nil {
		s.clearPendingLogin(pending.sessionID)
		if errors.Is(ctxErr, context.DeadlineExceeded) {
			err = domain.WrapError(
				domain.ErrorTimeout,
				"控制服务器可访问，但未在 30 秒内返回登录页面，请检查服务器认证配置后重试。",
				ctxErr,
			).WithRetryable(true)
		} else {
			err = normalizeError(ctxErr, "登录已取消。")
		}
		s.emitOperationFailed("BeginLogin", err)
		return LoginResult{}, err
	}
	if err != nil {
		s.clearPendingLogin(pending.sessionID)
		err = normalizeError(err, "Could not start login.")
		s.emitOperationFailed("BeginLogin", err)
		return LoginResult{}, err
	}
	validatedURL, err := tailscale.ValidateLoginURL(authURL)
	if err != nil {
		s.clearPendingLogin(pending.sessionID)
		err = normalizeError(err, "The local Tailscale service provided an invalid login URL.")
		s.emitOperationFailed("BeginLogin", err)
		return LoginResult{}, err
	}

	published, err := s.emitLoginURL(pending.sessionID, endpoint.ID, validatedURL)
	if err != nil || !published {
		s.clearPendingLogin(pending.sessionID)
		if err == nil {
			err = domain.NewError(domain.ErrorTimeout, "The login session expired before a login URL was received.").WithRetryable(true)
		}
		err = normalizeError(err, "Could not publish the login URL.")
		s.emitOperationFailed("BeginLogin", err)
		return LoginResult{}, err
	}
	return LoginResult{EndpointID: endpoint.ID, AuthURL: validatedURL}, nil
}

func (s *Service) SetAppSetting(key AppSettingKey, value bool) (AppSnapshot, error) {
	return s.mutate("SetAppSetting", func(ctx context.Context) error {
		settings, err := s.store.GetSettings(ctx)
		if err != nil {
			return err
		}
		switch key {
		case AppSettingLaunchAtLogin:
			previous := settings.LaunchAtLogin
			if status, ok := s.autostart.(AutostartStatus); ok {
				previous, err = status.IsEnabled()
				if err != nil {
					return domain.WrapError(domain.ErrorInternal, "Could not read launch-at-login integration.", err).WithRetryable(true)
				}
			}
			if err := setAutostart(s.autostart, value); err != nil {
				return domain.WrapError(domain.ErrorInternal, "Could not update launch-at-login integration.", err).WithRetryable(true)
			}
			settings.LaunchAtLogin = value
			if _, err = s.store.SaveSettings(ctx, settings); err != nil {
				_ = setAutostart(s.autostart, previous)
				return err
			}
			return nil
		case AppSettingNotifications:
			settings.NotificationsEnabled = value
		case AppSettingCloseToTray:
			settings.CloseToTray = value
		case AppSettingAutoUpdate:
			settings.CheckForUpdates = value
		default:
			return domain.NewError(domain.ErrorInvalidArgument, "Application setting key is invalid.").WithDetail("key")
		}
		_, err = s.store.SaveSettings(ctx, settings)
		return err
	})
}

func (s *Service) SetTheme(theme domain.Theme) (AppSnapshot, error) {
	return s.mutate("SetTheme", func(ctx context.Context) error {
		if !theme.Valid() {
			return domain.NewError(domain.ErrorInvalidArgument, "Theme is invalid.").WithDetail("theme")
		}
		settings, err := s.store.GetSettings(ctx)
		if err != nil {
			return err
		}
		settings.Theme = theme
		_, err = s.store.SaveSettings(ctx, settings)
		return err
	})
}

func (s *Service) SetLanguage(language domain.Language) (AppSnapshot, error) {
	return s.mutate("SetLanguage", func(ctx context.Context) error {
		if !language.Valid() {
			return domain.NewError(domain.ErrorInvalidArgument, "Language is invalid.").WithDetail("language")
		}
		settings, err := s.store.GetSettings(ctx)
		if err != nil {
			return err
		}
		settings.Language = language
		_, err = s.store.SaveSettings(ctx, settings)
		return err
	})
}

// Start begins the resilient daemon watcher. It is idempotent. A closed
// service cannot be restarted.
func (s *Service) Start() {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.watchDone != nil || s.serviceCtx.Err() != nil {
		return
	}
	s.watchDone = make(chan struct{})
	go func() {
		s.ensureManagedDaemonOnStartup()
		s.watchLoop(s.serviceCtx, s.watchDone)
	}()
}

func (s *Service) ensureManagedDaemonOnStartup() {
	ctx, cancel := context.WithTimeout(s.serviceCtx, s.queryTimeout)
	status, err := s.daemonLifecycle.Inspect(ctx)
	cancel()
	if err != nil || (status.Ownership != domain.EngineOwnershipManaged && status.Ownership != domain.EngineOwnershipPrepared) {
		return
	}
	_, _ = s.mutate("EnsureDaemonStartup", func(ctx context.Context) error {
		return s.ensureDaemonReady(ctx)
	})
}

func (s *Service) ensureDaemonReady(ctx context.Context) error {
	if err := s.daemonLifecycle.EnsureInstalled(ctx); err != nil {
		return err
	}
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	var lastErr error
	for {
		_, err := s.daemon.Snapshot(ctx)
		if err == nil {
			return nil
		}
		lastErr = err
		problem := domain.ProblemFromError(err)
		switch problem.Code {
		case domain.ErrorDaemonUnavailable, domain.ErrorDaemonStopped, domain.ErrorTimeout:
		default:
			return err
		}
		select {
		case <-ctx.Done():
			detail := "The LocalAPI endpoint did not become ready."
			if lastProblem := domain.ProblemFromError(lastErr); lastProblem != nil && lastProblem.Message != "" {
				detail = lastProblem.Message
			}
			return domain.WrapError(domain.ErrorTimeout, "The network service started, but LocalAPI did not become ready in time.", ctx.Err()).
				WithDetail(detail).WithRetryable(true)
		case <-ticker.C:
		}
	}
}

func (s *Service) ensureDaemonAvailable(ctx context.Context) error {
	if _, err := s.daemon.Snapshot(ctx); err == nil {
		return nil
	}
	return s.ensureDaemonReady(ctx)
}

func (s *Service) Close() {
	s.closeOnce.Do(func() {
		s.cancelService()
		s.clearAllPendingLogins()
	})

	s.lifecycleMu.Lock()
	done := s.watchDone
	s.lifecycleMu.Unlock()
	if done != nil {
		<-done
	}
}

func (s *Service) mutate(operation string, action func(context.Context) error) (AppSnapshot, error) {
	ctx, cancel := context.WithTimeout(s.serviceCtx, s.mutationTimeout)
	defer cancel()
	if err := s.acquireMutation(ctx); err != nil {
		err = normalizeError(err, "The requested change could not be applied.")
		s.emitOperationFailed(operation, err)
		return AppSnapshot{}, err
	}
	defer s.releaseMutation()
	if err := action(ctx); err != nil {
		err = normalizeError(err, "The requested change could not be applied.")
		s.emitOperationFailed(operation, err)
		return AppSnapshot{}, err
	}
	snapshot, err := s.composeSnapshot(ctx)
	if err != nil {
		err = normalizeError(err, "The application state could not be refreshed.")
		s.emitOperationFailed(operation, err)
		return AppSnapshot{}, err
	}
	s.emitSnapshotChanged(snapshot)
	return snapshot, nil
}

func (s *Service) composeSnapshot(ctx context.Context) (AppSnapshot, error) {
	endpoints, err := s.store.ListEndpoints(ctx)
	if err != nil {
		return AppSnapshot{}, normalizeError(err, "Could not load control servers.")
	}
	settings, err := s.store.GetSettings(ctx)
	if err != nil {
		return AppSnapshot{}, normalizeError(err, "Could not load application settings.")
	}
	if status, ok := s.autostart.(AutostartStatus); ok {
		if enabled, statusErr := status.IsEnabled(); statusErr == nil {
			settings.LaunchAtLogin = enabled
		}
	}
	engine, engineErr := s.daemonLifecycle.Inspect(ctx)
	if engineErr != nil || !engine.Valid() {
		engine = domain.EngineStatus{
			Ownership: domain.EngineOwnershipUnknown,
			Service:   domain.EngineServiceUnknown,
		}
	}

	raw, daemonErr := s.daemon.Snapshot(ctx)
	profiles := domain.ProfileCollection{Profiles: []domain.ProfileSummary{}}
	var profileErr error
	if daemonErr == nil {
		profiles, profileErr = s.daemon.Profiles(ctx)
		if profiles.Profiles == nil {
			profiles.Profiles = []domain.ProfileSummary{}
		}
	}

	return s.presentSnapshot(raw, endpoints, profiles, settings, engine, daemonErr, profileErr), nil
}

func (s *Service) presentSnapshot(
	raw domain.AppSnapshot,
	storedEndpoints []domain.ControlEndpoint,
	profileCollection domain.ProfileCollection,
	storedSettings domain.AppSettings,
	engine domain.EngineStatus,
	daemonErr error,
	profileErr error,
) AppSnapshot {
	state := raw.State
	fallbackReason := ""
	if daemonErr == nil && !state.Valid() {
		daemonErr = domain.NewError(domain.ErrorDaemonIncompatible, "The local Tailscale service returned an invalid state.").WithRetryable(true)
	}
	if daemonErr != nil {
		problem := domain.ProblemFromError(normalizeError(daemonErr, "The local Tailscale service is unavailable."))
		fallbackReason = problem.Message
		state = offlineState(problem.Code)
	} else if profileErr != nil {
		problem := domain.ProblemFromError(normalizeError(profileErr, "Could not load login profiles."))
		fallbackReason = problem.Message
	}
	switch engine.Service {
	case domain.EngineServiceMissing:
		state = offlineState(domain.ErrorDaemonMissing)
	case domain.EngineServiceStopped:
		state = offlineState(domain.ErrorDaemonStopped)
	}

	activeProfileID := profileCollection.ActiveID
	if activeProfileID == "" && raw.ActiveProfile != nil {
		activeProfileID = raw.ActiveProfile.ID
	}
	activeEndpointID := resolveActiveEndpointID(raw, activeProfileID, profileCollection.Profiles, storedEndpoints)

	endpoints := make([]Endpoint, 0, len(storedEndpoints))
	for _, endpoint := range storedEndpoints {
		status := EndpointStatusUnchecked
		if endpoint.ID == activeEndpointID {
			switch state.Control {
			case domain.ControlReachable:
				status = EndpointStatusReachable
			case domain.ControlUnreachable:
				status = EndpointStatusUnreachable
			}
		}
		endpoints = append(endpoints, Endpoint{
			ID:       endpoint.ID,
			Name:     endpoint.Name,
			URL:      endpoint.BaseURL,
			Kind:     kindFromProvider(endpoint.Provider),
			Status:   status,
			CustomCA: endpoint.CustomCARef != "",
			BuiltIn:  endpoint.BuiltIn,
		})
	}

	profiles := make([]LoginProfile, 0, len(profileCollection.Profiles))
	for _, profile := range profileCollection.Profiles {
		active := profile.Active || profile.ID == activeProfileID
		endpointID := resolveProfileEndpointID(profile, active, activeEndpointID, storedEndpoints)
		profileState := ProfileStateReady
		lastUsedAt := ""
		if active {
			lastUsedAt = raw.RefreshedAt
			switch state.Session {
			case domain.SessionApprovalRequired:
				profileState = ProfileStateApprovalRequired
			case domain.SessionNone, domain.SessionLoginRequired:
				profileState = ProfileStateLoginRequired
			}
		}
		profiles = append(profiles, LoginProfile{
			ID:          profile.ID,
			EndpointID:  endpointID,
			Account:     profile.LoginName,
			DisplayName: profile.Name,
			Active:      active,
			State:       profileState,
			LastUsedAt:  lastUsedAt,
		})
	}

	localDevice := LocalDevice{Addresses: []string{}, ConnectionType: ConnectionTypeOffline}
	if raw.LocalDevice != nil {
		localDevice.ID = raw.LocalDevice.ID
		localDevice.Name = raw.LocalDevice.Name
		localDevice.DNSName = raw.LocalDevice.DNSName
		localDevice.OS = raw.LocalDevice.OS
		localDevice.Addresses = nonNilStrings(raw.LocalDevice.Addresses)
		localDevice.ClientVersion = raw.DaemonVersion
		if state.Connection == domain.ConnectionRunning || state.Connection == domain.ConnectionDegraded {
			localDevice.ConnectionType = ConnectionTypeDirect
		}
	}

	devices := make([]PeerDevice, 0, len(raw.Peers))
	for _, peer := range raw.Peers {
		connectionType := ConnectionTypeUnknown
		switch peer.ConnectionType {
		case domain.PeerConnectionDirect:
			connectionType = ConnectionTypeDirect
		case domain.PeerConnectionRelay:
			connectionType = ConnectionTypeRelay
		case domain.PeerConnectionOffline:
			connectionType = ConnectionTypeOffline
		}
		devices = append(devices, PeerDevice{
			ID:             peer.ID,
			Name:           peer.Name,
			DNSName:        peer.DNSName,
			Owner:          peer.User,
			OS:             peer.OS,
			Addresses:      nonNilStrings(peer.Addresses),
			Online:         peer.Online,
			LastSeen:       peer.LastSeen,
			ConnectionType: connectionType,
			RelayRegion:    peer.RelayRegion,
			ExitNodeOption: peer.ExitNodeOption,
			Tags:           []string{},
		})
	}

	updatedAt := raw.RefreshedAt
	if updatedAt == "" {
		updatedAt = s.now().UTC().Format(time.RFC3339Nano)
	}
	var exitNodeID *string
	if raw.Preferences.ExitNodeID != "" {
		exitNodeID = stringPointer(raw.Preferences.ExitNodeID)
	}

	snapshot := AppSnapshot{
		Source:         SnapshotSourceNative,
		FallbackReason: fallbackReason,
		Runtime: RuntimeState{
			Daemon:     state.Daemon,
			Session:    state.Session,
			Connection: state.Connection,
			Control:    state.Control,
		},
		HealthWarnings: nonNilStrings(raw.HealthWarnings),
		LocalDevice:    localDevice,
		Devices:        devices,
		Endpoints:      endpoints,
		Profiles:       profiles,
		Preferences: QuickPreferences{
			ExitNodeID:     exitNodeID,
			AcceptDNS:      raw.Preferences.CorpDNS,
			AcceptRoutes:   raw.Preferences.AcceptRoutes,
			AllowLANAccess: raw.Preferences.ExitNodeAllowLANAccess,
			ShieldsUp:      raw.Preferences.ShieldsUp,
		},
		Settings: AppSettings{
			LaunchAtLogin: storedSettings.LaunchAtLogin,
			CloseToTray:   storedSettings.CloseToTray,
			Notifications: storedSettings.NotificationsEnabled,
			AutoUpdate:    storedSettings.CheckForUpdates,
			Theme:         storedSettings.Theme,
			Language:      storedSettings.Language,
		},
		Diagnostics: Diagnostics{
			AppVersion:    s.appVersion,
			WailsVersion:  s.wailsVersion,
			DaemonVersion: raw.DaemonVersion,
			LocalAPI:      s.localAPI,
			Platform:      s.platform,
		},
		Engine: EngineStatus{
			Ownership:        engine.Ownership,
			Service:          engine.Service,
			BundledVersion:   engine.BundledVersion,
			PayloadAvailable: engine.PayloadAvailable,
			CanInstall:       engine.CanInstall,
			CanStart:         engine.CanStart,
		},
		UpdatedAt: updatedAt,
	}
	if activeEndpointID != "" {
		snapshot.ActiveEndpointID = stringPointer(activeEndpointID)
	}
	if activeProfileID != "" {
		snapshot.ActiveProfileID = stringPointer(activeProfileID)
	}
	return snapshot
}

func (s *Service) watchLoop(ctx context.Context, done chan struct{}) {
	defer close(done)
	backoff := s.watchBackoffInitial
	for {
		if ctx.Err() != nil {
			return
		}
		s.refreshAndPublish(ctx, "WatchRefresh")

		startedAt := s.watchNow()
		err := s.daemon.Watch(ctx, func(event tailscale.Event) {
			s.handleDaemonEvent(ctx, event)
		})
		lifetime := s.watchNow().Sub(startedAt)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			s.emitOperationFailed("Watch", normalizeError(err, "The daemon event stream was interrupted."))
		}
		if lifetime >= s.watchStableTime {
			backoff = s.watchBackoffInitial
		}
		if !s.waitFor(ctx, backoff) {
			return
		}
		if lifetime < s.watchStableTime {
			backoff *= 2
			if backoff > s.watchBackoffMaximum {
				backoff = s.watchBackoffMaximum
			}
		}
	}
}

func (s *Service) handleDaemonEvent(ctx context.Context, event tailscale.Event) {
	switch event.Kind {
	case tailscale.EventChanged:
		s.refreshAndPublish(ctx, "WatchRefresh")
	case tailscale.EventLoginURL:
		pending := s.currentPendingLogin()
		sessionID := ""
		endpointID := ""
		if pending != nil {
			sessionID = pending.sessionID
			endpointID = pending.endpointID
		}
		published, err := s.emitLoginURL(sessionID, endpointID, event.URL)
		if err != nil {
			if pending != nil {
				s.clearPendingLogin(pending.sessionID)
			}
			s.emitOperationFailed("LoginURL", normalizeError(err, "The local Tailscale service provided an invalid login URL."))
			return
		}
		if !published && pending != nil {
			s.emitOperationFailed("LoginURL", domain.NewError(domain.ErrorTimeout, "The login session expired before its URL could be published.").WithRetryable(true))
		}
	case tailscale.EventLoginFinished:
		s.handleLoginFinished(ctx)
	case tailscale.EventError:
		if event.Err != nil {
			s.emitOperationFailed("Watch", normalizeError(event.Err, "The local Tailscale service reported an error."))
		}
	}
}

func (s *Service) refreshAndPublish(parent context.Context, operation string) {
	s.refreshAndPublishWith(parent, operation, nil)
}

func (s *Service) refreshAndPublishWith(parent context.Context, operation string, transform func(*AppSnapshot)) {
	ctx, cancel := context.WithTimeout(parent, s.queryTimeout)
	defer cancel()
	if err := s.acquireMutation(ctx); err != nil {
		if s.serviceCtx.Err() == nil {
			s.emitOperationFailed(operation, normalizeError(err, "The application state could not be refreshed."))
		}
		return
	}
	defer s.releaseMutation()
	snapshot, err := s.composeSnapshot(ctx)
	if err != nil {
		s.emitOperationFailed(operation, normalizeError(err, "The application state could not be refreshed."))
		return
	}
	if transform != nil {
		transform(&snapshot)
		snapshot.UpdatedAt = s.now().UTC().Format(time.RFC3339Nano)
	}
	s.emitSnapshotChanged(snapshot)
}

func (s *Service) handleLoginFinished(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, s.mutationTimeout)
	defer cancel()
	if err := s.acquireMutation(ctx); err != nil {
		if s.serviceCtx.Err() == nil {
			s.emitOperationFailed("FinishLogin", normalizeError(err, "Could not finish login."))
		}
		return
	}
	defer s.releaseMutation()

	pending := s.takePendingLoginForFinish()
	if pending != nil {
		if err := s.associateActiveProfile(ctx, *pending); err != nil {
			s.emitOperationFailed("FinishLogin", normalizeError(err, "Could not associate the new profile with its control server."))
		}
	}
	snapshot, err := s.composeSnapshot(ctx)
	if err != nil {
		s.emitOperationFailed("FinishLogin", normalizeError(err, "Could not refresh state after login."))
		return
	}
	sessionID := ""
	if pending != nil {
		sessionID = pending.sessionID
	}
	s.emitLoginFinished(sessionID, snapshot)
}

func (s *Service) associateActiveProfile(ctx context.Context, pending pendingLogin) error {
	profiles, err := s.daemon.Profiles(ctx)
	if err != nil {
		return err
	}
	raw, err := s.daemon.Snapshot(ctx)
	if err != nil {
		return err
	}
	activeProfileID := profiles.ActiveID
	if activeProfileID == "" && raw.ActiveProfile != nil {
		activeProfileID = raw.ActiveProfile.ID
	}
	if activeProfileID == "" {
		return domain.NewError(domain.ErrorPreconditionFailed, "The completed login has no active profile.")
	}
	if raw.ActiveProfile != nil && raw.ActiveProfile.ID != activeProfileID {
		return domain.NewError(domain.ErrorConflict, "The active profile changed while login was completing.")
	}
	endpoints, err := s.store.ListEndpoints(ctx)
	if err != nil {
		return err
	}
	for _, endpoint := range endpoints {
		if endpoint.ID != pending.endpointID {
			continue
		}
		if raw.ActiveEndpoint == nil || !sameControlURL(raw.ActiveEndpoint.BaseURL, endpoint.BaseURL) {
			return domain.NewError(domain.ErrorConflict, "The active control server does not match the completed login.")
		}
		for _, profile := range profiles.Profiles {
			if profile.ID != activeProfileID {
				continue
			}
			if profile.ControlURL != "" && !sameControlURL(profile.ControlURL, endpoint.BaseURL) {
				return domain.NewError(domain.ErrorConflict, "The active profile belongs to another control server.")
			}
			if profile.EndpointID != "" && profile.EndpointID != endpoint.ID {
				return domain.NewError(domain.ErrorConflict, "The active profile is already associated with another control server.")
			}
		}
		if endpoint.BuiltIn || containsString(endpoint.DaemonProfileIDs, activeProfileID) {
			return nil
		}
		profileIDs := append(nonNilStrings(endpoint.DaemonProfileIDs), activeProfileID)
		_, err = s.store.SaveEndpoint(ctx, domain.ControlEndpointInput{
			ID:               endpoint.ID,
			Name:             endpoint.Name,
			BaseURL:          endpoint.BaseURL,
			Provider:         endpoint.Provider,
			CustomCARef:      endpoint.CustomCARef,
			DaemonProfileIDs: profileIDs,
		})
		return err
	}
	return domain.NewError(domain.ErrorNotFound, "The login control server was not found.")
}

func (s *Service) findEndpoint(ctx context.Context, endpointID string) (domain.ControlEndpoint, error) {
	endpointID = strings.TrimSpace(endpointID)
	if endpointID == "" {
		return domain.ControlEndpoint{}, domain.NewError(domain.ErrorInvalidArgument, "An endpoint ID is required.").WithDetail("endpointId")
	}
	endpoints, err := s.store.ListEndpoints(ctx)
	if err != nil {
		return domain.ControlEndpoint{}, err
	}
	for _, endpoint := range endpoints {
		if endpoint.ID == endpointID {
			return endpoint, nil
		}
	}
	return domain.ControlEndpoint{}, domain.NewError(domain.ErrorNotFound, "Control server was not found.")
}

func (s *Service) reservePendingLogin(endpointID string) (*pendingLogin, error) {
	endpointID = strings.TrimSpace(endpointID)
	if endpointID == "" {
		return nil, domain.NewError(domain.ErrorInvalidArgument, "An endpoint ID is required.").WithDetail("endpointId")
	}
	sessionID, err := newSessionID()
	if err != nil {
		return nil, domain.WrapError(domain.ErrorInternal, "Could not create a login session.", err)
	}

	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	now := s.pendingNow()
	s.expirePendingLoginLocked(now)
	if s.pending != nil {
		return nil, domain.NewError(domain.ErrorConflict, "Another login is already in progress.").WithRetryable(true)
	}
	pending := &pendingLogin{
		sessionID:  sessionID,
		endpointID: endpointID,
		expiresAt:  now.Add(s.loginTimeout),
	}
	pending.timer = time.AfterFunc(s.loginTimeout, func() {
		s.expirePendingLogin(sessionID)
	})
	s.pending = pending
	copy := *pending
	return &copy, nil
}

func (s *Service) currentPendingLogin() *pendingLogin {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	s.expirePendingLoginLocked(s.pendingNow())
	if s.pending == nil {
		return nil
	}
	copy := *s.pending
	return &copy
}

func (s *Service) takePendingLoginForFinish() *pendingLogin {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	s.expirePendingLoginLocked(s.pendingNow())
	if s.pending == nil || s.pending.lastURL == "" {
		return nil
	}
	pending := s.pending
	s.pending = nil
	if pending.timer != nil {
		pending.timer.Stop()
	}
	copy := *pending
	return &copy
}

func (s *Service) clearPendingLogin(sessionID string) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	if s.pending != nil && s.pending.sessionID == sessionID {
		s.clearPendingLoginLocked()
	}
}

func (s *Service) clearAllPendingLogins() {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	s.clearPendingLoginLocked()
}

func (s *Service) expirePendingLogin(sessionID string) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	if s.pending != nil && s.pending.sessionID == sessionID {
		s.expirePendingLoginLocked(s.pendingNow())
	}
}

func (s *Service) expirePendingLoginLocked(now time.Time) {
	if s.pending != nil && !now.Before(s.pending.expiresAt) {
		s.clearPendingLoginLocked()
	}
}

func (s *Service) clearPendingLoginLocked() {
	if s.pending != nil && s.pending.timer != nil {
		s.pending.timer.Stop()
	}
	s.pending = nil
}

func (s *Service) emitSnapshotChanged(snapshot AppSnapshot) {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	s.sequence++
	s.sink.Emit(EventSnapshotChanged, SnapshotChangedEvent{Sequence: s.sequence, Snapshot: snapshot})
}

func (s *Service) emitLoginURL(sessionID, endpointID, rawURL string) (bool, error) {
	url, err := tailscale.ValidateLoginURL(rawURL)
	if err != nil {
		return false, err
	}
	if sessionID != "" {
		s.pendingMu.Lock()
		s.expirePendingLoginLocked(s.pendingNow())
		if s.pending == nil || s.pending.sessionID != sessionID {
			s.pendingMu.Unlock()
			return false, nil
		}
		if s.pending.lastURL == url {
			s.pendingMu.Unlock()
			return true, nil
		}
		s.pending.lastURL = url
		s.pending.expiresAt = s.pendingNow().Add(s.pendingLoginTTL)
		if s.pending.timer != nil {
			s.pending.timer.Stop()
		}
		sessionIDCopy := s.pending.sessionID
		s.pending.timer = time.AfterFunc(s.pendingLoginTTL, func() {
			s.expirePendingLogin(sessionIDCopy)
		})
		s.pendingMu.Unlock()
	}

	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	s.sequence++
	s.sink.Emit(EventLoginURL, LoginURLEvent{
		Sequence:   s.sequence,
		SessionID:  sessionID,
		EndpointID: endpointID,
		URL:        url,
	})
	return true, nil
}

func (s *Service) emitLoginFinished(sessionID string, snapshot AppSnapshot) {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	s.sequence++
	s.sink.Emit(EventLoginFinished, LoginFinishedEvent{
		Sequence:  s.sequence,
		SessionID: sessionID,
		Snapshot:  snapshot,
	})
}

func (s *Service) emitOperationFailed(operation string, err error) {
	problem := domain.ProblemFromError(err)
	if problem == nil {
		return
	}
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	s.sequence++
	s.sink.Emit(EventOperationFailed, OperationFailedEvent{
		Sequence:  s.sequence,
		Operation: operation,
		Problem:   *problem,
	})
}

func resolveActiveEndpointID(raw domain.AppSnapshot, activeProfileID string, profiles []domain.ProfileSummary, endpoints []domain.ControlEndpoint) string {
	for _, profile := range profiles {
		if profile.ID != activeProfileID {
			continue
		}
		if endpointID := resolveProfileDeclaredEndpointID(profile, endpoints); endpointID != "" {
			return endpointID
		}
		break
	}
	if raw.ActiveProfile != nil && (activeProfileID == "" || raw.ActiveProfile.ID == activeProfileID) {
		if endpointID := resolveProfileDeclaredEndpointID(*raw.ActiveProfile, endpoints); endpointID != "" {
			return endpointID
		}
	}
	if raw.ActiveEndpoint != nil {
		if endpointID := endpointIDForControlURL(raw.ActiveEndpoint.BaseURL, endpoints); endpointID != "" {
			return endpointID
		}
	}
	if activeProfileID != "" {
		for _, endpoint := range endpoints {
			if containsString(endpoint.DaemonProfileIDs, activeProfileID) {
				return endpoint.ID
			}
		}
	}
	return ""
}

func resolveProfileEndpointID(profile domain.ProfileSummary, active bool, activeEndpointID string, endpoints []domain.ControlEndpoint) string {
	if endpointID := resolveProfileDeclaredEndpointID(profile, endpoints); endpointID != "" {
		return endpointID
	}
	for _, endpoint := range endpoints {
		if containsString(endpoint.DaemonProfileIDs, profile.ID) {
			return endpoint.ID
		}
	}
	if active {
		return activeEndpointID
	}
	return ""
}

func resolveProfileDeclaredEndpointID(profile domain.ProfileSummary, endpoints []domain.ControlEndpoint) string {
	if endpointID := endpointIDForControlURL(profile.ControlURL, endpoints); endpointID != "" {
		return endpointID
	}
	if profile.EndpointID == "" {
		return ""
	}
	for _, endpoint := range endpoints {
		if endpoint.ID == profile.EndpointID {
			return endpoint.ID
		}
	}
	return ""
}

func endpointIDForControlURL(controlURL string, endpoints []domain.ControlEndpoint) string {
	if strings.TrimSpace(controlURL) == "" {
		return ""
	}
	for _, endpoint := range endpoints {
		if sameControlURL(endpoint.BaseURL, controlURL) {
			return endpoint.ID
		}
	}
	return ""
}

func offlineState(code domain.ErrorCode) domain.StateAxes {
	daemonState := domain.DaemonStopped
	switch code {
	case domain.ErrorDaemonMissing:
		daemonState = domain.DaemonMissing
	case domain.ErrorDaemonUnauthorized, domain.ErrorPermissionDenied:
		daemonState = domain.DaemonUnauthorized
	case domain.ErrorDaemonIncompatible:
		daemonState = domain.DaemonIncompatible
	case domain.ErrorDaemonStopped:
		daemonState = domain.DaemonStopped
	}
	return domain.StateAxes{
		Daemon:     daemonState,
		Session:    domain.SessionNone,
		Connection: domain.ConnectionStopped,
		Control:    domain.ControlUnknown,
	}
}

func providerFromKind(kind EndpointKind) (domain.EndpointProvider, error) {
	switch kind {
	case EndpointKindHeadscale:
		return domain.ProviderHeadscale, nil
	case EndpointKindTailscale:
		return domain.ProviderTailscale, nil
	case EndpointKindCompatible:
		return domain.ProviderCompatible, nil
	default:
		return "", domain.NewError(domain.ErrorInvalidArgument, "Endpoint kind is invalid.").WithDetail("kind")
	}
}

func kindFromProvider(provider domain.EndpointProvider) EndpointKind {
	switch provider {
	case domain.ProviderHeadscale:
		return EndpointKindHeadscale
	case domain.ProviderTailscale:
		return EndpointKindTailscale
	default:
		return EndpointKindCompatible
	}
}

func (s *Service) acquireMutation(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mutationWaiters.Add(1)
	defer s.mutationWaiters.Add(-1)
	select {
	case s.mutationGate <- struct{}{}:
		if err := ctx.Err(); err != nil {
			s.releaseMutation()
			return err
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Service) releaseMutation() {
	<-s.mutationGate
}

func sameControlURL(first, second string) bool {
	firstURL, firstErr := url.Parse(strings.TrimSpace(first))
	secondURL, secondErr := url.Parse(strings.TrimSpace(second))
	if firstErr != nil || secondErr != nil || firstURL.Host == "" || secondURL.Host == "" {
		return false
	}
	return strings.EqualFold(firstURL.Scheme, secondURL.Scheme) &&
		strings.EqualFold(firstURL.Host, secondURL.Host) &&
		strings.TrimRight(firstURL.EscapedPath(), "/") == strings.TrimRight(secondURL.EscapedPath(), "/") &&
		firstURL.RawQuery == secondURL.RawQuery &&
		firstURL.Fragment == secondURL.Fragment &&
		firstURL.User == nil && secondURL.User == nil
}

func normalizeError(err error, message string) error {
	if err == nil {
		return nil
	}
	var appError *domain.AppError
	if errors.As(err, &appError) {
		return err
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return domain.WrapError(domain.ErrorTimeout, message, err).WithRetryable(true)
	}
	if errors.Is(err, context.Canceled) {
		return domain.WrapError(domain.ErrorCancelled, message, err)
	}
	return domain.WrapError(domain.ErrorInternal, message, err)
}

func newSessionID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

func nonNilStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func stringPointer(value string) *string {
	return &value
}

func waitForContext(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func setAutostart(autostart Autostart, enabled bool) error {
	if enabled {
		return autostart.Enable()
	}
	return autostart.Disable()
}
