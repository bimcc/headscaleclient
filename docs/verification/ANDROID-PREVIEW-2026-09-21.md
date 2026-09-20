# Android preview verification — 2026-09-21

## Artifact and provenance

- Version: `0.2.1-android.1`; application ID: `com.bimcc.headscaleclient.preview`.
- Source: `723fda0e2de2be7db54c559bd539a25fb2d4b1ef`, branch `codex/android-preview`.
- [Android build and emulator run](https://github.com/bimcc/headscaleclient/actions/runs/35522357510): passed.
- [Desktop Windows/macOS/Linux regression run](https://github.com/bimcc/headscaleclient/actions/runs/35522357696): passed.
- APK: `bin/android-preview/headscaleclient-0.2.1-android.1.apk`.
- Size: 54,583,558 bytes (about 55 MB / 52.1 MiB).
- SHA-256: `7480a138a48362732f164f1fc9059132f51f44b8697479c43508ae38cca8bc81`.
- Downloaded artifact hash matches CI and the local handoff copy.
- Android debug signing: one signer, APK Signature Scheme v2 verified by apksigner.
  This is a technical preview, not a production signing identity or store release.

The APK contains ARM64 and x86_64 Go network libraries, shared frontend assets,
and full Tailscale, tailscale-android, WireGuard-Go, Go mobile, Wails, React,
React DOM, Lucide and Apache-2.0 license texts. Network sources are pinned to
Tailscale 1.102.2 and tailscale-android commit
`0b3c1bdb207a01e2ec226ad91bd127c295fa5c4b`.

## Checks completed

The shared frontend passed 68 tests and a production build. Shared Go application,
configuration and Tailscale adapter regression tests passed. Mobile boundary tests
cover invalid methods/arguments, ownership versus actual TUN state, and preserving
real warnings while removing the normal notice for an intentional disconnect.

Android ARM64/x86_64 packaging and lint passed. Lint reports zero errors and 11
warnings (including pinned dependency versions, target-SDK currency, explicit
JavaScript use, feature-check analysis and preference persistence). A single
documented manifest suppression addresses AGP 8.9's incomplete alarm-only model
for `systemExempted`; an authorized VPN also qualifies under Android's rules.
The emulator foreground-service test verifies that permission path on API 35.

Four instrumentation tests passed on an Android 15 / API 35 x86_64 emulator:

1. Embedded core reports a native, ready snapshot; the React overview renders
   through the real message bridge before and after Activity recreation. Core
   version is `1.102.2`; normal stopped state produces no network warning.
2. Keystore state round-trips and ciphertext cannot be copied to a different
   preference key without failing authentication.
3. The privileged asset origin rejects HTTP, foreign hosts, credentials, alternate
   ports and file URLs.
4. With VPN permission granted in the disposable emulator, the foreground service
   starts, survives Activity recreation and stops on disconnect. This test uses
   an unauthenticated core; it does not establish a peer-data tunnel.

The same APK was then installed normally with notification permission, launched,
and captured in `bin/android-preview/emulator-overview.png`. Visual review confirms
the Chinese mobile layout, correct version and absence of the false stopped-state
warning. No AndroidRuntime crash output was recorded.

## Resource sample and limits

The newly opened, unauthenticated main process reported PSS 113,171 KiB (110.5 MiB)
and RSS 244,680 KiB (238.9 MiB), with one Activity and one WebView. The raw sample
is `bin/android-preview/emulator-memory.txt`. This is a single emulator startup
sample, not total physical-phone memory, a steady-state VPN measurement, or a
battery-life result; isolated renderer accounting may differ.

Honor model and MagicOS version are not yet known. Physical ARM64 installation,
Headscale/Logto and official Tailscale login, actual TUN establishment and peer
traffic, permission denial and real OS revocation, screen lock, Wi-Fi/cellular
handover, long-term battery usage and Clash Meta proxy coexistence remain manual
acceptance gates. The user's Clash Meta is in proxy mode; two VPN/TUN providers
still cannot own the same Android user/profile simultaneously.

No production Headscale settings or accounts were changed during verification.
No stable Android GitHub Release was created. See [Android product/build notes](../product/ANDROID.md)
and [ADR 0007](../adr/0007-android-vpn-host.md).
