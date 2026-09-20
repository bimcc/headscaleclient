# ADR 0004: Independent macOS launchd distribution

- Status: Accepted
- Date: 2026-09-20
- Supersedes: the macOS Network Extension prerequisite in ADR 0003.

## Decision

The first independent macOS edition uses Wails/WKWebView and the unmodified
open-source `tailscaled` daemon with the native `utun` interface. A root-owned
LaunchDaemon supplies the networking service. This is an upstream-supported
daemon mode, separate from Tailscale's App Store and standalone GUI extensions.
A Network Extension is not required for this route.

One PKG installs the GUI, daemon, diagnostic CLI, fixed service controller,
launchd registration, license and source/build provenance. Apple Silicon and
Intel packages are built natively on separate macOS CI runners. GUI-only ZIPs
remain available for users who already run a compatible Tailscale service.

`tailscale.com` is built from the version in the daemon manifest, verified
against the committed Go module checksum, using the upstream module's own
dependency graph. This is a BIMCC build of upstream sources, not an official
Tailscale binary. Each payload records hashes after ad-hoc signing.

The managed daemon has a dedicated Unix socket and state directory. When its
installation exists, LocalAPI uses only that socket; it never falls through to
an official GUI's TCP endpoint. Installing or starting the independent service
rejects detected official/Homebrew installations and competing daemons.
This explicit conflict policy prevents two VPN stacks from fighting over DNS
and routes. We never stop or uninstall an external service automatically.

Installer.app handles privileged initial installation. Subsequent repair uses
a fixed root-owned script through a macOS authorization dialog; the WebView
cannot supply commands, paths, or service names. The GUI is not elevated.
Upstream LocalAPI authorizes macOS admin-group users. Standard local accounts
need an administrator to configure the daemon's operator separately.

## Release limits

Initial preview builds use ad-hoc code signatures; they have no trusted
Developer ID signature or notarization. Public trusted distribution requires
Developer ID Application, Developer ID Installer, signing every executable,
and notarization/stapling. Ad-hoc signing is not a substitute for these.

CI covers both CPU architectures, native builds, package contents, initial
LocalAPI, restarts, upgrade preservation, GUI launch and service removal.
Real Headscale/OIDC login, sleep/wake, network roaming, DNS, exit nodes and
Gatekeeper on downloaded artifacts remain a real-Mac acceptance matrix.

Network Extension/App Store delivery may be a later product choice, without
changing the shared application or frontend contracts.
