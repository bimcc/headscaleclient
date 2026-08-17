# Task List

Legend: `[ ]` pending, `[>]` in progress, `[x]` complete, `[!]` blocked.

## Documentation and decisions

- [x] DOC-001 Define product goals, non-goals, and MVP acceptance criteria.
- [x] DOC-002 Record system architecture and ownership boundaries.
- [x] DOC-003 Record Wails 3 and LocalAPI architecture decisions.
- [x] DOC-004 Define compact cross-platform UI principles.
- [x] DOC-005 Define security model and secret-handling rules.
- [x] DOC-006 Create milestone roadmap and executable task ledger.
- [x] DOC-007 Record managed daemon distribution and privilege boundaries.
- [x] DOC-008 Define the native tray quick surface, detailed-window boundary,
  shared snapshot flow, and account-menu interaction.
- [x] DOC-009 Document localization ownership, fallback, and verification.

## Repository and toolchain

- [x] REP-001 Scaffold Wails `v3.0.0-beta.8` with React and TypeScript.
- [x] REP-002 Pin Go `1.26.5` and `tailscale.com v1.102.2`.
- [>] REP-003 Configure pnpm, strict TypeScript, lint, format, and tests. Type checking and tests are configured; lint and format checks remain.
- [x] REP-004 Add build/test scripts that do not depend on a global Wails install.
- [x] REP-005 Add baseline CI matrix for Windows, macOS, and Linux.

## Domain and application backend

- [x] BE-001 Define product DTOs, state axes, capabilities, and error codes.
- [x] BE-002 Define `DaemonAPI`, event bus, clock, and config-store interfaces.
- [x] BE-003 Implement snapshot orchestration with timeout and cancellation.
- [x] BE-004 Implement connect/disconnect and masked preference patches.
- [x] BE-005 Implement watcher lifecycle, full refresh, reconnect, and mask fallback.
- [x] BE-006 Implement endpoint validation and atomic persistence.
- [x] BE-007 Implement empty-profile interactive login flow.
- [x] BE-008 Implement profile listing and switching.
- [>] BE-009 Implement diagnostics and version compatibility reporting. Runtime diagnostics are complete; the supported-version matrix remains.
- [x] BE-010 Implement confirmed logout for the active daemon profile.
- [x] BE-011 Add fail-fast endpoint reachability checks before interactive login.
- [x] BE-012 Replace TSMP route misclassification with bounded Disco probes and
  publish a refreshed, measured peer path after Ping.
- [x] BE-013 Auto-start prepared or managed services, repair stopped managed
  Windows registrations, and require LocalAPI readiness before success.

## LocalAPI adapter tests

- [x] TST-001 Test every preference value and matching `Set` bit.
- [x] TST-002 Test status/prefs mapping and unknown-field behavior.
- [x] TST-003 Test 403, 412, malformed JSON, timeout, and cancellation mapping.
- [x] TST-004 Test watch-before-login ordering and URL event delivery.
- [x] TST-005 Test watcher EOF, reconnect, and legacy-mask fallback.
- [ ] TST-006 Add isolated userspace `tailscaled` integration test.
- [x] TST-007 Test endpoint-specific login selection and active-profile logout.
- [x] TST-008 Test endpoint health success, unsupported health paths, HTTP 503, and transport failure.
- [x] TST-009 Test relay-to-direct probing, bounded relay attempts, unknown
  route evidence, and post-Ping snapshot publication.

## Frontend shell

- [x] UI-001 Implement responsive application shell and four-view navigation.
- [x] UI-002 Implement overview state and stable-size connection control.
- [>] UI-003 Implement actionable daemon/problem banner and diagnostics dialog. Banner and diagnostic summary are complete; a full dialog remains optional polish.
- [x] UI-004 Implement local device summary and copy actions.
- [x] UI-005 Implement devices table/list and detail view.
- [x] UI-006 Implement endpoint list, add/edit dialog, and validation states.
- [x] UI-007 Implement profile list and interactive login flow.
- [x] UI-008 Implement compact preference setting rows.
- [x] UI-009 Implement empty, loading, error, and toast states.
- [x] UI-010 Verify keyboard navigation, reduced motion, and narrow layout.
- [x] UI-011 Make endpoint login, active server, account switching, and logout explicit.
- [x] UI-012 Keep endpoint login failures visible and restore retry controls.
- [x] UI-013 Replace the decorative header account button with an accessible,
  functional profile switcher and account-management actions.
