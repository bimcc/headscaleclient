# ADR-0006: Restore the previous Wails beta.8 baseline

- Status: Accepted
- Date: 2026-09-20

## Context

The user reported Defender detecting the beta.23-based 0.2.0 Windows installer
as `Trojan:Win32/Wacatac.B!ml`, while their old beta.8 installer was accepted.
Binary comparison found identical network-service payloads and installer code;
the changed GUI links Wails beta.23 instead of beta.8. This narrows the
investigation but does not establish Wails as the detection cause.

The user explicitly requested restoring the previous Wails version.

## Decision

Restore Wails Go module and CLI to `v3.0.0-beta.8`, and the frontend runtime to
`3.0.0-beta.8`. Regenerate bindings with the matching CLI. Restore diagnostics,
development instructions and CI pins together, and check their agreement as
part of the existing version consistency script.

Keep Go 1.26.5, Tailscale 1.102.2 and application behavior, including current
device UI and unified macOS installer implementation. Application version stays
0.2.0 for this comparison; name the local installer explicitly with `wails-beta8`
to distinguish it from the existing published beta.23 artifact.

## Consequences

- This supersedes the beta.23 upgrade decision for the current source baseline.
  Historical release and verification documents continue to describe their
  original artifacts accurately.
- Passing compile and functional checks is not evidence of Defender acceptance.
  That requires scanning the actual rollback artifact on the affected system.
- Do not replace the already published 0.2.0 asset under the same filename/hash.
  A subsequent public release needs its own reviewed version and verification.
- Framework fixes introduced after beta.8 are also rolled back. Native Mac/Linux
  acceptance must be repeated before distributing new packages for those hosts.

## Follow-up comparison

The user reports that the beta.8 rollback no longer triggers the detection and
requests testing beta.20. The working tree advances to beta.20 for that local
comparison; beta.8 remains the retained fallback artifact. This does not promote
beta.20 to a verified public release. See
[beta.20 trial](../verification/WAILS-BETA20-TRIAL-2026-09-20.md).

The user subsequently accepted beta.20 and requested publication. Release
0.2.1 therefore selects beta.20, keeping all runtime and CLI pins aligned and
retaining both comparison artifacts. This supersedes beta.8 as the current
source baseline. Final release evidence is recorded separately in
[0.2.1 verification](../verification/RELEASE-0.2.1.md).
