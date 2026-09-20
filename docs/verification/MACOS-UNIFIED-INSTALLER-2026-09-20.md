# Unified macOS installer verification

## Scope

One PKG per architecture replaces the separate managed and GUI-only downloads.
Service registration is conditional; bundled inactive files are installed in
both modes, like the Windows installer. No external service is overwritten.

## Checks

- Windows-host Go test/vet: passed (platform-independent regression checks).
- Frontend: all 65 tests and production build passed, including external macOS
  unavailable/unauthorized/incompatible guidance with no replacement action.
- Installer policy: 7 Node tests passed for script configuration, fresh,
  official, Homebrew, managed, conflicting and custom-socket UI hints.
- Shell syntax: preinstall/postinstall, policy, controller, uninstall, packaging
  and smoke scripts checked individually.
- [Initial native run 35498840852](https://github.com/bimcc/headscaleclient/actions/runs/35498840852):
  both Apple Silicon and Intel passed fresh install, managed upgrade, stopped-app
  detection, running/stopped external daemon, preserved PID/payload/preferences,
  GUI launch, conflict rejection, service removal and explicit migration.
- [Final native run 35499091166](https://github.com/bimcc/headscaleclient/actions/runs/35499091166)
  passed on both architectures at `f654741`. It additionally checks upgrading
  from the published split-package PKG with pinned SHA-256, retaining the state
  sentinel and working LocalAPI. Installer hints now enable JavaScript
  expressions per Apple's Distribution Definition schema. Installer choice XML
  evaluation and the 7 policy-expression tests passed (not a visual UI test).
- [Cross-platform CI 35499091039](https://github.com/bimcc/headscaleclient/actions/runs/35499091039)
  passed on Windows, macOS and Ubuntu, including frontend/Go tests, native GUI
  builds and Linux DEB content inspection.

## Limits

The official-app directory fixture tests detection only. It does not replace
testing actual App Store/standalone Network Extensions, custom sockets or local
standard-user permissions on a real Mac. External modes are never described as
compatible until LocalAPI works. Signing/notarization remains outstanding.
