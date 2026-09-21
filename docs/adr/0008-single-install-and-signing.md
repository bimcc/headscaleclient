# ADR 0008: One installed product and persistent signing identity

Date: 2026-09-21. Status: accepted for the current implementation.

## Problem

Windows derived its uninstall key from the publisher display name. Renaming
HeadscaleClient Contributors to BIMCC., Ltd. produced two records and could leave
the original executable behind. The installer also did not recover the previous
custom install path. Android.1 used a per-run debug signing key, making updates
across CI machines incompatible despite having the same application ID.

## Decision

Windows uses the immutable key `io.headscaleclient.desktop`, independent of version,
publisher wording or architecture. First installs use
`C:\Program Files\BIMCC\HeadscaleClient`. Upgrades reuse the registered directory,
skip directory selection and overwrite the product after asking any running GUI to
exit. They do not create a versioned installation directory or run an old uninstaller.

The migration helper reads the two known historical machine-level uninstall keys
in both registry views, handles early binaries without version resources, and
prefers the current registration when several exist. It rejects unexpected product
metadata, broad/shared roots and path junctions. Only exact product files in old
registered directories are removed; unrelated files and account/core state remain.
Empty legacy directories may be removed. Legacy keys are removed only after the
new installation has been registered. Failure is explicit and does not silently
report a successful clean upgrade.

A Tailscale service pointing exactly to another recognized old managed directory
is stopped and removed before the new bundled service is installed. An external
Tailscale service is reused. A service in the chosen directory follows the existing
repair path. This migrates service registration, not identity storage.

Android retains `com.bimcc.headscaleclient.preview`, a monotonically increasing
versionCode and the persisted certificate starting with android.2. A signature
mismatch must fail installation; changing the package ID to permit parallel copies
is prohibited. Migration from the lost android.1 debug signer requires a one-time
reinstall and new local login. Later compatible updates retain app-local data.

Windows self-signing is an internal testing option only. Public Windows signing is
prepared for SignPath Foundation but requires project acceptance and maintainer
approval. Signatures, update identity and malware detection are distinct concerns.
See [Code signing policy](../../CODE_SIGNING.md).

## Verification

Local migration tests cover preferred/current registration, stale records,
explicit-file cleanup, retained user files, current-directory preservation and
unsafe path rejection. Disposable Windows CI installs over two legacy registrations,
checks one registration and executable, reruns setup in place, verifies the managed
service and uninstalls. It refuses to run on a machine with pre-existing product
records or a Tailscale service. The developer's installed network is not touched.

Android CI checks APK signatures and replacement installation retaining an app data
sentinel and exactly one package ID. Real-device upgrades and Honor background
reliability remain separate acceptance checks.
