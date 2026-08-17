# ADR 0003: Managed upstream daemon distribution

- Status: Accepted
- Date: 2026-08-15

## Context

Requiring users to install the official Tailscale desktop application makes
HeadscaleClient a secondary GUI instead of an independent product. Rewriting
the networking engine would duplicate WireGuard, DERP, NAT traversal, routing,
DNS, and OS integration while creating an unacceptable security burden.

## Decision

Release packages may distribute an unmodified, pinned upstream `tailscaled`
binary as a separately privileged system service. The Wails process remains a
normal-user control surface and communicates through LocalAPI.

Windows packages use verified files extracted from the pinned official MSI.
Packaging fails unless the MSI hash, extracted-file hashes, and Authenticode
signers match the committed manifest. A compatible existing `Tailscale`
service is reused. Uninstall removes a service only when its executable path
belongs to the same HeadscaleClient installation.

The Windows installer explicitly selects Simplified Chinese or English,
localizes setup, stores the selection as a first-launch language hint, and
offers a checked run-now action on its finish page. Saved application settings
remain user-owned and are not overwritten during reinstall.

The frontend may invoke only the fixed `EnsureDaemon` operation. It cannot
provide executable paths, service names, or command arguments.

Linux native packages use official pinned tarballs and verify archive hashes,
runtime-file hashes, command paths, and Tailscale Go module versions. They
install `headscaleclient-tailscaled.service` with separate persistent state.
An existing standard `tailscaled.service` is reused and never replaced. The
AppImage remains GUI-only because an AppImage cannot safely register a root
system service as part of normal launch.

## Consequences

- Windows users can install one HeadscaleClient package without first
  installing the official Tailscale GUI.
- Upstream daemon upgrades require an explicit manifest and compatibility
  change rather than an unpinned download.
- Windows installation and service repair require administrator approval, but
  daily GUI operation does not.
- Windows release signing requires a publicly trusted Authenticode certificate
  issued to BIMCC. The build signs the GUI before packaging and the installer
  afterward; unsigned development packages still show unknown publisher.
- Linux DEB/RPM/Arch packages can be installed without the official Tailscale
  GUI; real-host install and uninstall matrices remain release verification.
- macOS requires a signed Network Extension and privileged helper strategy;
  copying a daemon binary into an app bundle is not sufficient.
- Tailscale and, on Windows, Wintun license notices ship with the managed payload.
