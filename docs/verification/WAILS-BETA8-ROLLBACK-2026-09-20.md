# Wails beta.8 rollback verification

Status: local Windows rollback candidate built; automated checks passed.
The user subsequently retested it and reported that beta.8 no longer triggers
the detection. Scanner engine/signature details were not supplied.

## Scope

At the user's request, restore the old framework baseline across the Go module,
CLI, frontend runtime, generated bindings, diagnostics and CI configuration.
Go remains 1.26.5, Tailscale remains 1.102.2 and application version remains 0.2.0.
Current business logic and the macOS unified installer implementation are kept.

The Windows candidate uses a filename containing `wails-beta8`. Existing
GitHub 0.2.0 artifacts and their historical verification record are unchanged.

## Verification

- `go list -m github.com/wailsapp/wails/v3` and `go tool wails3 version`:
  `v3.0.0-beta.8`.
- `node tools/build/check-version.mjs`: passed, including the added agreement
  checks for the Go dependency, frontend runtime, CI CLI and diagnostics.
- `go mod verify`, `go test ./...` and `go vet ./...`: passed.
- Frontend `pnpm test`: 65 tests in 5 files passed.
- TypeScript checking and Vite production build: passed.
- Binding generation with the beta.8 CLI, including production mode: passed;
  generated bindings have no tracked changes.
- `go tool wails3 task windows:package ARCH=amd64`: passed.
- Extracted the final installer without executing it. The packaged GUI hash
  matches the build output. `go version -m` on that packaged GUI confirms
  Go 1.26.5, Wails `v3.0.0-beta.8` and Tailscale `v1.102.2`.
- Packaged `tailscale.exe`, `tailscaled.exe` and `wintun.dll` SHA-256 hashes
  match `build/daemon/manifest.json`.
- GUI and installer product/file metadata remain `0.2.0`, HeadscaleClient,
  BIMCC., Ltd. The installer is unsigned.
- `git diff --check`: passed.

## Local artifacts

Installer:
`bin/rollback-wails-beta8/headscaleclient-0.2.0-windows-amd64-wails-beta8-installer.exe`

- Size: 24,956,914 bytes.
- SHA-256: `f1776aac2c33324f68859c1654902573add778c12297ec02998561aee283f89e`.

GUI build output at the time of this build: `bin/headscaleclient.exe`.
The retained extracted copy is
`bin/rollback-wails-beta8/extracted/headscaleclient.exe`.

- Size: 16,428,032 bytes.
- SHA-256: `d25865e1dab5c2ab7eb6b0eb75ebff5e05375a5760372f2785bdf4a5840b4776`.

The installer uses NSIS 3 Unicode with non-solid Deflate and a 64,512-byte
stub, retaining the existing installer configuration. It contains 15 extracted
files. Changing the framework does not change the bundled network-service
binaries.

## Limits

This Windows development host has no active Defender scanner. The user's
successful beta.8 retest supports a framework-version-related difference, but
does not identify the particular upstream change or establish a universal
Defender verdict. Native macOS/Linux framework regression checks are also
pending. No new GitHub release is published by this rollback operation.
The already-running installed GUI and network service were not stopped or
replaced. Native GUI startup and installation of this candidate are not claimed
as verified by the automated build checks.

## Follow-up

After the successful user retest, the user requested a beta.20 comparison.
The beta.8 installer and extracted GUI remain available at the paths above.
See [beta.20 trial](WAILS-BETA20-TRIAL-2026-09-20.md) for that separate artifact.
