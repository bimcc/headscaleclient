# Release 0.2.0 verification

Status: published and verified on 2026-09-20. GitHub release channel: prerelease.

[Download HeadscaleClient v0.2.0](https://github.com/bimcc/headscaleclient/releases/tag/v0.2.0).

## Scope

Release source: `d43f9e72cd09e5c7c5eba6904af5922fad50573c`.

Synchronize application and installer metadata to 0.2.0 and rebuild Windows
AMD64, macOS ARM64/AMD64 and Linux AMD64 DEB from the release source commit.
Keep Wails beta.23, Go 1.26.5 and Tailscale 1.102.2 pinned.

`VERSION` is the release reference. `tools/build/check-version.mjs` checks Go,
frontend, Wails config, Windows resources/NSIS/MSIX, Mac/iOS template metadata
and Linux package metadata against it. Mobile templates are kept consistent;
this does not constitute a mobile release or mobile build validation.

## Verification plan

- Check metadata consistency; Go test/vet; frontend tests/production build.
- Windows NSIS build, numeric and display versions and runtime dependency metadata.
- Native Mac builds, installation and external-service lifecycle tests; check
  installed Info.plist and package receipt version, and installed executable
  equality, including an upgrade from the published 0.1.0 preview.
- Linux native build and DEB contents/version checks, then upload artifact.
- Download native CI packages; verify SHA-256 before publication and GitHub
  uploaded digests after publication.

## Completed checks

- [Cross-platform CI](https://github.com/bimcc/headscaleclient/actions/runs/35500703917):
  successful on Windows, macOS and Ubuntu 24.04. Each ran metadata consistency,
  module verification, 65 frontend tests, frontend build, Go test/vet and the
  native GUI build. Ubuntu also built the service-bearing AMD64 DEB and checked
  version `0.2.0-1`, application, daemon, systemd unit and provenance contents.
- Local Windows NSIS packaging succeeded with two language tables and the
  bundled daemon payload. Both application and installer report file/product
  version `0.2.0`, numeric versions `0.2.0.0`, company `BIMCC., Ltd.` and product
  `HeadscaleClient`. The EXE resource check also passed on the clean Windows CI
  runner. A missing fixed product version and neutral-language display strings
  were corrected before publishing; CI now checks the actual compiled resource.
- `go version -m bin/headscaleclient.exe` confirms Go `1.26.5`, Wails
  `v3.0.0-beta.23` and `tailscale.com v1.102.2` in the built Windows binary.
- [Native macOS package CI](https://github.com/bimcc/headscaleclient/actions/runs/35500703908):
  successful on macOS 15 ARM64 and Intel. Both passed installer policy tests,
  Go test/vet, 65 frontend tests, architecture/ad-hoc signature checks, and
  native installer lifecycle validation. Lifecycle checks included fresh
  installation, normal-user LocalAPI access, service restart, GUI launch,
  reinstall, uninstall, conflicting-service rejection, live/stopped external
  service reuse, and explicit migration after external-service removal.
- Both Mac jobs installed the previously published 0.1.0 split-package preview,
  confirmed that the installed app really was 0.1.0, then upgraded to 0.2.0.
  Installed app version, package receipt and executable equality matched the
  new build, and the saved state marker survived. External-service tests used
  an upstream userspace daemon and an official-app directory fixture; they did
  not exercise the signed official Network Extension app.
- Downloaded both Mac artifacts and matched their original CI SHA-256 files.
  Downloaded the Linux artifact from the successful CI run and independently
  read its DEB control metadata (`0.2.0-1`, `amd64`). Renaming package files for
  the release did not change their bytes.

## Artifacts

| File | Bytes | SHA-256 |
| --- | ---: | --- |
| `headscaleclient-0.2.0-windows-amd64-installer.exe` | 24992575 | `f2a23b636038344e6e25079094f2a9ba7fa35be12ba64da96a10b71dbfe987c5` |
| `headscaleclient-0.2.0-macos-arm64-installer.pkg` | 22403040 | `f246c62574f71934e9643743de07350457f64d89ece0d3e47bd7e269f22c9f00` |
| `headscaleclient-0.2.0-macos-amd64-installer.pkg` | 24584580 | `e67f982ea6fa37c077abfa0e6b974bfe147f1cf27d3a25a29742bd99221d8e49` |
| `headscaleclient-0.2.0-linux-amd64.deb` | 45052620 | `4fce0b6a9db72d0fd5e0911a367eda15df35629138843676c30d5cc72d595186` |

The release also includes `SHA256SUMS.txt` covering all four packages.
Its SHA-256 is `96fa322a85987609841119552591403597ab04a7477240c8272782f36d4cfbfb`.
After publication, all five assets reported `uploaded`; their exact byte sizes
and GitHub SHA-256 digests matched the local files. The release tag points to
the tested source commit above. The existing v0.1.0 stable/latest release was
retained; v0.2.0 is explicitly a prerelease.

## Limits

Windows lacks trusted Authenticode signing. macOS is ad-hoc signed with an
unsigned PKG, without notarization. Mac real-account/Network Extension behavior,
sleep/roaming and Linux real-systemd installation remain separate acceptance.
The 0.2.0 Windows GUI startup check was not rerun because the installed client
was already running; the single-instance application and its current session
were left undisturbed. This release does not claim a new Windows installation
or GUI startup acceptance run.
