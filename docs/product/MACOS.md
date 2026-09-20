# macOS edition

HeadscaleClient reuses the Windows edition's Wails 3 / React UI and application
logic. The macOS frontend uses WKWebView. Headscale, Tailscale endpoints, OIDC
browser login, account switching, device list/groups, Ping, network preferences,
Chinese/English and menu-bar controls use the same product operations.

## Packages

| Download | Intended use |
| --- | --- |
| `headscaleclient-macos-arm64-installer.pkg` | Apple Silicon (M-series), automatic service install/reuse |
| `headscaleclient-macos-amd64-installer.pkg` | Intel, automatic service install/reuse |

Build target: macOS 12 or newer. Native CI runs on macOS 15; macOS 12 and real
hardware networking are not yet separately certified. Choose the CPU shown in
Apple menu > About This Mac. The unified package requires administrator
approval and installs `/Applications/HeadscaleClient.app` and inactive bundled
components; a system service is registered only when no external service exists.
No official Tailscale GUI is needed. It defaults to Chinese; change the language
in Settings. The first preview is ad-hoc signed, not Developer ID signed or
notarized. macOS may require approval in System Settings > Privacy & Security
when opening a downloaded installer. Do not disable Gatekeeper globally.

There is no GUI-only/independent download choice. The welcome page explains the
policy and Installation Type shows advisory detection. Installation logs record
the actual decision (Window > Installer Log). Settings > Runtime diagnostics
shows the service source and readiness after installation.

- No service: register/start the bundled launchd service.
- Existing external official/Homebrew/manual service: leave it unchanged and
  use upstream LocalAPI discovery. Even a stopped detected installation is not
  silently replaced. Open the original app/start its service; unavailable,
  authorization and compatibility problems are reported in Settings.
- Existing managed service: update it while preserving state/settings.
- Both managed and external services: stop installation before changing either;
  explicitly resolve the conflict first.

Reuse shares the original service's active profile and network settings. Avoid
controlling it simultaneously from two interfaces. To switch to independent mode,
remove the external service using its own uninstall procedure and rerun the same
PKG. Its saved identities are not automatically copied; login may be needed.

## Runtime and permissions

- Wails runs as the signed-in user; networking runs under launchd as root.
- Service: `system/io.headscaleclient.tailscaled`.
- Payload: `/Library/Application Support/BIMCC/HeadscaleClient/daemon/`.
- Private state: `/Library/Application Support/BIMCC/HeadscaleClient/state/`.
- LocalAPI: `/var/run/headscaleclient-tailscaled.socket`.
- Service registration: `/Library/LaunchDaemons/io.headscaleclient.tailscaled.plist`.
- Installed policy result: `installation-mode` under the service root. This is
  only an ownership hint, not a credential or permission grant.
- Opening the UI reads the installed managed socket exclusively. Restart the
  GUI after installing or removing the managed service to change discovery mode.
- Mac admin-group users can control upstream LocalAPI without elevating the UI.
  For a standard local user, an administrator can explicitly grant operator
  access with the bundled CLI's `set --operator=<local-user>` option. This is a
  macOS account name, not a Headscale login. Check the authorization after
  creating/switching daemon profiles.
- Settings > Start and repair invokes the fixed service controller with the
  system administrator prompt. It verifies LocalAPI before reporting success.

Quit from the menu bar before upgrading. In managed mode, installing a new PKG
stops only the owned service, replaces binaries and restarts it. In external mode
the original service is left untouched. Its state and user settings
remain in place. Closing the GUI normally leaves networking running.

To remove the network service:

```sh
sudo /bin/sh '/Library/Application Support/BIMCC/HeadscaleClient/uninstall-service'
```

Then move `/Applications/HeadscaleClient.app` to Trash. The removal script
preserves login state and user settings; it does not uninstall an external
Tailscale service. Removing the app alone does not stop the background service.

## Build and validation

Run on a Mac with Xcode Command Line Tools, the pinned Go/Node/pnpm versions:

```sh
go tool wails3 task darwin:package:installer ARCH=arm64
# On an Intel Mac:
go tool wails3 task darwin:package:installer ARCH=amd64
```

The `macOS packages` GitHub Actions workflow builds both native architectures,
runs tests and checks installation on disposable Macs, then uploads PKG,
checksums and daemon provenance. Windows cannot run Apple's `pkgbuild`,
`codesign` or installation tests; use the workflow when developing on Windows.

The networking daemon is built from the checksum-pinned upstream source module
using its own dependency graph. Its source is not patched. Product-specific
version metadata identifies the independent build. Preview signing is ad-hoc;
trusted signing must be added before treating this as a notarized public build.

## Real-Mac acceptance still required

- Downloaded-package Gatekeeper approval and fresh install on the target Mac.
- Headscale OIDC login, reconnect, profile switching and logout.
- Virtual-IP traffic, direct/relay transitions, DNS, subnet routes and exit nodes.
- Sleep/wake, Wi-Fi changes, startup at login and menu-bar interaction.
- Upgrade/removal with an existing real identity, standard-user authorization.

See [ADR 0004](../adr/0004-macos-launchd-distribution.md) for the service choice.
See [ADR 0005](../adr/0005-macos-unified-installer.md) for unified installer policy.
