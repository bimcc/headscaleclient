# HeadscaleClient

HeadscaleClient is an independent cross-platform desktop client for
Tailscale-compatible networks. Release packages manage a pinned upstream
`tailscaled` service and support both Headscale and the official Tailscale
control plane without requiring the official Tailscale GUI.

The project is intentionally a GUI and application layer. WireGuard, DERP,
NAT traversal, routing, and DNS remain owned by the upstream daemon.

## Current baseline

- Wails `v3.0.0-beta.8`
- Go `1.26.5`
- `tailscale.com` `v1.102.2`
- React + TypeScript
- Chinese and English interface; Chinese is the default
- Windows, macOS, and Linux desktop

## Development prerequisites

- Go `1.26.5`
- Node.js `24`
- pnpm `11.21.0`
- Wails CLI `v3.0.0-beta.8`
- An existing `tailscaled` daemon for live LocalAPI development

Install the pinned Wails CLI and pnpm versions with:

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.8
npm install --global pnpm@11.21.0
```

Platform build requirements:

- Windows: the WebView2 Runtime (included with current Windows releases).
- macOS: Xcode Command Line Tools.
- Ubuntu 24.04: `build-essential`, `pkg-config`, `libgtk-4-dev`, and
  `libwebkitgtk-6.0-dev`.

On Ubuntu, install the GUI build dependencies with:

```sh
sudo apt-get update
sudo apt-get install --yes build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev
```

The native development binary reuses the host daemon. The full Windows machine
installer and Linux DEB/RPM/Arch packages include and manage the verified
upstream service when one is not already installed. The GUI-only Linux AppImage
continues to require an external daemon.

## Development

Install dependencies, generate type-safe Wails bindings, and start the desktop
development process:

```sh
go mod download
go tool wails3 generate bindings -clean=true -ts -i
pnpm --dir frontend install --frozen-lockfile
pnpm --dir frontend build
go tool wails3 dev -config ./build/config.yml -port 9245
```

The repository keeps an empty `frontend/dist` directory so bindings can be
generated before the first frontend build on a clean checkout.

## Verification

Run backend checks and frontend checks from the repository root:

```sh
go tool wails3 generate bindings -clean=true -ts -i
pnpm --dir frontend test
pnpm --dir frontend build
go test ./...
go vet ./...
```

The CI workflow runs these checks on Windows, macOS, and Ubuntu and also
asserts the pinned Wails and `tailscale.com` module versions.

## Build

Build a native desktop binary for the current platform:

```sh
go tool wails3 build
```

Build output is written under `bin/`. Run `go tool wails3 doctor` when platform
tooling or GUI libraries are not detected.

Prepare the verified daemon payload and create the Windows machine installer:

```powershell
go tool wails3 task package
```

This produces `bin/headscaleclient-amd64-installer.exe`. The installer reuses a
compatible existing `Tailscale` service or installs the bundled service on a
fresh machine. Generated daemon files are not committed to the repository.

On Linux, create service-bearing packages for the selected architecture:

```sh
go tool wails3 task linux:package ARCH=amd64
```

The native packages install the GUI, pinned daemon, diagnostic CLI, provenance,
license, and `headscaleclient-tailscaled.service`. They preserve an existing
`tailscaled.service`; only the HeadscaleClient-owned unit is disabled on
uninstall. `linux:package:gui-only` creates an AppImage without a daemon.

## Repository layout

- [`frontend/`](frontend/) contains the React and TypeScript interface.
- [`internal/application/`](internal/application/) coordinates use cases and
  Wails-facing data transfer objects.
- [`internal/domain/`](internal/domain/) contains platform-neutral state and
  validation types.
- [`internal/tailscale/`](internal/tailscale/) isolates upstream LocalAPI and
  `tailscale.com` integration.
- [`internal/daemon/`](internal/daemon/) owns fixed, platform-specific service
  inspection and repair operations.
- [`internal/config/`](internal/config/) owns local configuration persistence.
- [`build/`](build/) contains Wails and platform build assets.
- [`docs/`](docs/) is the architecture, product, security, and delivery record.
- [`.github/workflows/ci.yml`](.github/workflows/ci.yml) defines continuous
  integration.
- [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md) records direct dependency
  licenses and upstream sources.

## Documentation

Start at [docs/README.md](docs/README.md).

- [Product scope](docs/product/PRODUCT.md)
- [UI principles](docs/product/UI.md)
- [Architecture](docs/architecture/ARCHITECTURE.md)
- [Technical design](docs/architecture/TECHNICAL-DESIGN.md)
- [Security model](docs/architecture/SECURITY.md)
- [Roadmap](docs/ROADMAP.md)
- [Task list](docs/TASKS.md)
- [Windows verification](docs/verification/WINDOWS-2026-08-15.md)
- [Linux packaging verification](docs/verification/LINUX-PACKAGING-2026-08-16.md)
- [Managed daemon decision](docs/adr/0003-managed-daemon-distribution.md)

## Status

The Windows MVP and unsigned independent machine installer have passed native
and isolated install/uninstall smoke verification. Service-bearing Linux package
implementation and supply-chain verification are complete; installation on a
real systemd desktop remains to be verified. Signing, fresh-machine testing,
Headscale login completion, and independent macOS distribution remain release
work. The authoritative state is maintained in [docs/TASKS.md](docs/TASKS.md).
