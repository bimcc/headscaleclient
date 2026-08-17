# Technical Design

## Pinned toolchain

| Component | Version | Policy |
| --- | --- | --- |
| Wails | `v3.0.0-beta.8` | Exact pin while v3 is beta |
| Go | `1.26.5` | Match the selected Tailscale module |
| Tailscale module | `v1.102.2` | Upgrade only in a dedicated compatibility change |
| Frontend | React + TypeScript + Vite | Strict TypeScript |
| Package manager | pnpm | Lockfile committed |

CI must assert the selected `tailscale.com` module version. No dependency on
Tailscale `main` or an unreviewed pseudo-version is allowed.

## Repository layout

```text
headscaleclient/
  frontend/                   React application
  internal/
    application/              use cases and state machine
    domain/                   product DTOs and errors
    tailscale/                LocalAPI adapter and mappings
    daemon/                   privileged service lifecycle adapter
    config/                   endpoint/settings persistence
    desktop/                  Wails platform integration
  tests/integration/          daemon contract tests
  docs/                       engineering source of truth
  build/                      Wails packaging assets
  main.go                     composition root only
```

## Backend contract

The first public service exposes a narrow contract:

```text
GetSnapshot() -> AppSnapshot
EnsureDaemon() -> AppSnapshot
SetRunning(bool) -> AppSnapshot
PatchPreferences(PreferencePatch) -> AppSnapshot
PingDevice(deviceID) -> PingResult
ListProfiles() -> ProfileCollection
SwitchProfile(profileID) -> AppSnapshot
Logout() -> AppSnapshot
BeginInteractiveLogin(endpointID) -> LoginSession
CancelInteractiveLogin(sessionID)
ListEndpoints() -> []ControlEndpoint
SaveEndpoint(ControlEndpointInput) -> ControlEndpoint
DeleteEndpoint(endpointID)
RetryDaemonConnection()
```

Events are versioned product events:

```text
app:snapshot-changed
app:login-url
app:login-finished
app:operation-failed
```

Every event includes a monotonically increasing sequence number. The frontend
fetches a full snapshot when it detects a gap.

Desktop presentation also uses `app:navigate` with a typed view target. It is
a local window-navigation request, not a sequenced product-state event.

## Peer route probing

Peer route status is a best-effort snapshot. `PeerStatus.CurAddr` is mapped to
Direct, DERP or peer-relay metadata is mapped to Relay, and an online peer with
neither is mapped to Unknown rather than Offline.

`PingDevice` uses `PingDisco`, matching the route-reporting mode of the official
Tailscale CLI. It makes at most three probes with a short bounded interval and
stops immediately when a direct endpoint is reported. A DERP region or peer
relay is Relay; a direct endpoint is Direct; missing route fields are Unknown.
TSMP is not used for route classification because Tailscale `v1.102.2` does not
populate endpoint or DERP fields for TSMP responses.

After a successful probe, the application composes and emits a new full
snapshot. The measured route and latency are applied to the matching peer
before publication, preventing a successful direct probe from leaving a stale
relay badge in the WebView or tray.

## Desktop quick surfaces

The main WebView is the detailed management surface. A native Wails
`SystemTray` menu is the quick interaction surface. `v3.0.0-beta.8` provides
runtime menu replacement, submenus, radio items, checkboxes, tray click
handlers, and window show/hide APIs on the supported desktop platforms.

The tray is implemented by `internal/desktop/TrayController` and follows these
rules:

1. It stores only the latest product `AppSnapshot`; raw Tailscale types never
   reach it.
2. It projects snapshot, language, busy, and error state into a render model.
   Equal models are ignored so repeated watcher events do not rebuild the
   native menu.
3. Menu callbacks invoke the same application service used by Wails bindings.
   The existing mutation gate serializes simultaneous tray and WebView actions.
4. Native UI callbacks never block Wails' UI loop. They disable the affected
   item, execute through Wails' callback worker, and render the resulting
   snapshot or failure.
5. Window destinations such as devices, accounts, and settings are requested
   through a typed `app:navigate` event after restoring the main window.

The initial native menu contains:

```text
connection state and connect/disconnect
active account -> saved profile radio items, manage accounts
this device identity
online devices -> bounded list plus open-all action
exit nodes -> none plus eligible ExitNodeOption peers
preferences -> MagicDNS, accept routes, shields-up
open detailed window
quit
```

Exit-node menus must use the daemon's `ExitNodeOption` capability. Being online
does not make a peer an eligible exit node.

