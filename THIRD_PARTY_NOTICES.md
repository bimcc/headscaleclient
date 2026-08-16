# Third-Party Notices

This document lists only the dependencies declared directly by HeadscaleClient
in `go.mod` and `frontend/package.json`. It is not a complete inventory of
transitive dependencies. The upstream license files are authoritative.

## Runtime and application dependencies

| Dependency | Version | License | Upstream |
| --- | --- | --- | --- |
| Wails v3 (`github.com/wailsapp/wails/v3`) | `v3.0.0-beta.8` | MIT | <https://github.com/wailsapp/wails> |
| Tailscale Go module (`tailscale.com`) | `v1.102.2` | BSD-3-Clause | <https://github.com/tailscale/tailscale> |
| Wails runtime (`@wailsio/runtime`) | `3.0.0-beta.8` | MIT | <https://github.com/wailsapp/wails> |
| Lucide React (`lucide-react`) | `0.468.0` | ISC | <https://github.com/lucide-icons/lucide> |
| React (`react`) | `18.3.1` | MIT | <https://github.com/facebook/react> |
| React DOM (`react-dom`) | `18.3.1` | MIT | <https://github.com/facebook/react> |

The Tailscale Go module carries the following upstream notice:

> Copyright (c) 2020 Tailscale Inc & contributors.

Lucide incorporates portions of Feather. Its upstream notice states that
those portions are copyright Cole Bemis (2013-2022), and the remaining Lucide
work is copyright Lucide Contributors (2022).

## Managed daemon payloads

The Windows machine installer may redistribute an unmodified Tailscale
`tailscaled.exe` and diagnostic `tailscale.exe` at version `1.102.2`, together
with Wintun `0.14.1`. Payload provenance and exact hashes are recorded in
`build/daemon/manifest.json`.

The installer includes the full notices at:

- `daemon/licenses/TAILSCALE-LICENSE.txt`
- `daemon/licenses/WINTUN-PREBUILT-LICENSE.txt`

Tailscale and Wintun names identify upstream components only. HeadscaleClient
is independent and is not endorsed by or affiliated with Tailscale Inc. or
WireGuard LLC.

Linux DEB/RPM/Arch packages may redistribute the unmodified Tailscale
`tailscaled` and diagnostic `tailscale` binaries at version `1.102.2`. The
archive and file hashes are recorded in `build/daemon/manifest.json`, and the
full Tailscale license is installed under
`/usr/share/doc/headscaleclient/daemon/`.

## Direct build and test dependencies

These packages are used to develop, build, type-check, or test HeadscaleClient.

| Dependency | Version | License | Upstream |
| --- | --- | --- | --- |
| Testing Library jest-dom (`@testing-library/jest-dom`) | `7.0.1` | MIT | <https://github.com/testing-library/jest-dom> |
| Testing Library React (`@testing-library/react`) | `16.3.2` | MIT | <https://github.com/testing-library/react-testing-library> |
| Testing Library user-event (`@testing-library/user-event`) | `14.6.4` | MIT | <https://github.com/testing-library/user-event> |
| Node.js type definitions (`@types/node`) | `26.2.0` | MIT | <https://github.com/DefinitelyTyped/DefinitelyTyped> |
| React type definitions (`@types/react`) | `18.3.31` | MIT | <https://github.com/DefinitelyTyped/DefinitelyTyped> |
| React DOM type definitions (`@types/react-dom`) | `18.3.7` | MIT | <https://github.com/DefinitelyTyped/DefinitelyTyped> |
| Vite React plugin (`@vitejs/plugin-react`) | `6.0.5` | MIT | <https://github.com/vitejs/vite-plugin-react> |
| jsdom (`jsdom`) | `30.0.1` | MIT | <https://github.com/jsdom/jsdom> |
| TypeScript (`typescript`) | `5.9.3` | Apache-2.0 | <https://github.com/microsoft/TypeScript> |
| Vite (`vite`) | `8.2.1` | MIT | <https://github.com/vitejs/vite> |
| Vitest (`vitest`) | `4.1.10` | MIT | <https://github.com/vitest-dev/vitest> |

Review the resolved `go.sum` and `frontend/pnpm-lock.yaml` dependency graphs
and their license texts when preparing a distributable release.
