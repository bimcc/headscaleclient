# Release 0.2.0 verification

Status: preparing artifacts. GitHub release channel: prerelease.

## Scope

Synchronize application and installer metadata to 0.2.0 and rebuild Windows
AMD64, macOS ARM64/AMD64 and Linux AMD64 DEB from the release source commit.
Keep Wails beta.23, Go 1.26.5 and Tailscale 1.102.2 pinned.

`VERSION` is the release reference. `tools/build/check-version.mjs` checks Go,
frontend, Wails config, Windows resources/NSIS/MSIX, Mac/iOS template metadata
and Linux package metadata against it. Mobile templates are kept consistent;
this does not constitute a mobile release or mobile build validation.

## Verification plan

- Check metadata consistency; Go test/vet; frontend tests/production build.
- Windows NSIS build, numeric and display versions, runtime dependency metadata,
  and GUI/WebView2 startup without changing account/network settings.
- Native Mac builds, installation and external-service lifecycle tests; check
  installed Info.plist and package receipt version, and installed executable
  equality, including an upgrade from the published 0.1.0 preview.
- Linux native build and DEB contents/version checks, then upload artifact.
- Download native CI packages; verify SHA-256 before publication and GitHub
  uploaded digests after publication.

Results and artifact hashes will be recorded after those checks complete.

## Limits

Windows lacks trusted Authenticode signing. macOS is ad-hoc signed with an
unsigned PKG, without notarization. Mac real-account/Network Extension behavior,
sleep/roaming and Linux real-systemd installation remain separate acceptance.
