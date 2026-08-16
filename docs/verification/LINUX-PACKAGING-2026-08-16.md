# Linux Packaging Verification: 2026-08-16

This report records implementation and host-independent verification of the
managed Linux daemon path. It does not claim a real systemd desktop smoke test.

## Implemented

- Linux AMD64 and ARM64 official Tailscale `1.102.2` tarballs are pinned.
- The preparation tool accepts only the official HTTPS package host.
- Archive SHA-256, runtime-file SHA-256, command path, module path, and module
  version are verified before payload output is written.
- DEB, RPM, and Arch package definitions contain the GUI, daemon, CLI, service
  unit, provenance, and license notice.
- `headscaleclient-tailscaled.service` uses the standard LocalAPI socket and a
  product-owned persistent state directory.
- Install scripts preserve a loaded external `tailscaled.service`.
- Remove scripts disable only the HeadscaleClient-owned service.
- Runtime inspection prefers a running external service, reports ownership,
  and exposes only a fixed service-start operation through PolicyKit.
- The AppImage task is explicitly GUI-only and requires an external daemon.

## Verification completed

- Both official archives and all four runtime files matched committed hashes.
- Go build metadata reported `tailscale.com v1.102.2` and the expected
  `tailscale.com/cmd/tailscale[d]` paths.
- Linux daemon lifecycle tests ran successfully in a Linux Go container.
- Linux shell lifecycle scripts passed `sh -n` parsing.
- Linux package creation and required-content inspection are configured in CI.
- The Linux-specific Go package cross-compiled successfully from Windows.

## Remaining real-host checks

- Build a native Linux GUI artifact on an Ubuntu runner or Linux workstation.
- Install the DEB on a clean systemd desktop and complete interactive login.
- Verify upgrade, reboot, daemon repair, and uninstall with no external service.
- Repeat while an official `tailscaled.service` is installed and confirm its
  executable, state, enablement, and running status remain unchanged.
- Build and inspect RPM and Arch artifacts on representative distributions.

The local Windows Docker daemon could access an alternate container registry
but could not reach Debian package repositories, so a real GUI package was not
fabricated from a placeholder binary. CI is the authoritative package-build
path until a Linux host is available.
