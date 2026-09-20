# ADR 0007: Android preview host and VPN lifetime

Date: 2026-09-20. Status: accepted for technical preview; phone acceptance pending.

## Decision

Desktop remains Wails v3.0.0-beta.20. Android uses a native Java Activity,
WebViewAssetLoader, and foreground VpnService, with one gomobile Go runtime.
The service and Application own the network backend, independently of Activity
creation/destruction. The Wails beta.20 Android template calls native shutdown
from Activity.onDestroy, which does not meet this lifetime requirement without
maintaining a custom host anyway. Do not load two Go runtimes in one process.

Reuse the React UI and internal application/config/domain/Tailscale adapter.
The isolated mobile Go module pins tailscale-android commit
`0b3c1bdb207a01e2ec226ad91bd127c295fa5c4b` (upstream Android 1.102.2)
and tailscale.com v1.102.2. Upstream handles WireGuard, TUN reconfiguration,
socket protection, route subtraction and DNS integration. No root, external
tailscaled executable, desktop service manager or Headscale administration key
is required. The APK contains its network core.

## Trust and lifecycle

- Browser login stays in the system browser. Headscale decides whether that
  means OIDC, registration approval or another authentication mechanism.
- Only bundled HTTPS-origin assets can call the named application-method bridge;
  no raw LocalAPI, arbitrary network endpoint or Java reflection bridge.
- VPN permission requires an explicit Android system confirmation. Revocation
  clears desired connection and tears down the service; never compete with a
  different VPN for ownership. Connection status also reflects established TUN.
- Identity state uses Android Keystore AES-GCM; backups are disabled. Network
  settings stay app-private. Remote diagnostic log upload defaults off.
- Closing/recreating the Activity does not stop VPN. A foreground notification
  makes the running service visible and provides disconnect. Process recreation
  resumes only an explicitly desired connection with VPN permission intact.
- No implicit exit node or subnet routes. Existing safe preference defaults apply.

## Compatibility and acceptance

Android 8+ (API 26), ARM64 phone and x86_64 emulator initially. Honor is the
first physical-phone target. The user uses Clash Meta in proxy mode: coexistence
is supported in principle; application-specific proxy routing still matters.
Two VPN/TUN providers cannot simultaneously own the same Android user/profile.
Do not silently add Clash package exclusions or alter Clash settings.

Require real-device browser/OIDC return, screen lock, background survival,
Wi-Fi/cellular handover, revoke, reconnect and Clash proxy tests before calling
the Android edition stable. MagicOS battery controls vary by model/version;
document settings without automatically requesting a battery exemption.

The preview uses its own application ID and debug signing. Desktop stable
releases and server ACL/OIDC settings are unaffected. Store submission, release
signing, auto-update, boot-start UI and Android Taildrop are later work.
