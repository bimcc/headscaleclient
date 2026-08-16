# ADR-0001: Use Wails 3 for the desktop application

- Status: Accepted
- Date: 2026-08-15

## Context

The application needs a modern cross-platform UI while its most reliable
LocalAPI client is already implemented in Go. Tauri would introduce a Rust host
plus a Go sidecar. Avalonia would introduce C# DTOs or the same sidecar boundary.

## Decision

Use Wails `v3.0.0-beta.8` with React and TypeScript. The Go application imports
the official Tailscale Go module directly, and Wails provides generated
frontend bindings, event delivery, native windows, tray, single-instance,
autostart, and packaging facilities.

## Consequences

- The main application is one Go process plus the system `tailscaled` service.
- The source stack is Go and TypeScript rather than Go, Rust, and TypeScript.
- Wails 3 beta API churn is a real delivery risk.
- All Wails-specific code is isolated in the composition root and `desktop` package.
- The exact beta version is pinned; upgrades require an ADR review note and full build matrix.
- If Wails 3 blocks release quality, Tauri 2 plus the existing Go application layer is the fallback.

