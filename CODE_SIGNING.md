# Code signing policy

Status (2026-09-21): SignPath integration is prepared, but **no SignPath Foundation
approval or publicly trusted Windows certificate has been obtained**. The project
must not describe its current releases as signed by SignPath.

## Windows

Self-signed certificates are for internal signature/integrity tests. They do not
remove Windows unknown-publisher warnings on ordinary computers and do not prevent
antivirus detections. Never ask ordinary users to disable antivirus or install our
test certificate as a trusted root. `tools/signing/new-windows-test-certificate.ps1`
creates a non-exportable private key in the current Windows user's personal store
and exports only its public certificate. It does not modify system trust stores.

The planned public signing provider is [SignPath Foundation](https://signpath.org/).
Their [terms](https://signpath.org/terms) and [application](https://signpath.org/apply)
must be reviewed by the repository owner. Open-source status is necessary but does
not guarantee admission. Their certificate names **SignPath Foundation** as publisher,
not BIMCC. The application can continue to identify BIMCC as its developer/publisher
in product metadata and About.

Before enabling `.github/workflows/windows-signpath.yml`:

1. The repository owner supplies contact details and establishes the real authors,
   reviewers and signing approvers; do not invent team membership. All need MFA.
2. SignPath reviews the project, licenses, release history and build provenance.
   In particular, our bundled `wintun.dll` has WireGuard's **Prebuilt Binaries
   License**, not the MIT license of this repository. The WebView2 bootstrapper is
   another separately licensed component. Foundation rules generally prohibit
   proprietary components except qualifying System Libraries; request an explicit
   determination for this packaging. Do not assume repository MIT licensing makes
   the complete installer eligible or replace signed drivers to bypass this review.
3. Install the SignPath GitHub integration and configure approved application and
   installer artifact policies. Restrict the product name to HeadscaleClient and
   enforce consistent version metadata. Only our GUI and installer are submitted;
   bundled Tailscale/Wintun binaries keep their upstream signatures and licenses.
4. Configure `windows-code-signing` environment with protected branches and required
   maintainers, `SIGNPATH_API_TOKEN`, and repository/environment variables
   `SIGNPATH_ORGANIZATION_ID`, `SIGNPATH_PROJECT_SLUG`, `SIGNPATH_SIGNING_POLICY_SLUG`,
   `SIGNPATH_APPLICATION_CONFIGURATION`, `SIGNPATH_INSTALLER_CONFIGURATION`.
5. Set `HEADSCALE_SIGNPATH_ENABLED=true` only after review. Manually dispatch on the
   approved commit; the application and installer requests each wait for approval.
   Validate signer/timestamp, package contents and an actual installation before release.
   The current template does not separately sign the NSIS-generated uninstaller;
   that coverage must be explicitly reviewed with SignPath.
6. After acceptance, replace the pending status with the required attribution:
   “Free code signing provided by SignPath.io, certificate by SignPath Foundation”,
   link the actual named team roles, and include this policy on release pages.

The workflow is a prepared integration, not a completed SignPath signing test. It
does not automatically publish releases or fall back to unsigned output on failure.

## Android

Android updates use a self-signed application certificate, which is the standard
APK signing model. Store the same application ID and signing key across versions;
increase versionCode. Never use a version-specific package ID to bypass a signature
mismatch. Android.2 establishes a persisted signing key (SHA-256 fingerprint
`11204724BB01854AC87127702E8D2C44A9712F1EB5F2D84FA40036A76B191253`).
The android.1 ephemeral debug signer cannot be recovered from its public APK, so
that first transition requires reinstalling and recreating local login state.

The encrypted key and certificate are retained outside Git at
`%LOCALAPPDATA%/BIMCC/Signing/Android`; GitHub Actions holds the keystore and password
as separate secrets. The local DPAPI password copy is tied to its creating Windows
user. The owner must retain an offline secure backup of both key and password.
Signing does not turn this debuggable technical preview into a stable release.

## Privacy and system changes

See [network and privacy disclosures](docs/product/PRIVACY.md). Installer and VPN
permission dialogs must describe their system changes. A public certificate is
not a guarantee of network privacy, absence of bugs, or antivirus acceptance.