- [x] UI-014 Group saved accounts under a selected control server using a
  responsive master-detail layout with keyboard server navigation.
- [x] UI-015 Make the device list's active network/account scope explicit,
  expose multiple-address semantics, and prevent anonymous device rows.
- [x] UI-016 Add persistent Chinese and English UI localization with Chinese
  as the default and immediate language switching.
- [x] UI-017 Clarify current-network setting scope, make LAN access subordinate
  to exit-node selection, and replace the duplicated Overview peer list with
  an online-device count linking to Devices.
- [x] UI-018 Reorganize Settings into general and runtime-diagnostics groups,
  move product attribution to a dedicated About view, and record upstream
  copyright and official project links.
- [x] UI-019 Remove redundant selected-server metadata and status rows, place
  reachability beside the current-network label, and move account count to the
  Accounts section header.
- [x] UI-020 Strengthen the exit-node setting hierarchy with a compact nested
  LAN-access row, connector, and explicit disabled styling.
- [x] UI-021 Rename device route state to recent path, distinguish online
  unknown routes from offline peers, and synchronize accurate probe results.
- [x] UI-022 Add settings-level service repair and an actionable exit-node LAN
  isolation warning; default newly selected exit nodes to LAN access enabled.
- [x] UI-023 Label peers as current-network-visible rather than account-owned,
  clarify opaque login identities, and expose exact daemon health warnings.
- [x] UI-024 Explain empty exit-node eligibility, retain approved offline peers
  as disabled options, and make stale selections recoverable.
- [x] UI-025 Make Escape close the header identity menu even before its deferred
  focus transfer completes.
- [x] UI-026 Classify daemon health output into warnings and configuration
  notices, localize the known accept-routes notice, and keep informational
  notices from degrading a healthy connection.

## Desktop integration

- [x] DESK-001 Implement single-instance behavior.
- [x] DESK-002 Implement baseline tray presence, open/quit menu, and close-to-tray.
- [>] DESK-003 Implement cross-platform autostart preference. Implementation is complete; real login-cycle verification remains.
- [x] DESK-004 Add native icons and packaging metadata.
- [>] DESK-005 Verify Windows, macOS, and representative Linux desktop behavior. Windows is verified; macOS and Linux remain.
- [x] DESK-006 Implement a snapshot-driven native tray with connection state,
  connect/disconnect, active account, window navigation, and status text.
- [x] DESK-007 Add tray profile switching, eligible exit nodes, bounded online
  devices, and quick preference toggles.
- [>] DESK-008 Verify dynamic tray updates and hidden-window operation on
  Windows, then macOS and representative Linux desktops. Projection and
  navigation-event tests are complete; real desktop smoke checks remain.
- [x] DESK-009 Localize the native tray from the shared snapshot and suppress
  native menu rebuilds when projected state is unchanged.
- [x] DESK-010 Add an always-visible Simplified Chinese/English installer
  choice, seed first-launch language without overwriting saved preferences,
  localize custom setup messages, and enable the checked finish-page run action.

## Resource efficiency

- [x] PERF-001 Confirm the frontend has no polling loop and the application
  owns one resilient daemon watcher.
- [x] PERF-002 Deduplicate native tray rendering for equal projected state.
- [x] PERF-003 Record Windows idle CPU, memory, bundle, binary, and installer
  measurements with their test limitations.
- [x] PERF-004 Remove unused scaffold images and an unreferenced font from the
  embedded frontend payload.

## Managed daemon distribution

