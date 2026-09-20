# Release 0.2.1 verification

Status: preparing the user-authorized stable release using Wails beta.20.

## Scope

The user reports that both beta.8 and beta.20 Windows comparison packages no
longer trigger the detection reported for beta.23, and explicitly requests a
new GitHub release using beta.20. Pin the Go module, CLI, frontend runtime and
diagnostics together. Keep Go 1.26.5, Tailscale 1.102.2 and application behavior.

Synchronize product/package metadata to 0.2.1. Build Windows AMD64 locally and
macOS ARM64/AMD64 plus Linux AMD64 DEB on native CI hosts. Publish a new tag and
stable release with four installers and SHA256SUMS, preserving old release
assets. Record the tested source commit, CI runs and hashes after completion.

## Verification

Checks and artifacts are in progress.

## Limits

The accepted beta.20 comparison installer used product version 0.2.0. This
release rebuild changes metadata and bytes; its final hash is different. The
development host has no usable Defender scanner, so no independent Defender
pass is claimed for the final 0.2.1 artifact. User comparison results do not
establish a universal antivirus verdict or the exact upstream detection cause.

Windows lacks trusted Authenticode signing. macOS uses ad-hoc application
signing with an unsigned, unnotarized PKG. Native CI lifecycle fixtures do not
replace real-account networking/Network Extension, sleep/roaming or Linux
real-systemd installation acceptance. The installed Windows GUI and current
network service are left running and undisturbed during release packaging.
