# Release 0.2.1 verification

Status: published and verified on 2026-09-20 as the latest stable GitHub release.

[Download HeadscaleClient v0.2.1](https://github.com/bimcc/headscaleclient/releases/tag/v0.2.1).

## Scope

The user reports that both beta.8 and beta.20 Windows comparison packages no
longer trigger the detection reported for beta.23, and explicitly requests a
new GitHub release using beta.20. Pin the Go module, CLI, frontend runtime and
diagnostics together. Keep Go 1.26.5, Tailscale 1.102.2 and application behavior.

Synchronized product/package metadata to 0.2.1. Built Windows AMD64 locally and
macOS ARM64/AMD64 plus Linux AMD64 DEB on native CI hosts. Published a new tag
and stable release with four installers and SHA256SUMS, preserving old release
assets.

Release source: `74b3b5fe14fc9c36c787f8d5d3055845c146cac2`.

## Verification

- Local version consistency, Go test/vet and 65 frontend tests passed.
- Windows production frontend, binding generation and NSIS packaging passed.
  Packaged GUI hash matches the compiled output. Both GUI and installer report
  display version `0.2.1`, numeric version `0.2.1.0`, company `BIMCC., Ltd.` and
  product `HeadscaleClient`.
- `go version -m` on the extracted Windows GUI confirms Go 1.26.5, Wails
  `v3.0.0-beta.20` and Tailscale `v1.102.2`. The GUI SHA-256 is
  `c4617a2f5cb4eff8b9486147b1801231649d0fdb42308b036bf63cb3740f1ce0`.
- The extracted daemon/CLI/Wintun hashes match the pinned manifest. Windows
  installer retains 15 payload files, NSIS 3 Unicode with non-solid Deflate,
  1011 install instructions, 112 uninstall instructions and two languages.
- [Cross-platform CI](https://github.com/bimcc/headscaleclient/actions/runs/35510775198)
  passed on Windows, macOS and Ubuntu 24.04 against the release source above.
  All three passed metadata/module checks, 65 frontend tests, Go test/vet and
  native GUI builds. Windows checked compiled PE metadata. Ubuntu also built
  the service-bearing DEB and checked its version and required contents.
- Downloaded the Linux artifact from that CI run and independently checked its
  control metadata: `headscaleclient`, version `0.2.1-1`, architecture `amd64`.
- [Mac package CI](https://github.com/bimcc/headscaleclient/actions/runs/35510775237)
  passed on macOS 15 ARM64 and Intel. Both ran installer policy tests, Go
  test/vet, 65 frontend tests, signature/architecture checks and native package
  lifecycle validation: fresh install, normal-user LocalAPI, service restart,
  GUI launch, reinstall, service removal, conflicting service rejection,
  live/stopped external-service reuse and explicit migration. Both upgraded
  from the hash-pinned 0.1.0 split-package preview, checked the installed app
  and receipt versions, compared installed/build executable bytes and retained
  the state marker. External fixtures used an upstream userspace daemon and
  official-app directory fixture, not the signed official Network Extension.
- Downloaded both Mac packages and matched their original CI SHA-256 files.
  Release filename changes preserve each package's exact bytes.

## Artifacts

| File | Bytes | SHA-256 |
| --- | ---: | --- |
| `headscaleclient-0.2.1-windows-amd64-installer.exe` | 24991306 | `589daf0e54c2838ff7fe4b455820a34ec065ab7ca64c4492fab76d3656af36ef` |
| `headscaleclient-0.2.1-macos-arm64-installer.pkg` | 22402728 | `e0670d9f839df60a50743ab27313e10719253c7c0ca853b1eae090308ea282b1` |
| `headscaleclient-0.2.1-macos-amd64-installer.pkg` | 24583723 | `718de183053b7ee21d16dab10c529326348bee8773f19d688d7c39797327fa27` |
| `headscaleclient-0.2.1-linux-amd64.deb` | 45041036 | `994a41d1a83e0d1135d1cad816731623e172e1ce9e36a55ebb2163a245c5de3a` |

The release also includes `SHA256SUMS.txt` covering all four installers.
Its SHA-256 is `0e32da15f03c2c8e03cacb4300b1264e964d015f0a22883eaaee0a3675d80331`.
All five uploaded assets reported `uploaded` and matched the local byte sizes
and GitHub SHA-256 digests before publication. Published at
`2026-09-20T12:53:10Z`, then independently verified `/releases/latest` points to
`v0.2.1` with draft/prerelease both false. The release tag resolves directly to
the tested source commit above. All five existing 0.2.0 assets were rechecked
against their preserved local files and retain their original sizes/digests.

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
