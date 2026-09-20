# Unified macOS installer verification

## Scope

One PKG per architecture replaces the separate managed and GUI-only downloads.
Service registration is conditional; bundled inactive files are installed in
both modes, like the Windows installer. No external service is overwritten.

## Checks

- Windows-host Go test/vet: passed (platform-independent regression checks).
- Frontend: external macOS read/error guidance and no replacement action tests.
- Shell syntax: preinstall/postinstall, policy, controller, uninstall, packaging
  and smoke scripts checked individually.
- Native Apple Silicon/Intel: pending build, installer UI choice evaluation,
  fresh install, upgrade, stopped-app detection, running/stopped external daemon,
  preserved PID/payload/preferences, service removal and explicit migration.

## Limits

The official-app directory fixture tests detection only. It does not replace
testing actual App Store/standalone Network Extensions, custom sockets or local
standard-user permissions on a real Mac. External modes are never described as
compatible until LocalAPI works. Signing/notarization remains outstanding.
