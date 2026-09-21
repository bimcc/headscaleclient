# Android technical preview

Status: installable technical preview, APK and emulator checks passed on
2026-09-21. Honor physical-phone acceptance remains pending. This is not a stable
Android release. See [latest interaction and update verification](../verification/ANDROID-INTERACTION-2026-09-21.md)
and the [initial preview evidence](../verification/ANDROID-PREVIEW-2026-09-21.md).
The published desktop release remains 0.2.1; the upgrade-fix candidate is 0.2.2.
Wails remains beta.20.

## Product

One APK contains the interface and network core; no separate Tailscale app, root,
server component or administration API key is needed. Supports Android 8+ on
ARM64; x86_64 is included for emulator testing. UI defaults to Chinese and has
an English setting. Network and account management reuse the desktop model.

Add a control URL such as `https://tailsc.example.com` (not `/admin`). Login opens
the system browser and follows the server's OIDC/registration flow. Returning
to the application refreshes status. Disconnect keeps the account; logout
removes its local login and may require authentication/approval next time.

Android asks for VPN permission on first connect/login. The ongoing notification
provides disconnect. Closing the screen does not stop the service. Starting
another VPN revokes this VPN; we must show stopped and never reclaim it silently.
Always-on, boot startup and lockdown mode are not enabled in this preview.

## Honor and Clash Meta

Initial target: the user's Honor phone, exact model and MagicOS version unknown.
Clash Meta is used in proxy mode, not TUN/VPN. This can coexist with our VPN;
applications explicitly using the proxy still follow its configured routing.
Two VpnService-based TUNs cannot own the same Android user/profile concurrently.
We do not alter Clash configuration or exclude arbitrary packages automatically.

If MagicOS terminates VPN after screen lock, check system Settings → Applications
or Battery → App launch management, allow manual/background running for
HeadscaleClient. Menu names differ across MagicOS versions. Do not disable
system battery controls globally. The app does not demand an exemption upfront.

## Build

Architecture and dependency decisions: [ADR 0007](../adr/0007-android-vpn-host.md).

Prerequisites: Go 1.26.5, Node 24, pnpm 11.21.0, JDK 17, Gradle 8.13,
Android SDK 35 / Build Tools 35.0.0 / NDK 28.2.13676358. Set ANDROID_HOME
and ANDROID_NDK_HOME. Build the frontend, then:

```sh
pnpm --dir frontend install --frozen-lockfile
pnpm --dir frontend build
bash mobile/android/build-preview.sh
```

On Windows, use the Dockerfile in mobile/android or the Android preview GitHub
Actions workflow. The latter stores an APK as a build artifact without creating
a GitHub Release. Output: `bin/android-preview/headscaleclient-0.2.1-android.2.apk`.
Distributed previews from android.2 use a persisted PKCS12 signing identity in
GitHub Actions secrets. The application ID remains `com.bimcc.headscaleclient.preview`
and versionCode increases for every delivery, so updates replace the app and retain
data. Never add a version suffix to the application ID or regenerate its key.
The original android.1 used an ephemeral debug signer: it cannot be updated with
the new identity. A one-time uninstall/reinstall may be required and removes local
accounts/settings. This package is still debuggable and is not a production release.

Signing secrets: `HEADSCALE_ANDROID_KEYSTORE_BASE64` and
`HEADSCALE_ANDROID_KEY_PASSWORD`. A local encrypted backup resides outside Git in
`%LOCALAPPDATA%/BIMCC/Signing/Android`; `password.dpapi` is readable only by the
generating Windows user. Back up the keystore AND its password securely before
moving computers. CI must fail instead of silently falling back to a new debug key.

Android.2 fixes the invalid LocalAPI notification mask shared with desktop clients,
resizes the WebView for system bars and the IME, keeps server forms scrollable,
and maps structured runtime errors to the selected interface language. Emulator
acceptance includes an actual soft keyboard and an unauthenticated official
Tailscale login URL request; successful account authentication still requires a user.

Local builds recompile the Go core. Only CI opts into a cached AAR after checking
its key against the native sources and dependency locks. Desktop Wails dependencies
stay pinned; the Android host uses no Wails Activity lifecycle.

## Acceptance gate

- Build ARM64 and x86_64 with full network core; verify APK signature and licenses.
- Emulator: cold start, actual native status, UI lifecycle, no demo fallback;
  reject external/iframe privileged messages; permission denied and VPN revoke.
- Honor: connect to Headscale/OIDC and official Tailscale, return from browser,
  ping a permitted peer; wrong endpoint fails visibly; repeat logout/login.
- Lock screen for 30 minutes, close/reopen UI, switch Wi-Fi/mobile data, verify
  route recovery, memory/CPU and battery use; repeat with Clash Meta proxy mode.
- Verify local LAN/internet routes, no implicit exit node/subnet routes, and
  correct handling of an intentionally selected exit node.
- Record physical device model, Android/MagicOS version and measured results.

Passing a build does not demonstrate phone background reliability. No real-phone
acceptance is claimed until these tests are recorded.
