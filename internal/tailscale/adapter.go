package tailscale

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/headscaleclient/headscaleclient/internal/domain"
	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
	"tailscale.com/util/dnsname"
)

type Adapter struct {
	daemon       daemonAPI
	pingAttempts int
	pingInterval time.Duration
}

const (
	defaultPingAttempts = 3
	defaultPingInterval = time.Second
	PingViaDirect       = "direct"
	PingViaRelay        = "relay"
	PingViaUnknown      = "unknown"
)

func NewAdapter() *Adapter {
	return newAdapterWithDaemon(newLocalDaemon())
}

func newAdapterWithDaemon(daemon daemonAPI) *Adapter {
	return &Adapter{
		daemon:       daemon,
		pingAttempts: defaultPingAttempts,
		pingInterval: defaultPingInterval,
	}
}

func (a *Adapter) Snapshot(ctx context.Context) (domain.AppSnapshot, error) {
	status, err := a.daemon.Status(ctx)
	if err != nil {
		return domain.AppSnapshot{}, classifyError(err)
	}
	prefs, err := a.daemon.GetPrefs(ctx)
	if err != nil {
		return domain.AppSnapshot{}, classifyError(err)
	}

	snapshot := mapSnapshot(status, prefs)
	current, _, profileErr := a.daemon.ProfileStatus(ctx)
	if profileErr == nil && current.ID != "" {
		profile := mapProfile(current, true)
		snapshot.ActiveProfile = &profile
	}
	return snapshot, nil
}

func (a *Adapter) SetRunning(ctx context.Context, running bool) (domain.Preferences, error) {
	patch := domain.PreferencePatch{WantRunning: &running}
	return a.PatchPreferences(ctx, patch)
}

func (a *Adapter) PatchPreferences(ctx context.Context, patch domain.PreferencePatch) (domain.Preferences, error) {
	masked := maskedPrefs(patch)
	if masked.IsEmpty() {
		return domain.Preferences{}, domain.NewError(domain.ErrorInvalidArgument, "At least one preference must be provided.")
	}
	prefs, err := a.daemon.EditPrefs(ctx, masked)
	if err != nil {
		return domain.Preferences{}, classifyError(err)
	}
	return mapPreferences(prefs), nil
}

func (a *Adapter) Profiles(ctx context.Context) (domain.ProfileCollection, error) {
	current, profiles, err := a.daemon.ProfileStatus(ctx)
	if err != nil {
		return domain.ProfileCollection{}, classifyError(err)
	}
	result := domain.ProfileCollection{Profiles: make([]domain.ProfileSummary, 0, len(profiles))}
	for _, profile := range profiles {
		active := profile.ID == current.ID
		result.Profiles = append(result.Profiles, mapProfile(profile, active))
		if active {
			result.ActiveID = string(profile.ID)
		}
	}
	return result, nil
}

func (a *Adapter) SwitchProfile(ctx context.Context, profileID string) error {
	if strings.TrimSpace(profileID) == "" {
		return domain.NewError(domain.ErrorInvalidArgument, "A profile ID is required.")
	}
	if err := a.daemon.SwitchProfile(ctx, ipn.ProfileID(profileID)); err != nil {
		return classifyError(err)
	}
	return nil
}

func (a *Adapter) Logout(ctx context.Context) error {
	if err := a.daemon.Logout(ctx); err != nil {
		return classifyError(err)
	}
	return nil
}

func (a *Adapter) BeginInteractiveLogin(ctx context.Context, controlURL string) (string, error) {
	if err := a.daemon.SwitchToEmptyProfile(ctx); err != nil {
		return "", classifyError(err)
	}

	watcher, err := a.openWatcher(ctx)
	if err != nil {
		return "", err
	}
	defer watcher.Close()

	prefs, err := a.daemon.GetPrefs(ctx)
	if err != nil {
		return "", classifyError(err)
	}
	prefs.ControlURL = controlURL
	if err := a.daemon.Start(ctx, ipn.Options{UpdatePrefs: prefs}); err != nil {
		return "", classifyError(err)
	}
	if err := a.daemon.StartLoginInteractive(ctx); err != nil {
		return "", classifyError(err)
	}

	for {
		notify, err := watcher.Next()
		if err != nil {
			return "", classifyError(err)
		}
		if notify.ErrMessage != nil && *notify.ErrMessage != "" {
			return "", classifyError(&daemonMessageError{message: *notify.ErrMessage})
		}
		if notify.BrowseToURL != nil && *notify.BrowseToURL != "" {
			return ValidateLoginURL(*notify.BrowseToURL)
		}
	}
}

type PingResult struct {
	DeviceID    string `json:"deviceId"`
	LatencyMS   int64  `json:"latencyMs"`
	Via         string `json:"via"`
	RelayRegion string `json:"relayRegion,omitempty"`
	Endpoint    string `json:"endpoint,omitempty"`
}

