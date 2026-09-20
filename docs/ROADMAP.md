# Roadmap

## M0: Foundation

- Documentation, ADRs, pinned toolchain, repository conventions.
- Wails application shell and CI-friendly build commands.
- Domain contracts, structured errors, and config store.

Exit: clean builds, tests run locally, and the application opens without a daemon.

## M1: Local daemon MVP

- Daemon discovery, status, preferences, and connection toggle.
- Full snapshot plus resilient event watching.
- Overview and diagnostic states.

Exit: connect/disconnect against a supported installed daemon on Windows.

## M1.5: Independent Windows distribution

- Verified pinned daemon and Wintun payload.
- Machine-level NSIS installer with service ownership protection.
- Runtime engine source/status reporting and explicit repair action.

Exit: one installer works without the official Tailscale GUI and preserves an
existing external service during install and uninstall.

## M2: Endpoints and authentication

- Official endpoint preset and custom compatible endpoints.
- Empty-profile creation, interactive login, browser URL, cancellation.
- Profile listing and switching.

Exit: authenticate against both Headscale and official Tailscale in manual tests.

## M3: Daily client workflows

- Device list/detail, copy, search, and ping.
- DNS, accept routes, shields up, exit node, and LAN access.
- Snapshot-driven native quick tray, functional header account switcher,
  close-to-tray, autostart, and single-instance behavior.

Exit: daily-use feature set passes Windows/macOS/Linux smoke matrix.

## M4: Distribution quality

- Signed Windows installer, notarized macOS application, Linux packages.
- Updater signing and rollback.
- Compatibility matrix and diagnostics export.
- Accessibility and responsive screenshot verification.

## M5: Android technical preview (in progress)

- Native foreground VPN service, shared Go business layer and React UI (ADR 0007).
- Independent APK, Headscale/OIDC and official Tailscale browser login.
- Keystore identity storage, VPN revoke and Activity/process lifecycle.
- Honor phone: background/screen-lock, network handover and Clash Meta proxy mode.

Exit: built APK plus emulator smoke evidence; physical-phone acceptance is a
separate gate before a stable Android release.

## Later

- Taildrop, subnet advertisement, SSH controls.
- Provider-specific admin adapters.
- Real-host Linux systemd install, upgrade, and uninstall matrix.
- Trusted macOS Developer ID signing/notarization and real-hardware acceptance
  for the independent launchd edition (ADR 0004); Network Extension/App Store
  delivery is a later option.
- iOS as a separate native VPN-extension project; Android store delivery/signing.
