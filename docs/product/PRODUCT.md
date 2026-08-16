# Product Scope

## Product statement

HeadscaleClient is a quiet, independently installable desktop client for
Tailscale-compatible networks. It manages an unmodified upstream networking
service and lets a user connect to Headscale, the official Tailscale service,
or another compatible control server without binding the application to a
single provider.

The name describes the primary self-hosted use case, not a provider lock.

## Primary users

- A self-hoster connecting personal devices to Headscale.
- An operator who uses multiple compatible control servers.
- A user who wants a consistent desktop UI across Windows, macOS, and Linux.
- A support engineer who needs clear daemon and connectivity diagnostics.

## Product goals

- Show daemon, authentication, control-plane, and tunnel state accurately.
- Make connect, disconnect, login, profile switching, and exit-node selection fast.
- Treat official Tailscale and Headscale as peers in the endpoint model.
- Keep upstream `tailscaled` unmodified and replaceable.
- Deliver one product installer that supplies the pinned daemon when needed.
- Reuse an existing compatible daemon without replacing or taking ownership of it.
- Degrade cleanly when a daemon or server does not support a capability.
- Ship one coherent interface on all three desktop platforms.

## Non-goals for the MVP

- Reimplementing WireGuard or the Tailscale control protocol.
- Hosting a control server.
- Editing ACLs, users, or server policy.
- Reimplementing platform VPN drivers or hiding privilege elevation.
- Supporting Android or iOS VPN extensions.
- Claiming feature parity for Serve, Funnel, Tailnet Lock, or every admin API.

## Core workflows

### Control servers and accounts

- Control servers are saved login targets. Saving several endpoints does not
  connect to them concurrently.
- Login creates a daemon-owned account profile associated with the selected
  endpoint. Several profiles can remain saved on the device.
- The management UI presents this ownership as a master-detail hierarchy:
  saved control servers are the first level, and only the local profiles
  associated with the selected server appear in its account list.
- Exactly one daemon profile, and therefore one control server, can be active
  at a time. Switching profiles stops the current connection before activating
  the selected network.
- Every endpoint exposes its own login/add-account action. The application
  never silently substitutes the current endpoint for the selected endpoint.
- Login first performs a short control-server reachability check. DNS, TLS,
  timeout, and HTTP 5xx failures are shown against the selected endpoint and do
  not leave the daemon in a new empty profile.
- Logout applies only to the active profile. It contacts that profile's control
  server, disconnects it, and removes its local daemon profile after explicit
  confirmation. Other saved profiles remain available.

### First run

1. Detect, install, or start the local networking service with explicit privilege prompts.
2. Choose the official service or add a custom HTTPS control server.
3. Create a new daemon profile and begin interactive login.
4. Open the authentication URL and show a QR option.
5. Distinguish login, approval, connected, and failure states.

### Daily use

Daily interaction is split deliberately between two surfaces:

- The native system-tray menu is the quick surface. It shows connection
  health, the active account, this device, online devices, exit nodes, and the
  highest-frequency preferences without opening a WebView window.
- The main window is the detailed surface. It owns endpoint editing, login,
  logout confirmation, diagnostics, device inspection, and all explanatory
  states.

Both surfaces consume the same application snapshot and call the same
serialized application service. The tray never maintains an independent
connection or account state.

1. See current connection and health from the tray or main window.
2. Connect or disconnect with one action.
3. Copy this device's IP or DNS name.
4. Inspect peers and switch an eligible exit node.
5. Switch accounts/control servers from the tray, header, or account page
   without destroying another profile.
6. Log out the active account without removing other saved accounts.

### Device and address scope

- The device page is a live view of the active daemon profile's network map.
  It is not an aggregate inventory across every saved server or account.
- Switching the active profile can replace the full peer list because the new
  profile may belong to a different control server and private network.
- Virtual `100.x` and IPv6 addresses are assigned inside that profile's
  network. They are normally stable while the node identity remains registered,
  but are not a permanent cross-server device identifier and may change after
  deletion, re-registration, or identity reset.
- A peer may have multiple virtual addresses. The table shows the first address
  and a count; device details expose every address.

## MVP acceptance criteria

- Starts on Windows, macOS, and Linux from the same source tree.
- Detects missing, stopped, unauthorized, incompatible, and ready daemons.
- Installs a verified pinned daemon when the platform package supports it.
- Never replaces or removes an externally installed compatible daemon.
- Reads status and preferences without blocking the UI.
- Connects and disconnects through masked preference updates.
- Adds, validates, persists, edits, and removes control-server metadata.
- Starts interactive login against a selected control server.
- Fails fast with an actionable message when the selected server is unavailable.
- Surfaces the browser authentication URL from the daemon event stream.
- Lists local daemon profiles and switches between them.
- Logs out the active daemon profile with explicit confirmation.
- Shows the local node and peers with online state and addresses.
- Supports DNS, accept-routes, shields-up, exit node, and LAN access where available.
- Provides a stateful native tray menu, single-instance behavior,
  close-to-tray, and autostart.
- Provides a functional account switcher in the main-window header.
- Never persists auth keys or disables TLS verification silently.
