# Network and privacy disclosures

HeadscaleClient is a local GUI for a Tailscale network core. BIMCC does not require
an administration API key, operate a mandatory account service, or add advertising
or analytics SDKs to the interface.

The client/core communicates with the chosen control server, its authentication
provider, peers, DNS, and configured DERP/STUN services to establish and maintain
network access. Servers can receive device names, addresses, public keys and other
coordination metadata. Relay operators forward encrypted tunnel traffic and can
observe connection metadata; the control server and ACL configuration determine
which peers are visible/permitted.

Bundled/reused upstream software has its own operational logging and service
behavior. This project has not certified that every upstream diagnostic upload is
disabled. Do not promise that no third-party network communication occurs. Consult
[Tailscale's privacy policy](https://tailscale.com/privacy-policy), the selected
Headscale operator and any OIDC identity provider. Browser authentication may set
cookies in the external browser. It is not a BIMCC-hosted login page.

Local account settings persist across upgrades. Android core identities are
encrypted using Android Keystore and excluded from backup/device transfer. Windows
uses upstream service storage and the local user's application settings. Removing
the Android app deletes its local accounts/configuration. Disconnecting retains
identity; explicitly removing an identity requires authentication again.

Windows installation can install or reuse the Tailscale system service and
requires administrative permission. Android requires the system's VPN permission
and shows a foreground-service notification. Exit nodes, subnet routes and managed
DNS can change network behavior and must remain explicit preferences. Uninstall
through Windows Installed Apps or Android App info; upstream externally managed
Tailscale services are not removed by our installer.

SignPath application review still requires the maintainers to confirm these
disclosures against the final release, upstream telemetry defaults and actual
installer notices; this document is not a completed Foundation compliance review.