The main-window account button is a React popover with the same profile list
and `SwitchProfile` operation. It supports outside click, Escape, keyboard
focus, a busy state, and explicit navigation to account management.

Windows is the first acceptance platform. macOS uses a template tray icon.
Linux behavior depends on a StatusNotifier/AppIndicator host; absence of one
must not stop the main window or networking service.

## Localization

`AppSettings.Language` is the only language preference. Supported values are
`zh-CN` and `en-US`. The Windows installer always presents a bilingual language
selector, stores its selection as `DefaultLanguage` in the 64-bit machine
uninstall registry key, and localizes its own pages and service messages.
`config.Store` uses that value only when the configuration or language field is
missing. Explicit saved preferences survive reinstall. Portable, non-Windows,
missing, and invalid platform values fall back to `zh-CN`.
`SetLanguage` persists the preference and returns a new `AppSnapshot`, so the
React tree, document language, and native tray update from the same state.

Frontend product copy uses typed translation keys with placeholder
interpolation. Endpoint names, account names, hostnames, addresses, tags, URLs,
and server-supplied diagnostic details are data and are never translated.

## Product DTOs

`AppSnapshot` includes:

- daemon state and version
- daemon ownership, service state, bundled version, and available fixed actions
- session and connection state
- current daemon health-warning details
- active control server and profile
- local device identity and addresses
- peer summaries
- user-editable preferences
- discovered capabilities
- last refresh time and optional structured problem

DTOs contain only JSON-friendly primitives and product enums. Unknown fields
from newer daemons are ignored; missing fields produce an explicit unknown
state rather than a zero-value assumption.

`AppSnapshot.Devices` is scoped to `ActiveProfileID` and
`ActiveEndpointID`. The frontend must clear profile-specific filters and
selection when either identity changes. It must not merge peer lists from
inactive profiles. Peer display names fall back from host name to MagicDNS,
virtual address, and stable node ID so malformed or partially populated daemon
records cannot render as anonymous rows.

The peer collection comes only from the active LocalAPI status map. It can
contain nodes owned by other server users when the control server publishes
them. The client labels this as visibility, not ownership or authorization;
Headscale/Tailscale policy and destination filtering remain authoritative.
Likewise, `ProfileSummary.LoginName` is an opaque login identity. An
email-shaped value is not treated as proof that an optional Headscale email
attribute exists.

Non-empty `ipnstate.Status.Health` entries are trimmed and copied through the
domain and product snapshots. Health entries drive degraded status as before,
but the frontend also renders their exact text so a transient warning is
diagnosable instead of being reduced to a boolean.

## Managed daemon supply chain

All daemon-bearing packages consume `build/daemon/manifest.json`. The Windows
preparation step downloads the exact upstream MSI and verifies:

1. The committed MSI SHA-256.
2. A valid Authenticode signature from Tailscale Inc.
3. The committed SHA-256 of `tailscaled.exe`, `tailscale.exe`, and `wintun.dll`.
4. Valid extracted-file signatures from Tailscale Inc. or WireGuard LLC.

Only verified generated files enter `bin/daemon/windows-<arch>/`. NSIS stops a
recognized managed service before replacing its executable, then registers and
starts the verified payload. A current or previous default product path is
repairable; an official or other external `Tailscale` service remains external
and is never re-registered. Setup validates the service start result. Uninstall
compares the actual service executable path with the product path before
removing it.

At runtime, a stopped prepared or managed Windows service is started or repaired
through one fixed elevated operation. Service state alone is insufficient:
`EnsureDaemon` waits until the protected LocalAPI endpoint responds before it
reports success. Application startup performs this check automatically only for
prepared or product-managed services; external services require an explicit
user action.

The trusted Windows release task signs the GUI before NSIS embeds it and signs
the resulting installer afterward, timestamping both signatures. A public
Authenticode certificate issued to BIMCC is an external release prerequisite;
unsigned development builds intentionally continue to show unknown publisher.

The Linux preparation step accepts only `https://pkgs.tailscale.com`, verifies
the committed archive SHA-256, verifies the committed SHA-256 for `tailscaled`
and `tailscale`, and parses Go build information to require the expected command
path and `tailscale.com v1.102.2`. DEB, RPM, and Arch packages install the
payload under `/usr/lib/headscaleclient/daemon/` and the dedicated
`headscaleclient-tailscaled.service` unit. Package scripts enable that unit only
when the standard `tailscaled.service` is not loaded.

