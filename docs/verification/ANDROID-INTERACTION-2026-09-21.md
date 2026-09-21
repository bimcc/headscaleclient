# Android.2: keyboard, login and update verification

Status: rebuilt technical preview with automated Android 15 verification and
manual screenshot inspection. Honor physical-phone acceptance remains required.

## Artifact

- Version `0.2.1-android.2`, versionCode `2`, unchanged application ID
  `com.bimcc.headscaleclient.preview`.
- Source: `8bbda73` (production changes through `d1b35df`, followed by test-only frame synchronization).
- [Successful Android CI](https://github.com/bimcc/headscaleclient/actions/runs/35578505839).
- File: `bin/android-preview/headscaleclient-0.2.1-android.2.apk`.
- Size: 55,724,530 bytes.
- SHA-256: `c7609cfbba37df0b5e8f7f014f289af358e5fa3f567edc1514dab0cd121f1bd7`.
- RSA signing-certificate SHA-256:
  `11204724bb01854ac87127702e8d2c44a9712f1eb5f2d84fa40036a76b191253`.
- The private key is persisted outside Git and supplied to CI via repository
  secrets. The OpenSSL PKCS12 container was exported with Java 17 compatible PBE;
  the key and certificate were retained. APK signature verification passed.

## Root causes and changes

Official login failed locally because `NotifyRateLimit` was combined with
`NotifyNoNetMap`, `NotifyInitialStatus` and `NotifyPeerPatches`. Tailscale 1.102.2
explicitly rejects that subscription. This was our integration bug, not an official
control-server rejection, and can affect Headscale login too. The incompatible bit
was removed; the adapter has an upstream-validator regression test and Android
transport validates before reporting a successfully opened stream.

Android now lays out the WebView inside a real container padded for system bars,
display cutouts and the IME. This changes the web viewport size, unlike the previous
WebView-padding workaround. Server forms use a bounded scrolling body with a
visible header and sticky save action. Opening the Android sheet does not summon
the keyboard automatically. Native Back can dismiss a sheet, URL entry disables
autocapitalization/spellcheck, small-screen touch controls are larger, account
counts do not shrink offscreen and long addresses/errors wrap. Structured native
errors use the selected interface language without serializing internal causes.

## Checks

- Shared frontend: 69 tests passed and production bundle built.
- Root Go packages and mobile bridge tests passed; desktop CI passed on Windows,
  macOS and Linux. See the [Windows upgrade report](WINDOWS-UPGRADE-2026-09-21.md).
- Android lint: 0 errors, 12 warnings (pinned dependency/target updates, WebView
  feature guard analysis, intentional JavaScript, synchronous encrypted identity
  persistence and existing translation warnings).
- Six Android 15 emulator tests passed: exact privileged origin, official browser
  login URL from the real embedded core, authenticated encryption/key binding,
  native startup and Activity recreation, real soft-keyboard layout, and authorized
  foreground-service lifecycle/disconnect.
- The keyboard test waits for IME visibility AND the smaller CSS viewport, checks
  the URL field/save action and physical system-bar bounds, then captures a
  committed WebView frame. It also checks modal dismissal and account-count overflow.
- A same-signer replacement install preserved an app-data sentinel and left exactly
  one installed package ID. This is an install-in-place test, not a successful
  migration from android.1's lost debug signing key.
- `emulator-crashes.txt` was empty. Both final screenshots were visually inspected:
  URL field and Save remain above the keyboard; normal navigation is outside the
  status/gesture areas. Files in `bin/android-preview/`:
  `emulator-keyboard-android.2.png`, `emulator-overview-android.2.png`.

An earlier test reported success against pre-animation geometry and produced a
partly hidden sheet screenshot. That false pass was rejected during visual review.
Another run timed out waiting three seconds for a WebView callback on the emulator.
The final test waits for actual resized geometry and committed frames, with a
bounded ten-second callback timeout. The final screenshots supersede those runs.

## Limits and upgrade instructions

The official-service test obtains a login URL without authenticating a real account
or registering a peer. Complete official/Headscale browser authentication, peer
traffic, Honor keyboard variants, lock-screen persistence, network handover and
Clash Meta proxy coexistence still need real-phone validation.

Android.1 used an ephemeral debug certificate. Android does not allow replacing
that app with a differently signed APK. One-time uninstall/reinstall removes local
accounts/settings and requires login again. Do not change application ID to bypass
this restriction. Android.2 establishes the preserved signing identity for future
updates; it remains a debuggable technical preview, not a stable release.
