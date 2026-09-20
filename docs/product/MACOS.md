# macOS edition

HeadscaleClient reuses the Windows edition's Wails 3 / React UI and application
logic. The macOS frontend uses WKWebView. Headscale, Tailscale endpoints, OIDC
browser login, account switching, device list/groups, Ping, network preferences,
Chinese/English and menu-bar controls use the same product operations.

## Packages

| Download | Intended use |
| --- | --- |
| `headscaleclient-macos-arm64-installer.pkg` | Apple Silicon (M-series), independent install |
| `headscaleclient-macos-amd64-installer.pkg` | Intel, independent install |
| `headscaleclient-macos-<arch>-gui-only.zip` | GUI only, existing compatible Tailscale service |

Build target: macOS 12 or newer. Native CI runs on macOS 15; macOS 12 and real
hardware networking are not yet separately certified. Choose the CPU shown in
Apple menu > About This Mac. The independent package requires administrator
approval and installs `/Applications/HeadscaleClient.app` plus a system service.
No official Tailscale GUI is needed. It defaults to Chinese; change the language
in Settings. The first preview is ad-hoc signed, not Developer ID signed or
notarized. macOS may require approval in System Settings > Privacy & Security
when opening a downloaded installer. Do not disable Gatekeeper globally.

If the official Tailscale app or a Homebrew Tailscale service is installed, use
the GUI-only archive to reuse it. The independent PKG rejects these detected
conflicts. To switch to independent mode, first remove the other service using
its own uninstall procedure; its saved identities are not automatically copied.

## Runtime and permissions

- Wails runs as the signed-in user; networking runs under launchd as root.
- Service: `system/io.headscaleclient.tailscaled`.
- Payload: `/Library/Application Support/BIMCC/HeadscaleClient/daemon/`.
- Private state: `/Library/Application Support/BIMCC/HeadscaleClient/state/`.
- LocalAPI: `/var/run/headscaleclient-tailscaled.socket`.
- Service registration: `/Library/LaunchDaemons/io.headscaleclient.tailscaled.plist`.
- Opening the UI reads the installed managed socket exclusively. Restart the
  GUI after installing or removing the managed service to change discovery mode.
- Mac admin-group users can control upstream LocalAPI without elevating the UI.
  For a standard local user, an administrator can explicitly grant operator
  access with the bundled CLI's `set --operator=<local-user>` option. This is a
  macOS account name, not a Headscale login. Check the authorization after
  creating/switching daemon profiles.
- Settings > Start and repair invokes the fixed service controller with the
  system administrator prompt. It verifies LocalAPI before reporting success.

Quit from the menu bar before upgrading. Installing a new PKG stops only the
owned service, replaces binaries and restarts it. Its state and user settings
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
runs tests and checks installation on disposable Macs, then uploads PKG, ZIP,
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