The Linux GUI starts a stopped external or managed unit through a fixed
`systemctl` operation. It first uses the normal system service authorization
path and then falls back to PolicyKit (`pkexec`) for an explicit administrator
prompt. The frontend cannot choose the executable, unit, or arguments. Package
removal disables only `headscaleclient-tailscaled.service` and preserves daemon
state for reinstall or recovery. The GUI-only AppImage does not claim managed
service installation.

The release matrix treats daemon and GUI upgrades as one compatibility change.
Automatic daemon self-update remains disabled until signed product update and
rollback behavior are complete.

## Preference updates

The frontend sends a patch whose values are pointers/optional fields. The
adapter creates an `ipn.MaskedPrefs` and sets each matching mask bit. In
particular, `false`, an empty exit-node ID, and an empty list are valid updates
and must not be confused with an omitted field.

The frontend can never submit a complete daemon `Prefs` object because an old
UI could overwrite fields introduced by a newer daemon.

Selecting an exit node through HeadscaleClient also enables
`ExitNodeAllowLANAccess` by default. This prevents the exit-node blackhole route
from unexpectedly isolating locally accessible subnets. Users may still disable
LAN access explicitly. Existing profiles with an exit node and LAN access off
are not silently mutated; the UI presents a warning and a one-click recovery.
`RouteAll` remains an independent user preference because a Headscale-advertised
subnet that overlaps the current physical LAN can cause a separate route
conflict.

## Interactive login sequence

1. Validate and normalize the selected endpoint locally.
2. Probe the endpoint's HTTPS health path with a five-second timeout. DNS, TLS,
   transport, and HTTP 5xx failures stop here without mutating daemon state.
3. Switch to a new empty daemon profile.
4. Start the IPN watcher before starting login to avoid losing `BrowseToURL`.
5. Read the empty profile preferences and set the new `ControlURL`.
6. Call `Start` with the updated preferences.
7. Call `StartLoginInteractive` and require a login URL within 30 seconds.
8. Convert `BrowseToURL`, state, and login completion notifications to product events.
9. On completion, fetch a full snapshot and associate the daemon profile with the endpoint.

Changing the `ControlURL` of a running authenticated profile is not supported.
The UI creates or switches profiles instead of silently forcing reauthentication.

Endpoint records and daemon profiles have different ownership. The config store
persists endpoint metadata and profile associations; `tailscaled` persists
account credentials and selects exactly one active profile. `Logout` delegates
to LocalAPI for the current profile only. It does not use `DeleteProfile` to
pretend that an inactive local-profile deletion is a server logout.

## Watch strategy

Preferred mask:

```text
NotifyInitialState
| NotifyInitialPrefs
| NotifyInitialStatus
| NotifyPeerPatches
| NotifyNoNetMap
| NotifyRateLimit
```

If a daemon rejects the mask, retry with mask zero and fetch status/preferences
explicitly. After EOF or reconnect, always fetch a full snapshot before
applying new deltas. Raw `ipn.Notify` data never crosses the service boundary.

## Endpoint persistence

```text
ControlEndpoint
- id: UUID
- name: user-visible name
- baseURL: normalized HTTPS URL
- provider: auto | headscale | tailscale | compatible
- customCARef: optional secure-store reference
- daemonProfileIDs: zero or more associations
- createdAt / updatedAt
```

The official endpoint is a built-in record. Custom endpoints default to
provider `auto`; feature visibility is based on observed capabilities rather
than hostname matching.

## Testing strategy

### Unit tests

- Every preference patch maps to the correct value and `Set` bit.
- URL normalization rejects credentials, fragments, and unsafe schemes.
- State derivation covers daemon, session, connection, and control axes.
- Configuration migration and atomic persistence.

### Contract tests

- Inject `local.Client.Transport` to verify methods, paths, JSON, and status mapping.
- Exercise 204, 403, 412, malformed JSON, cancellation, and stream EOF.
- Verify login watches before issuing Start/Login calls.
- Verify route probing uses Disco, stops on direct, bounds relay attempts, and
  never treats missing route metadata as direct.

### Integration tests

- Run isolated Linux `tailscaled` with userspace networking and a temporary socket.
- Build matrix for Windows, macOS, and Linux.
- Verify current and two explicitly supported previous stable daemon minors.

### Frontend tests

- Vitest for state and components.
- Testing Library for keyboard and accessibility behavior.
- Account-popover tests cover switching, outside click, Escape, and busy state.
- Tray-controller tests cover status/profile/device/exit-node projection and
  event-listener lifecycle without starting a WebView.
- Playwright screenshots at desktop and narrow widths before UI milestones close.
- Localization tests verify the Chinese default, immediate English switching,
  document language, and English tray projection.
