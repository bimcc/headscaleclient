# Managed Daemon Payload

HeadscaleClient release packages may include an unmodified, pinned upstream
`tailscaled` service. Generated binaries are never committed to the repository.

`manifest.json` is the source of truth for the upstream version, download URL,
source-archive hashes, and extracted runtime-file hashes. The Windows
preparation script also requires valid Authenticode signatures from Tailscale
Inc. and WireGuard LLC before it writes a package payload. The Linux preparer
requires the expected Tailscale command paths and Go module version.

Prepare a Windows payload from the repository root:

```powershell
go tool wails3 task windows:prepare:daemon ARCH=amd64
```

The output is written to `bin/daemon/windows-<arch>/` and consumed by the NSIS
machine installer. It contains `tailscaled.exe`, the diagnostic `tailscale.exe`
CLI, `wintun.dll`, provenance, and the applicable license notices.

An existing `Tailscale` Windows service is reused and is never replaced by the
installer. A fresh machine receives the bundled service. Uninstall removes the
service only when its configured executable path is inside the same
HeadscaleClient installation directory.

Prepare a Linux payload from any supported build host:

```sh
go tool wails3 task linux:prepare:daemon ARCH=amd64
```

The output is written to `bin/daemon/linux-<arch>/` and consumed by the DEB,
RPM, and Arch package definitions. Native Linux packages install a dedicated
systemd unit only for the HeadscaleClient payload and preserve a standard
external `tailscaled.service`. The AppImage is intentionally GUI-only.
