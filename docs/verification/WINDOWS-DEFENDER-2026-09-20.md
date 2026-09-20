# Windows installer detection investigation, 2026-09-20

Status: binary comparison completed; the user subsequently reports that the
beta.8 rollback no longer triggers the detection. No local Defender scan was
possible. No replacement release or antivirus bypass was applied.

The user reports that the old installer is accepted and the new installer is
detected as `Trojan:Win32/Wacatac.B!ml`. The screenshot names the downloaded
installer and says remediation/quarantine failed. That is a reported detection,
not a verified false positive or proof of a running infection.

## Exact samples

The old sample was located at
`bin/release-0.2.0/headscaleclient-0.10-amd64-installer.exe` (one directory below
the initially supplied path). Its embedded version is 0.1.0 and its SHA-256
matches the existing v0.1.0 GitHub asset.

| Sample | Bytes | SHA-256 |
| --- | ---: | --- |
| User's old installer | 24956269 | `d51dd42ae38c0ceb7050583a1469004fd21e66f4097943a04183c5b4a9836cdb` |
| Published 0.2.0 installer | 24992575 | `f2a23b636038344e6e25079094f2a9ba7fa35be12ba64da96a10b71dbfe987c5` |
| Extracted old GUI | 16425984 | `048f78ac71da7bc45fa9be960a8447e9b1024b1e315b6b27a9f04eac9d91bde2` |
| Extracted new GUI | 16548352 | `8a96fd88fefa311443032f2568f1e4f14cc7635a01af10897d12e34b3bdd1102` |

Both installers and both GUIs are unsigned. Lack of signing is shared by the
two versions and cannot, by itself, explain their different reported results.

## Package and instruction comparison

The original samples were extracted, never installed or launched. 7-Zip 26.03
was downloaded from the upstream release linked by 7-zip.org and its download
digests checked against the upstream GitHub release metadata.

- Both installers are NSIS 3 Unicode, non-solid Deflate, with a 64512-byte stub.
  Their `.text`, `.rdata`, `.data` and `.ndata` sections are byte-identical.
  Resource sections differ, including product version information.
- Both contain exactly 15 files with no extra new payload. 13 have identical
  SHA-256 values: `tailscale.exe`, `tailscaled.exe`, `wintun.dll`, provenance and
  licenses, the Microsoft WebView2 bootstrapper, NSIS plugins and wizard assets.
- The bundled Tailscale, Wintun and WebView2 bootstrapper signatures validate
  as Tailscale Inc., WireGuard LLC and Microsoft Corporation respectively.
- Only `headscaleclient.exe` and `uninstall.exe` differ. The GUI grows by 122368
  bytes. The uninstaller retains the same size, code/data sections and entire
  decompressed NSIS instruction header; its resources differ.
- Both installer headers contain 1011 instructions. No opcodes changed.
  Only eight instruction operands differ: the GUI file timestamp and offsets
  into the compressed payload after the GUI. The string-table comparison finds
  only `0.1.0` becoming `0.2.0`. No new installation action was found.
- Repository diffs between the old release (`e38d497`) and the new build
  (`d43f9e7`) likewise show no Windows service-installation script changes;
  Windows installer source changes are version metadata/comment changes.

This materially narrows the regression investigation to the rebuilt GUI and
changed package metadata/content. It does not prove that the GUI alone is the
item Defender detects: an archive-level verdict must be tested separately.

## GUI source and dependency verification

`go version -m` on both extracted GUIs reports Go 1.26.5, `production`,
`CGO_ENABLED=0`, `windows/amd64`, `GOAMD64=v1` and trimpath. Of the linked module
versions, only Wails differs: beta.8 versus beta.23. Tailscale remains 1.102.2.
`go mod verify` passes for the current module cache.

The beta.20 loader removal is **not an established cause**. The old binary was
already built without `native_webview2loader` and did not link `go-winloader`.
The old default loader was already the pure-Go implementation. Removing an
unused native-loader build option does not establish an executable regression.

A rebuild using an independent Go compilation cache and the verified current
release source bytes reproduced the extracted new GUI exactly:
`8a96fd88fefa311443032f2568f1e4f14cc7635a01af10897d12e34b3bdd1102`.
This supports source-to-binary correspondence; it does not prove the source or
toolchain is free of vulnerabilities or explain the antivirus classification.

A separate detached checkout did not initially reproduce the same bytes:
Windows checkout line endings changed embedded HTML/CSS and `.gitkeep` bytes,
and source-byte changes can affect Go build IDs. The comparison identified this
as a reproducibility concern, not a demonstrated Defender trigger. No release
artifact was replaced with either of those diagnostic rebuilds.

## Remaining evidence needed

This development host has no usable Defender command-line scanner or running
WinDefend service. No local Defender pass/fail result is claimed. A read-only
VirusTotal lookup of the published installer hash returned no existing report;
no sample was uploaded and no independent antivirus verdict was obtained.

The next necessary experiment is scanning old/new installers, extracted GUIs
and uninstallers with the same active Defender engine/signature version. The
prepared local helper is
`bin/diagnostics/wacatac-20260920/scan-defender.ps1`.
It records hashes, engine versions, exit codes and full scanner output, uses
`-Scan -ScanType 3 -DisableRemediation`, and does not execute samples, change
exclusions, turn off protection, restore quarantined files or upload anything.
Real-time protection remains active. Missing/quarantined samples are recorded
as missing; exit code 2 is not automatically classified as a malware finding.

Interpretation requires both levels:

- New GUI detected alone: investigate the application/framework changes first.
- Only the new installer detected: investigate package/metadata classification
  with Microsoft; do not assume an embedded component is malicious.
- Both old and new are now detected: the historical old-package result is not
  a present-day negative control.
- No local detection: this does not clear a reported cloud/download-time verdict.

Any framework rollback or installer-format change needs this evidence; neither
self-signing nor repacking until a scanner accepts a file establishes a fix.

Raw local evidence: `bin/diagnostics/wacatac-20260920/comparison.json` and
`nsis-differences.json`. Original release assets are intact.

## Framework comparison follow-up

After this investigation, the user requested restoring beta.8 while retaining
current application features and the 0.2.0 product version. The resulting
installer SHA-256 is
`f1776aac2c33324f68859c1654902573add778c12297ec02998561aee283f89e`.
The user reports that this candidate no longer triggers the detection.
This strengthens the evidence of a framework-version-related difference,
without identifying the individual upstream change or detection signature.
No scanner engine/signature logs were supplied for that retest.

The user next requested testing beta.20. See the separate
[beta.8 record](WAILS-BETA8-ROLLBACK-2026-09-20.md) and
[beta.20 trial](WAILS-BETA20-TRIAL-2026-09-20.md). Both comparison artifacts have
distinct filenames, and the existing published 0.2.0 installer is unchanged.

[Microsoft scanner arguments](https://learn.microsoft.com/en-us/defender-endpoint/command-line-arguments-microsoft-defender-antivirus)