func (a *Adapter) PingDevice(ctx context.Context, deviceID string) (PingResult, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return PingResult{}, domain.NewError(domain.ErrorInvalidArgument, "A device ID is required.")
	}
	status, err := a.daemon.Status(ctx)
	if err != nil {
		return PingResult{}, classifyError(err)
	}
	for _, peer := range status.Peer {
		if peer == nil || string(peer.ID) != deviceID {
			continue
		}
		if len(peer.TailscaleIPs) == 0 {
			return PingResult{}, domain.NewError(domain.ErrorPreconditionFailed, "The device has no Tailscale address.")
		}
		attempts := max(a.pingAttempts, 1)
		var last PingResult
		hasResult := false
		for attempt := 0; attempt < attempts; attempt++ {
			result, err := a.daemon.Ping(ctx, peer.TailscaleIPs[0], tailcfg.PingDisco)
			if err != nil {
				if hasResult && ctx.Err() == nil {
					return last, nil
				}
				return PingResult{}, classifyError(err)
			}
			if err := ctx.Err(); err != nil {
				return PingResult{}, classifyError(err)
			}
			if result == nil {
				if hasResult {
					return last, nil
				}
				return PingResult{}, domain.NewError(domain.ErrorDaemonUnavailable, "The device returned an empty ping response.").WithRetryable(true)
			}
			if result.Err != "" {
				if hasResult {
					return last, nil
				}
				return PingResult{}, domain.NewError(domain.ErrorDaemonUnavailable, "The device did not respond to ping.").
					WithDetail(result.Err).WithRetryable(true)
			}
			last = mapPingResult(deviceID, result)
			hasResult = true
			if last.Via == PingViaDirect || attempt == attempts-1 {
				return last, nil
			}
			if err := waitForPingAttempt(ctx, a.pingInterval); err != nil {
				return PingResult{}, classifyError(err)
			}
		}
		return last, nil
	}
	return PingResult{}, domain.NewError(domain.ErrorNotFound, fmt.Sprintf("Device %q was not found.", deviceID))
}

func mapPingResult(deviceID string, result *ipnstate.PingResult) PingResult {
	ping := PingResult{DeviceID: deviceID, Via: PingViaUnknown}
	if result == nil {
		return ping
	}
	ping.LatencyMS = int64(result.LatencySeconds*1000 + 0.5)
	ping.Endpoint = result.Endpoint
	ping.RelayRegion = result.DERPRegionCode
	switch {
	case result.PeerRelay != "", result.DERPRegionID != 0, result.DERPRegionCode != "":
		ping.Via = PingViaRelay
	case result.Endpoint != "":
		ping.Via = PingViaDirect
	}
	return ping
}

