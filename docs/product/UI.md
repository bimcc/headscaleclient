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

The main navigation contains four destinations:

| View | Purpose |
| --- | --- |
| Overview | Connection, current identity, local addresses, quick preferences |
| Devices | Searchable peer list, details, copy address, ping |
| Networks | Control servers and daemon login profiles |
| Settings | Application, connection, notification, update, and diagnostics settings |

Diagnostics is reached from an error banner or Settings. It does not occupy a
permanent navigation slot.

## Networks and accounts

The Networks view separates two related lists:

- Control servers show saved login targets. Each row owns its login or
  add-account action and displays `Current` plus reachability only when that
  endpoint is the daemon's active target. Inactive rows show `Not logged in` or
  their saved account count; they do not claim to have been probed.
- Accounts show the endpoint association and authentication state. The active
  row exposes logout, while inactive rows expose switch. Logout requires a
  confirmation because it disconnects and removes the active daemon profile.

The view always states that multiple accounts may be saved while only one
network can be active.

## Overview composition

1. Compact connection row with state, network, account, and a stable-size toggle.
2. Persistent alert only for actionable conditions.
3. Local device facts with individual copy actions.
4. Setting rows for exit node, LAN access, DNS, routes, and shields-up.
5. At most five recent online devices with a link to the full list.

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

- Settings offers Chinese and English, with Chinese selected by default.
- Changes apply immediately to the detailed window and native tray and persist
  across launches.
- User and server data is displayed verbatim; only product-owned interface copy
  is translated.
- Both languages must pass the desktop and narrow responsive checks.

## MVP components

`AppShell`, `PrimaryNavigation`, `ConnectionControl`, `PersistentAlert`,
`LocalDeviceSummary`, `QuickPreferences`, `DeviceTable`, `DeviceDetails`,
`EndpointList`, `EndpointDialog`, `ProfileList`, `LoginFlow`, `SettingRow`,
`DaemonHealthPanel`, `DiagnosticsDialog`, `EmptyState`, `Skeleton`,
`ToastRegion`, and `ConfirmDialog`.
