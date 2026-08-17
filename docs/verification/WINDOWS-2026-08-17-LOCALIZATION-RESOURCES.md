# Windows Localization and Resource Verification, 2026-08-17

## Scope

- Host: Windows AMD64, 12 logical processors.
- Wails: `v3.0.0-beta.8`.
- Tailscale module and bundled daemon: `v1.102.2`.
- Product language values: `zh-CN` and `en-US`; Windows first launch follows
  the installer choice, with `zh-CN` as the portable and invalid-value fallback.

## Automated checks

- `go test ./...`: passed.
- `go vet ./...`: passed.
- Frontend Vitest: 57 tests passed.
- Frontend TypeScript and production build: passed.
- Generated binding surface: 1 service, 14 methods, 19 models, 5 events.
- Chinese default, immediate English switching, `document.lang`, and English
  tray projection have dedicated tests.
- Store tests verify that an English installer default initializes missing and
  legacy language fields while an explicit saved Chinese preference wins.
- Route-probe tests verify Disco-only probing, relay-to-direct convergence,
  bounded relay attempts, unknown metadata handling, and refreshed snapshot
  publication. The frontend verifies that missing route evidence is never
  labelled Direct.
- A live sanitized comparison found an online peer with no `CurAddr` and relay
  region `hkg`. Three official CLI Disco probes all reported `DERP(hkg)` and
  ended with direct connection not established, confirming that the former
  TSMP-derived Direct label was false.

## Responsive checks

The production frontend was inspected in the local browser preview.

| Viewport | Language | Result |
| --- | --- | --- |
| 960 x 680 | Chinese | No document or control overflow |
| 960 x 680 | English | No document or control overflow |
| 390 x 844 | Chinese | Overview, networks, Settings, and About have no document or control overflow |
| 390 x 844 | English | Overview, Settings, and About have no document or control overflow |

The narrow control-server selector retains its intentional horizontal list
scroll. The page itself does not scroll horizontally.

The review also confirmed that Overview uses a compact online-device count
instead of a duplicated peer list, LAN access is disabled until an exit node is
selected, and the current server/account scope is visible. The nested LAN row
uses a smaller `13px` title, `11.5px` description, L-shaped connector, and
muted disabled state. It had no horizontal overflow in desktop or narrow
browser checks. The selected-server
detail places current-network and reachability badges beside the name, removes
the former status strip, and places its account count in the Accounts header.
Settings has General and Runtime & diagnostics groups; product and upstream
attribution appears only in the bilingual About view. The five navigation
labels and the mobile device icon plus online count fit at 390 px.

The Devices view calls status-derived routing `Recent path`. Its Ping action
reports an independently measured `Probe result`; successful probes publish a
new snapshot so row, drawer, and tray paths update together. Online peers with
no direct or relay evidence display Path unknown instead of Offline.
Chinese and English device details were checked at `960 x 680` and `390 x 844`;
the recent-path label and a DERP probe result remained inside the drawer with no
document overflow.

## Resource review

Static review confirmed that the frontend does not poll. The application owns
one resilient LocalAPI watcher; tray and WebView consume the same snapshots.
Toast expiry, account-menu focus, and bounded daemon service-state waits are
short-lived operations rather than background polling. The tray now compares
its projected render state and skips `SetMenu` when snapshot, language, busy,
and error state are unchanged. Five unused public scaffold assets and their
unused font license were removed; Vite no longer copies about 686 KiB of
unreferenced runtime assets into the embedded frontend.

One freshly built native process was started without installation, allowed to
settle for eight seconds, sampled over the next ten seconds, and then stopped:

| Metric | Sample |
| --- | ---: |
| Working set | 29.5 MiB |
| Private memory | 54.5 MiB |
| CPU delta over 10 seconds | 0.00% of one core |
| Production frontend JavaScript | 249.03 kB |
| Production frontend JavaScript, gzip | 77.43 kB |
| Native executable | 15.63 MiB |
| Windows installer | 23.78 MiB |

This is a single idle sample on one development machine, not a cross-platform
performance guarantee. Login, peer churn, active transfer, DERP traffic, and
daemon resource usage were not included.

## Package

- Artifact: `bin/headscaleclient-amd64-installer.exe`
- SHA-256: `2F5854B2F446D7AA3F7BAD2F756EB2774208DE0979B1480226175B32AF175FC6`
- Default machine installation directory: `C:\Program Files\BIMCC\HeadscaleClient`
- Installer company metadata: `BIMCC., Ltd.`
- Installer copyright: `(c) 2026 BIMCC., Ltd.`
- In-app publisher attribution: About view, `BIMCC., Ltd.`
- NSIS compile result: four installer pages, two installer language tables, and
  two uninstaller language tables.
- Setup always presents a bilingual Simplified Chinese/English selector. The
  selected locale controls built-in pages and custom service messages, writes
  `DefaultLanguage` for first launch, and leaves the finish-page run checkbox
  checked by default.
- `headscaleclient.exe` and the installer both report `NotSigned`. The release
  task now signs the GUI before packaging and the outer installer afterward,
  but a publicly trusted Authenticode certificate issued to BIMCC is still
  required before a trusted artifact can be produced.
