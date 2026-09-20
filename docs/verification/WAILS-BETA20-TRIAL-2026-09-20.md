# Wails beta.20 comparison trial

Status: local Windows comparison candidate built and verified. The user
subsequently reported that beta.20 also passed and requested a GitHub release.

## Scope and evidence

The user reported that the beta.8 rollback installer no longer triggers the
detection reported for the beta.23-based 0.2.0 installer. This supports testing
an intermediate framework version. No Defender engine/signature logs from
that retest were provided; the precise detection trigger remains unknown.

Pin the Go module and CLI to `v3.0.0-beta.20` and frontend runtime to
`3.0.0-beta.20`, including diagnostics and CI. Keep application version 0.2.0,
Go 1.26.5, Tailscale 1.102.2, application features and installer configuration.
Retain the beta.8 comparison installer; name the new artifact with `wails-beta20`.
Do not replace the published GitHub 0.2.0 assets with a comparison build.

## Compatibility review

The [upstream beta.20 release notes](https://github.com/wailsapp/wails/releases/tag/v3.0.0-beta.20)
remove the optional native WebView2 loader and its embedded DLLs in favor of
the pure-Go loader. Our beta.8 Windows binary already used the default pure-Go
loader, with no `native_webview2loader` build tag or linked `go-winloader`.
This removal does not require adopting a different loader in our build and
does not by itself identify the cause of the reported detection.

## Verification

- `go list -m github.com/wailsapp/wails/v3` and `go tool wails3 version`:
  `v3.0.0-beta.20`.
- `go mod verify`: all modules verified.
- `node tools/build/check-version.mjs`: passed; module, frontend runtime,
  CLI configuration, diagnostics and application metadata agree.
- Beta.20 binding generation, including production mode: passed, with no
  tracked binding changes.
- `go test ./...` and `go vet ./...`: passed.
- Frontend `pnpm test`: 65 tests in 5 files passed.
- TypeScript checking and Vite production build: passed.
- `go tool wails3 task windows:package ARCH=amd64`: passed.
- Extracted the staged installer without executing it. The packaged GUI's
  SHA-256 matches the build output. Its Go build information confirms Go
  1.26.5, Wails `v3.0.0-beta.20`, Tailscale `v1.102.2`, `production`,
  `CGO_ENABLED=0` and `windows/amd64`.
- All 15 package files have counterparts in the beta.8 candidate. Only the
  GUI differs; the other 14 files are byte-identical, including the uninstaller,
  NSIS plugins, WebView2 bootstrapper, Tailscale service and Wintun DLL.
- The three daemon payload hashes also match `build/daemon/manifest.json`.
- Installer remains NSIS 3 Unicode, non-solid Deflate, 64,512-byte stub,
  1011 install instructions, 112 uninstall instructions and two languages.
- GUI/installer product and file versions remain 0.2.0. Installer is unsigned,
  as were the beta.8 and beta.23 comparison packages.
- The preserved beta.8 installer still matches its recorded SHA-256.
- `git diff --check`: passed.

## Local artifacts

Installer:
`bin/trial-wails-beta20/headscaleclient-0.2.0-windows-amd64-wails-beta20-installer.exe`

- Size: 24,991,303 bytes.
- SHA-256: `0bb68670655111faf64de1979f1596265f8aed39c5f1b7e4f8df7a91a71b30e6`.

Packaged GUI: `bin/trial-wails-beta20/extracted/headscaleclient.exe`

- Size: 16,545,792 bytes.
- SHA-256: `766bc768c64659956e280620121e9385457247c3a800b4c436af84d7e13045e3`.

The checksum file is alongside the installer at
`bin/trial-wails-beta20/SHA256SUMS.txt`. The retained beta.8 fallback and its
hash are documented in [the rollback record](WAILS-BETA8-ROLLBACK-2026-09-20.md).

## Limits

This development host has no usable Defender scanner. Acceptance of this exact
beta.20 artifact is based on the user's report; scanner engine/signature logs
were not supplied. Native macOS/Linux
regression checks are not part of the Windows comparison build. No public
release is published by this trial.
The installed GUI and network service were not stopped or replaced. Native
GUI startup and installation of beta.20 are not claimed as verified by the
automated checks.

## Release follow-up

The user's accepted beta.20 trial is the framework baseline for release 0.2.1.
Changing product version metadata requires rebuilding the packages, so the
trial's SHA-256 above must not be attributed to the new release installer.
See [release verification](RELEASE-0.2.1.md) for the final artifacts.
