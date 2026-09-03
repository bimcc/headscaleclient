# UI Principles

## Direction

HeadscaleClient is an operational desktop tool. The interface is compact,
restrained, and optimized for repeated use. It does not use a marketing page,
large status hero, decorative gradients, nested cards, or a network map as its
primary view.

The first viewport answers three questions:

1. Is the tunnel connected?
2. Which network and account are active?
3. Is this device healthy?

## Navigation

The main navigation contains five destinations:

| View | Purpose |
| --- | --- |
| Overview | Connection, current identity, local addresses, quick preferences |
| Devices | Searchable peer list, details, copy address, ping |
| Networks | Control servers and daemon login profiles |
| Settings | General desktop preferences plus local-service runtime diagnostics |
| About | Product version, publisher, independence statement, and upstream notices |

Diagnostics is reached from an error banner or Settings. It does not occupy a
permanent navigation slot.

Settings uses two unambiguous groups: General settings contains interactive
desktop preferences, while Runtime & diagnostics contains daemon status,
technical values, and the copy-summary action. Product attribution does not
appear as a settings footer; it belongs to the separate About view.

When the daemon is unavailable, Runtime & diagnostics exposes a fixed
`Start and repair` action for a product-managed service or `Start service` for
an external service. A service operation is successful only after LocalAPI is
reachable; users are never instructed to launch `tailscaled.exe` directly.

## Networks and accounts

The Networks view separates two related lists:

- Control servers show saved login targets. Each row owns its login or
  add-account action and displays `Current` plus reachability only when that
  endpoint is the daemon's active target. Inactive rows show `Not logged in` or
  their saved account count; they do not claim to have been probed.
- Accounts show the endpoint association and authentication state. The active
  row exposes `Remove identity`, while inactive rows expose switch. Removal is
  deliberately distinguished from temporarily disconnecting: its confirmation
  states that the daemon profile is deleted, the next connection requires
  browser authentication, and Headscale without OIDC also requires administrator
  approval. Temporary disconnection remains the Overview connection switch.

After an interactive login starts, the view explains that authentication is
completed in the browser and that non-OIDC Headscale deployments use an
administrator-approval page. Approval completion is still observed through the
daemon event stream, so no manual refresh is required.

The view always states that multiple accounts may be saved while only one
network can be active.

The selected-server heading places `Current network` and reachability badges
beside the server name. It does not repeat the server kind or account count
from the left list. The account count belongs to the Accounts section header;
there is no separate status strip between server actions and accounts.

## Overview composition

1. Compact connection row with state, network, account, online-device count,
   a link to Devices, and a stable-size toggle.
2. Persistent alert only for actionable conditions.
3. Local device facts with individual copy actions.
4. Current-network settings with an explicit control-server and account scope.
5. Setting rows for exit node, DNS, routes, and shields-up. LAN access is
   visually subordinate to exit-node selection and remains unavailable until
   an exit node is selected.

Selecting an exit node defaults LAN access to enabled. If an existing daemon
profile has an exit node with LAN access disabled, Overview shows a persistent
warning and a direct `Allow LAN access` action. Accept-routes copy warns that an
advertised subnet overlapping the current physical LAN must be disabled or
corrected at the control plane.

The exit-node selector distinguishes three server states: no peer has been
approved as an exit node, approved exit nodes exist but are offline, or online
eligible peers are selectable. Offline approved peers remain visible but
disabled. A stale selected node remains identifiable and can be cleared.

The Overview does not duplicate peer rows. The Devices view is the single
detailed inventory for the active daemon profile; switching the active account
or control server can replace that entire list.

The Devices heading calls these `Devices visible on this network`. A quiet
scope notice states that peers are published by the control server and that
visibility is not ownership or access authorization. Every row labels its
reported owner explicitly. Account-facing labels use `sign-in identity` where
the value is a daemon `LoginName`, because it may be a username or an email
address.

