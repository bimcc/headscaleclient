# Windows tray and information-architecture verification

Date: 2026-08-16 (Asia/Shanghai)

## Scope

- Snapshot-driven native tray and hidden-window navigation.
- Functional header account switcher.
- Control-server master list with server-scoped account details.
- Active-network and active-account scope on the device page.
- Device-name and multiple-address presentation fallbacks.

## Automated evidence

- `go test ./...`: passed.
- `go vet ./...`: passed.
- `pnpm test`: 51 tests passed across 5 files.
- `pnpm build`: passed with strict TypeScript and a production Vite bundle.
- Wails bindings: 1 service, 13 methods, 19 models, and 5 events.

## Visual evidence

The production frontend was inspected with the demo snapshot at:

- `960x680`: 224 px server master list, 606 px detail pane, intentional URL
  truncation only, and no button or page overflow.
- `390x844`: horizontally scrollable server selector above the detail pane,
  no page overflow, no device-scope overlap, and the account menu remained
  inside the viewport.

Selecting the official Tailscale server displayed only its associated profile;
the Headscale profile was absent until its server was selected. The device page
displayed both the active network and active account above the live peer list.
Browser console error count was zero.

## Package

- Artifact: `bin/headscaleclient-amd64-installer.exe`
- Size: 25,460,521 bytes (24.28 MiB)
- Modified: `2026-08-16 23:26:58 +08:00`
- SHA-256: `4D30D7DB3DE6DD2623C55BFB8736BA83B1A28568403B955895D89CF202BE5220`
- Authenticode: not signed; release signing remains a tracked release task.

The installed executable observed during verification was the preceding build,
so this record does not claim that the final installer was installed or that
every native tray command was manually exercised on the final package.
