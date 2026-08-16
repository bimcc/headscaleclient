# ADR-0002: Attach to upstream tailscaled through client/local

- Status: Accepted
- Date: 2026-08-15

## Context

Reimplementing the Tailscale data plane or its platform transports would add
substantial security and compatibility risk. Directly hand-coding LocalAPI in
the UI would duplicate named-pipe, Unix-socket, macOS authentication, capability,
and JSON behavior.

## Decision

Keep upstream `tailscaled` unchanged and attach through
`tailscale.com/client/local`, pinned initially to `v1.102.2`. The application
uses the zero value of `local.Client` so upstream platform transport selection,
same-user proof, and capability headers remain authoritative.

LocalAPI is hidden behind HeadscaleClient-owned interfaces and DTOs. Stable
upstream methods are preferred; unavoidable unstable methods such as IPN watch,
Start, and profile listing remain confined to one adapter.

## Consequences

- The MVP requires a separately installed and running daemon.
- The GUI does not need global elevation.
- Tailscale module updates are explicit compatibility work.
- Headscale and official Tailscale share the same local client path through `ControlURL`.
- Bundled daemon installation may be added later as a separate platform feature.