- [x] MDE-001 Define managed, external, prepared, missing, and service lifecycle states.
- [x] MDE-002 Expose a fixed `EnsureDaemon` operation without generic command execution.
- [x] MDE-003 Pin Windows AMD64/ARM64 MSI and runtime hashes with signer verification.
- [x] MDE-004 Package daemon, CLI, Wintun, provenance, and licenses in NSIS.
- [x] MDE-005 Preserve an external Windows service during isolated install and uninstall.
- [>] MDE-006 Verify managed-service install, repair, and uninstall on a clean Windows VM.
- [>] MDE-007 Add independent Linux systemd packages and privilege policy. Runtime lifecycle, verified AMD64/ARM64 payloads, native-package scripts, and CI package inspection are implemented; real systemd host installation remains.
- [ ] MDE-008 Add signed macOS Network Extension and privileged helper.
- [>] MDE-009 Sign the GUI, daemon-bearing installer, and update manifest. The
  Windows build now signs the GUI before packaging and the outer installer
  afterward; a public BIMCC Authenticode certificate and update signing remain.
- [x] MDE-010 Stop managed services before payload replacement, re-register them
  during install, and validate the final Windows service start result.

## Release readiness

- [>] REL-001 Establish supported daemon compatibility matrix. `tailscaled 1.102.2` and Headscale `v0.29.3` are the first verified entries.
- [ ] REL-002 Add signed update-manifest design and rollback test.
- [>] REL-003 Configure Windows signing, macOS signing/notarization, Linux packaging. The correct Windows signing order and artifact paths plus service-bearing Linux package definitions are implemented; a trusted certificate, a real Linux build artifact, and macOS remain.
- [>] REL-004 Complete license notices for Wails, Tailscale, frontend dependencies. Direct dependencies are recorded; transitive release inventory remains.
- [>] REL-005 Run accessibility, screenshot, integration, and clean-machine checks. Frontend accessibility/responsive checks and Windows native smoke testing are complete.

## Verification evidence

- [x] 2026-08-15 Windows native build and lifecycle smoke test.
- [x] 2026-08-15 Live LocalAPI read against `tailscaled 1.102.2`.
- [x] 2026-08-15 Test Headscale `v0.29.3` HTTPS health and version probe.
- [x] 2026-08-15 Verified Windows daemon supply-chain payload and NSIS build.
- [x] 2026-08-15 Isolated installer/uninstaller preserved the external service.
- [x] 2026-08-16 Verified Linux AMD64/ARM64 daemon payloads and lifecycle tests.
- [>] 2026-08-16 Linux native package implementation. CI inspection is configured; real systemd installation remains.
- [x] 2026-08-16 Verified account/server master-detail and device-scope layouts
  at 960x680 and 390x844 with no page or control overflow.
- [x] 2026-08-16 Rebuilt the unsigned Windows AMD64 installer after tray,
  account hierarchy, and device-scope changes.
- [x] 2026-08-17 Verified Chinese/English switching and responsive layouts,
  recorded an idle resource sample, and rebuilt the Windows AMD64 installer.
- [x] 2026-08-17 Re-verified Overview, network detail, Settings, About, and
  five-item navigation at 960x680 and 390x844, then rebuilt the Windows AMD64
  installer.
- [x] 2026-08-17 Verified nested network-setting hierarchy, installer-selected
  first-launch language, two NSIS language tables, and checked run-now setup;
  rebuilt the unsigned Windows AMD64 installer and recorded its hash.
- [x] 2026-08-17 Corrected TSMP's false-direct classification, verified bounded
  Disco route probing and post-Ping path synchronization, and added unknown-path
  handling across backend, WebView, and tray.
- [x] 2026-08-17 Simplified the machine-wide Windows installation directory to
  `C:\Program Files\BIMCC\HeadscaleClient` without changing publisher metadata,
  added an existing-install confirmation, and implemented an exact-path
  migration for the previous managed service directory.
- [x] 2026-08-17 Hardened Windows service install/start/repair and LocalAPI
  readiness, then added exit-node LAN protection and diagnostics.
- [ ] Complete a separate-profile interactive login against the test Headscale endpoint.
- [ ] Complete macOS, real-host Linux, signed installer, and clean-machine matrices.

See [Windows verification](verification/WINDOWS-2026-08-15.md) for the sanitized report.
See [tray and information-architecture verification](verification/WINDOWS-2026-08-16-TRAY-UI.md)
for the latest automated, visual, and package evidence.
See [localization and resource verification](verification/WINDOWS-2026-08-17-LOCALIZATION-RESOURCES.md)
for language, responsive, resource, and installer evidence.
