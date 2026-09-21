# Windows 0.2.2 upgrade candidate and signing preparation

Status: built and automatically verified candidate, not a new public stable release.

## Evidence

- Desktop sources: `8828eac` (later Android-only changes do not alter this installer).
- [CI run 35575152992](https://github.com/bimcc/headscaleclient/actions/runs/35575152992):
  Windows, macOS and Linux Go/frontend/native checks passed.
- Windows disposable-runner integration passed: seeded two historical publisher
  records, installed once, confirmed one stable product record and removal of the
  old executable, preserved an unrelated user file, reinstalled in place, checked
  managed service startup, and uninstalled.
- Local PowerShell migration tests passed. Local NSIS compilation also passed.
- The developer's installed applications and Tailscale service were not modified.

The observed local records were `BIMCC., Ltd.HeadscaleClient` (0.2.0) and
`HeadscaleClient ContributorsHeadscaleClient` (0.1.0). The new stable uninstall
identity is `io.headscaleclient.desktop`. Updates keep the existing directory and
account state. Known duplicate program files and historical records are removed
after a successful replacement. The installer does not invoke old uninstallers.

## Artifacts

The distributed candidate is the exact installer downloaded from the successful
Windows CI job:

| File in `bin/windows-preview/` | SHA-256 |
| --- | --- |
| `headscaleclient-0.2.2-windows-amd64-installer.exe` | `9281674e11cee08db538e0d57d646b2834f1208319764f8ff62b4fc7b6e8c9cd` |
| `headscaleclient-0.2.2-windows-amd64-installer-selfsigned-test.exe` | `70ba1234b3928e23056ee72b2375607abc82c8d3a9505437148d12b87c96db00` |

The second file is a copy of the first with a local SHA-256 Authenticode test
signature applied to the outer installer only. Subject:
`CN=BIMCC HeadscaleClient (Self-signed Test)`. Certificate thumbprint:
`0E033811E3B0E9F98C7F1D3BFF3BCC8474C89B57`.

`Get-AuthenticodeSignature` finds this signer but returns `UnknownError` because
the certificate chain ends at an untrusted root. This is expected and is **not**
publicly trusted signing. No root certificate was installed. The unsigned inner
GUI/uninstaller and upstream components were not re-signed by this test.
Public certificate: `bin/signing/HeadscaleClient-test.cer`; non-exportable private
key remains in the creating user's Windows personal certificate store.

## SignPath

[Code signing policy](../../CODE_SIGNING.md) and a gated manual GitHub workflow
are prepared. SignPath has not accepted the project, supplied credentials or
signed an artifact. Maintainer roles/MFA, repository integration, reviewed artifact
policies, privacy disclosures and the Wintun/WebView2 license eligibility question
must be resolved with the provider before enabling it. Self-signing and public
signing do not guarantee absence of antivirus detections.