func waitForPingAttempt(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		return nil
	}
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func mapSnapshot(status *ipnstate.Status, prefs *ipn.Prefs) domain.AppSnapshot {
	now := time.Now().UTC().Format(time.RFC3339)
	state := mapState(status)
	daemonVersion := ""
	peerCount := 0
	if status != nil {
		daemonVersion = status.Version
		peerCount = len(status.Peer)
	}
	snapshot := domain.AppSnapshot{
		State:         state,
		DisplayState:  domain.DeriveDisplayState(state),
		HealthNotices: []domain.HealthNotice{},
		DaemonVersion: daemonVersion,
		Peers:         make([]domain.PeerSummary, 0, peerCount),
		Preferences:   mapPreferences(prefs),
		Capabilities: domain.Capabilities{
			ExitNode:     domain.CapabilitySupported,
			LANAccess:    domain.CapabilitySupported,
			DNS:          domain.CapabilitySupported,
			AcceptRoutes: domain.CapabilitySupported,
			ShieldsUp:    domain.CapabilitySupported,
			Ping:         domain.CapabilitySupported,
			Taildrop:     domain.CapabilityUnknown,
			Serve:        domain.CapabilityUnknown,
			Funnel:       domain.CapabilityUnknown,
			TailnetLock:  domain.CapabilityUnknown,
		},
		RefreshedAt: now,
	}

	if prefs != nil {
		controlURL := normalizeControlURL(prefs.ControlURL)
		provider := domain.ProviderAuto
		if controlURL == ipn.DefaultControlURL {
			provider = domain.ProviderTailscale
		}
		snapshot.ActiveEndpoint = &domain.EndpointSummary{
			Name:     controlURL,
			BaseURL:  controlURL,
			Provider: provider,
		}
	}
	if status != nil && status.Self != nil {
		device := mapDevice(status, status.Self)
		snapshot.LocalDevice = &device
	}
	if status == nil {
		return snapshot
	}
	for _, message := range status.Health {
		if notice, ok := mapHealthNotice(message); ok {
			snapshot.HealthNotices = append(snapshot.HealthNotices, notice)
		}
	}
	for _, peer := range status.Peer {
		if peer == nil || peer.ShareeNode {
			continue
		}
		lastSeen := ""
		if !peer.LastSeen.IsZero() {
			lastSeen = peer.LastSeen.UTC().Format(time.RFC3339)
		}
		connectionType, relayRegion := mapPeerConnection(peer)
		snapshot.Peers = append(snapshot.Peers, domain.PeerSummary{
			DeviceIdentity: mapDevice(status, peer),
			Online:         peer.Online,
			LastSeen:       lastSeen,
			ConnectionType: connectionType,
			RelayRegion:    relayRegion,
			ExitNode:       peer.ExitNode,
			ExitNodeOption: peer.ExitNodeOption,
		})
	}
	slices.SortFunc(snapshot.Peers, func(a, b domain.PeerSummary) int {
		if a.Online != b.Online {
			if a.Online {
				return -1
			}
			return 1
		}
		return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return snapshot
}

func mapState(status *ipnstate.Status) domain.StateAxes {
	state := domain.StateAxes{
		Daemon:     domain.DaemonReady,
		Session:    domain.SessionNone,
		Connection: domain.ConnectionStopped,
		Control:    domain.ControlUnknown,
	}
	if status == nil {
		state.Daemon = domain.DaemonUnknown
		return state
	}
	switch status.BackendState {
	case "NoState":
	case "InUseOtherUser":
		state.Daemon = domain.DaemonUnauthorized
	case "NeedsLogin":
		state.Session = domain.SessionLoginRequired
	case "NeedsMachineAuth":
		state.Session = domain.SessionApprovalRequired
	case "Starting":
		state.Session = domain.SessionAuthenticated
		state.Connection = domain.ConnectionStarting
	case "Running":
		state.Session = domain.SessionAuthenticated
		state.Connection = domain.ConnectionRunning
		state.Control = domain.ControlReachable
	case "Stopped":
		if status.HaveNodeKey {
			state.Session = domain.SessionAuthenticated
		} else {
			state.Session = domain.SessionLoginRequired
		}
	default:
		state.Daemon = domain.DaemonIncompatible
	}
	if state.Connection == domain.ConnectionRunning {
		for _, message := range status.Health {
			notice, ok := mapHealthNotice(message)
			if !ok || notice.Severity != domain.HealthNoticeWarning {
				continue
			}
			state.Connection = domain.ConnectionDegraded
			lower := strings.ToLower(notice.Message)
			if strings.Contains(lower, "control") || strings.Contains(lower, "map response") {
				state.Control = domain.ControlUnreachable
			}
		}
	}
	return state
}

func mapHealthNotice(message string) (domain.HealthNotice, bool) {
	message = strings.TrimSpace(message)
	if message == "" {
		return domain.HealthNotice{}, false
	}
	lower := strings.ToLower(message)
	if strings.Contains(lower, "advertising routes") &&
		strings.Contains(lower, "--accept-routes") &&
		strings.Contains(lower, "false") {
		return domain.HealthNotice{
			Code:     domain.HealthNoticeRoutesNotAccepted,
			Severity: domain.HealthNoticeInfo,
			Message:  message,
		}, true
	}
	return domain.HealthNotice{
		Code:     domain.HealthNoticeTailscaleWarning,
		Severity: domain.HealthNoticeWarning,
		Message:  message,
	}, true
}

func mapDevice(status *ipnstate.Status, peer *ipnstate.PeerStatus) domain.DeviceIdentity {
	addresses := make([]string, 0, len(peer.TailscaleIPs))
	for _, address := range peer.TailscaleIPs {
		addresses = append(addresses, address.String())
	}
	dnsName := strings.TrimSuffix(peer.DNSName, ".")
	magicDNSSuffix := status.MagicDNSSuffix
	if status.CurrentTailnet != nil && status.CurrentTailnet.MagicDNSSuffix != "" {
		magicDNSSuffix = status.CurrentTailnet.MagicDNSSuffix
	}
	name := dnsname.TrimSuffix(peer.DNSName, magicDNSSuffix)
	name = cmp.Or(name, dnsName, dnsname.SanitizeHostname(peer.HostName))
	if name == "" && len(addresses) > 0 {
		name = addresses[0]
	}
	if name == "" {
		name = string(peer.ID)
	}
	user := ""
	if profile, ok := status.User[peer.UserID]; ok {
		user = cmp.Or(profile.DisplayName, profile.LoginName)
	}
	return domain.DeviceIdentity{
		ID:        string(peer.ID),
		Name:      name,
		DNSName:   dnsName,
		User:      user,
		OS:        peer.OS,
		Addresses: addresses,
	}
}

func mapPeerConnection(peer *ipnstate.PeerStatus) (domain.PeerConnectionType, string) {
	if peer == nil || !peer.Online {
		return domain.PeerConnectionOffline, ""
	}
	if peer.CurAddr != "" {
		return domain.PeerConnectionDirect, ""
	}
	if peer.Relay != "" || peer.PeerRelay != "" {
		return domain.PeerConnectionRelay, peer.Relay
	}
	return domain.PeerConnectionUnknown, ""
}

func mapProfile(profile ipn.LoginProfile, active bool) domain.ProfileSummary {
	controlURL := normalizeControlURL(profile.ControlURL)
	return domain.ProfileSummary{
		ID:         string(profile.ID),
		Name:       cmp.Or(profile.Name, profile.UserProfile.DisplayName, controlURL),
		LoginName:  profile.UserProfile.LoginName,
		ControlURL: controlURL,
		Active:     active,
	}
}

func normalizeControlURL(value string) string {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if value == "" || ipn.IsLoginServerSynonym(strings.ToLower(value)) {
		return ipn.DefaultControlURL
	}
	return value
}