The Devices view provides a manual refresh action. It requests a new snapshot
from the local Tailscale LocalAPI, so the inventory, online state, connection
path, and names converge without changing the active account or network. Peer
display names prefer the control server's MagicDNS name with the network
suffix removed; the daemon hostname, virtual address, and stable node ID are
fallbacks when MagicDNS is unavailable. This keeps names changed in Headplane
or another control-server UI visible in the client.

When `tailscaled` reports health notices, Overview separates actionable health
warnings from informational configuration notices. Warnings use the amber
panel, affect the connection summary, and remain in copied diagnostics. Known
notices use localized product copy in the selected language; for example,
`routes-not-accepted` explains that advertised subnet routes are ignored while
ordinary virtual-address connections remain available. It uses a quiet teal
panel and does not degrade an otherwise healthy connection.

The exact upstream message remains available in copied diagnostics and element
metadata. Unrecognized messages stay in the warning panel with their original
text rather than receiving an unsafe guessed translation. Summary warning
counts exclude informational notices.

## Device paths

The device list labels its status-derived route as `Recent path`, not an
ever-current guarantee. A direct endpoint means Direct; a DERP or peer-relay
route means Relay; an online peer with no route evidence means Path unknown and
must never be presented as Offline.

The Ping action performs a bounded route probe. Its result is labelled `Probe
result`, uses only explicit direct-endpoint or relay evidence, and reports Path
unknown when the daemon omits route data. After a successful probe, the backend
refreshes the full snapshot and applies the measured path to the matching peer
so the list, detail drawer, and tray converge on the latest observation.

## State language

The UI does not reduce state to a boolean. It maintains these independent axes:

```text
daemon:     unknown | missing | stopped | ready | unauthorized | incompatible
session:    none | login-required | approval-required | authenticated
connection: stopped | starting | running | stopping | degraded
control:    unknown | reachable | unreachable
```

Derived user-visible states include Connected, Connecting, Disconnected,
Login required, Waiting for approval, Service unavailable, and Limited
connectivity. A temporarily unreachable control server does not imply an
existing peer tunnel is disconnected.

## Responsive layout

- `>= 1024px`: 184px left navigation, device table, right detail drawer.
- `720-1023px`: icon navigation, reduced table columns.
- `< 720px`: four-item bottom navigation, compact device rows, full-page details.
- Default window: `960 x 680`.
- Minimum window: `720 x 520`.

Names and URLs truncate with a tooltip. Async labels and icons have stable
dimensions so state changes do not shift the surrounding layout.

## Visual system

- Neutral surfaces with blue-green reserved for connected state.
- Amber for warnings and red for failures.
- Radius at or below 8px.
- Shadows only for menus, dialogs, and drawers.
- Lucide icons for familiar actions.
- 20-24px page titles and 14px default application text.
- No font size based on viewport width and no negative letter spacing.
- Respect system theme, scaling, reduced motion, and high contrast.

## Accessibility

- WCAG 2.2 AA contrast target.
- Complete keyboard navigation and visible focus.
- State never communicated by color alone.
- Connection changes announced through an `aria-live` region.
- Icon-only controls have accessible names and tooltips.
- Desktop targets are at least 36px; narrow-layout targets are at least 44px.

## Language

- The Windows installer always asks for Simplified Chinese or English. Its
  selection localizes setup and becomes the application's initial language.
- Portable and non-Windows builds use Chinese when no platform default exists.
- Changes apply immediately to the detailed window and native tray and persist
  across launches. A saved preference takes priority over the installer's
  initial value and is not overwritten on reinstall.
- User and server data is displayed verbatim; only product-owned interface copy
  is translated.
- Both languages must pass the desktop and narrow responsive checks.

## MVP components

`AppShell`, `PrimaryNavigation`, `ConnectionControl`, `PersistentAlert`,
`LocalDeviceSummary`, `QuickPreferences`, `DeviceTable`, `DeviceDetails`,
`EndpointList`, `EndpointDialog`, `ProfileList`, `LoginFlow`, `SettingRow`,
`DaemonHealthPanel`, `DiagnosticsDialog`, `AboutView`, `EmptyState`, `Skeleton`,
`ToastRegion`, and `ConfirmDialog`.
