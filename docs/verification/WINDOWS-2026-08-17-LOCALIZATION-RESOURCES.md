# Windows Localization and Resource Verification, 2026-08-17

## Scope

- Host: Windows AMD64, 12 logical processors.
- Wails: `v3.0.0-beta.8`.
- Tailscale module and bundled daemon: `v1.102.2`.
- Product language values: `zh-CN` and `en-US`; default `zh-CN`.

## Automated checks

- `go test ./...`: passed.
- `go vet ./...`: passed.
- Frontend Vitest: 52 tests passed.
- Frontend TypeScript and production build: passed.
- Generated binding surface: 1 service, 14 methods, 19 models, 5 events.
- Chinese default, immediate English switching, `document.lang`, and English
  tray projection have dedicated tests.

## Responsive checks

The production frontend was inspected in the local browser preview.

| Viewport | Language | Result |
| --- | --- | --- |
| 960 x 680 | Chinese | No document or control overflow |
| 960 x 680 | English | No document or control overflow |
| 390 x 844 | English | Settings, networks, and devices have no document or control overflow |

The narrow control-server selector retains its intentional horizontal list
scroll. The page itself does not scroll horizontally.

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
| Production frontend JavaScript | 246.02 kB |
| Production frontend JavaScript, gzip | 76.39 kB |
| Native executable | 15.61 MiB |
| Windows installer | 23.77 MiB |

This is a single idle sample on one development machine, not a cross-platform
performance guarantee. Login, peer churn, active transfer, DERP traffic, and
daemon resource usage were not included.

## Package

- Artifact: `bin/headscaleclient-amd64-installer.exe`
- SHA-256: `26EB95680F1507526BCD91EB94E53FE634EFB2D9D71CB4B7831EC0DD4710A89E`
- Installer company metadata: `BIMCC., Ltd.`
- Installer copyright: `(c) 2026 BIMCC., Ltd.`
- In-app attribution: `Powered by BIMCC., Ltd.`
