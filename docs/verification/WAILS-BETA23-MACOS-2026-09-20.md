# Wails beta.23 migration and macOS preview

## Scope

- Upgrade Go module, pinned CLI, frontend runtime and diagnostics from Wails
  `v3.0.0-beta.8` to `v3.0.0-beta.23`; regenerate TypeScript bindings.
- Keep Go `1.26.5` and Tailscale `1.102.2` unchanged.
- Add independent macOS launchd/utun packages, separate from GUI-only archives.
- Supersedes the version pin in ADR 0001; its original decision is preserved.

## Upstream compatibility review

- [beta.20](https://github.com/wailsapp/wails/releases/tag/v3.0.0-beta.20)
  replaces the embedded native WebView2 loader / go-winloader with a pure-Go
  implementation. This does **not** remove Microsoft's WebView2 Runtime
  requirement. Keep the existing NSIS runtime detection/bootstrap. The old
  `native_webview2loader` tag is now a no-op; this product does not use it.
- beta.19 gates private macOS APIs behind `private_mac_apis`. This product uses
  public window/menu/tray APIs and does not enable that opt-in tag. The beta.20
  documentation removals do not remove macOS support.
- beta.16 fixes macOS tray click behavior; beta.18 fixes Linux/Darwin allocation
  leaks; beta.21 fixes Windows menu replacement resources and bindings.
- [beta.23](https://github.com/wailsapp/wails/releases/tag/v3.0.0-beta.23)
  includes Linux app-ID and single-instance fixes. No routing, ACL, DNS or
  WireGuard implementation is replaced by this GUI-framework migration.
- Customized Windows installer/build tasks are retained rather than overwritten
  with a fresh framework template. Existing login/account/device semantics stay
  unchanged.

## Verification

In progress: Windows backend/frontend checks and installer rebuild; native macOS
Apple Silicon/Intel builds, install/upgrade/removal and normal-user LocalAPI
smoke tests. Results will be recorded after the actual runs.

## Release limitations

macOS is an unsigned-installer/ad-hoc-app-signed preview, not a notarized mature
release. Real Headscale/OIDC login, sleep/wake, network changes and downloaded
package Gatekeeper behavior still require real-Mac acceptance. Windows remains
unsigned without a public certificate. See [macOS guide](../product/MACOS.md).
