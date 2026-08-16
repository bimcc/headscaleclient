param(
    [Parameter(Mandatory = $true)]
    [ValidateSet("amd64", "arm64")]
    [string]$Architecture,

    [Parameter(Mandatory = $true)]
    [string]$OutputDirectory
)

$ErrorActionPreference = "Stop"

$repositoryRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot "..\.."))
$manifestPath = Join-Path $repositoryRoot "build\daemon\manifest.json"
$manifest = Get-Content -LiteralPath $manifestPath -Raw -Encoding UTF8 | ConvertFrom-Json
$entry = $manifest.windows.$Architecture
if ($null -eq $entry) {
    throw "No Windows daemon manifest entry exists for $Architecture."
}

$cacheDirectory = Join-Path $repositoryRoot ".task\daemon\windows-$Architecture"
$extractDirectory = Join-Path $cacheDirectory "extracted"
$msiPath = Join-Path $cacheDirectory ([IO.Path]::GetFileName([Uri]$entry.url))
$outputPath = [IO.Path]::GetFullPath((Join-Path $repositoryRoot $OutputDirectory))

New-Item -ItemType Directory -Path $cacheDirectory -Force | Out-Null
New-Item -ItemType Directory -Path $extractDirectory -Force | Out-Null
New-Item -ItemType Directory -Path $outputPath -Force | Out-Null

function Assert-Hash([string]$Path, [string]$Expected) {
    $actual = (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -ne $Expected.ToLowerInvariant()) {
        throw "SHA-256 mismatch for $Path. Expected $Expected, got $actual."
    }
}

function Assert-Signature([string]$Path, [string]$ExpectedSubject) {
    $signature = Get-AuthenticodeSignature -LiteralPath $Path
    if ($signature.Status -ne [Management.Automation.SignatureStatus]::Valid) {
        throw "Authenticode verification failed for ${Path}: $($signature.StatusMessage)"
    }
    if (-not $signature.SignerCertificate.Subject.StartsWith($ExpectedSubject, [StringComparison]::Ordinal)) {
        throw "Unexpected signer for ${Path}: $($signature.SignerCertificate.Subject)"
    }
}

$downloadRequired = -not (Test-Path -LiteralPath $msiPath)
if (-not $downloadRequired) {
    try {
        Assert-Hash $msiPath $entry.sha256
    } catch {
        $downloadRequired = $true
    }
}
if ($downloadRequired) {
    Invoke-WebRequest -Uri $entry.url -OutFile $msiPath -UseBasicParsing
}

Assert-Hash $msiPath $entry.sha256
Assert-Signature $msiPath "CN=Tailscale Inc."

$arguments = @("/a", $msiPath, "/qn", "TARGETDIR=$extractDirectory")
$installer = Start-Process -FilePath "msiexec.exe" -ArgumentList $arguments -WindowStyle Hidden -Wait -PassThru
if ($installer.ExitCode -ne 0) {
    throw "MSI administrative extraction failed with exit code $($installer.ExitCode)."
}

$provenanceFiles = @{}
foreach ($property in $entry.files.PSObject.Properties) {
    $name = $property.Name
    $matches = @(Get-ChildItem -LiteralPath $extractDirectory -Recurse -File -Filter $name)
    if ($matches.Count -ne 1) {
        throw "Expected exactly one $name in the MSI, found $($matches.Count)."
    }
    $source = $matches[0].FullName
    Assert-Hash $source ([string]$property.Value)
    $expectedSigner = if ($name -eq "wintun.dll") { "CN=WireGuard LLC" } else { "CN=Tailscale Inc." }
    Assert-Signature $source $expectedSigner
    Copy-Item -LiteralPath $source -Destination (Join-Path $outputPath $name) -Force
    $provenanceFiles[$name] = ([string]$property.Value).ToLowerInvariant()
}

$licenseOutput = Join-Path $outputPath "licenses"
New-Item -ItemType Directory -Path $licenseOutput -Force | Out-Null
Copy-Item -LiteralPath (Join-Path $repositoryRoot "build\daemon\licenses\TAILSCALE-LICENSE.txt") -Destination $licenseOutput -Force
Copy-Item -LiteralPath (Join-Path $repositoryRoot "build\daemon\licenses\WINTUN-PREBUILT-LICENSE.txt") -Destination $licenseOutput -Force

$provenance = [ordered]@{
    schemaVersion = 1
    upstream = "tailscale/tailscale"
    version = [string]$manifest.version
    platform = "windows"
    architecture = $Architecture
    source = [string]$entry.url
    sourceSha256 = ([string]$entry.sha256).ToLowerInvariant()
    files = $provenanceFiles
}
$provenance | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $outputPath "provenance.json") -Encoding UTF8

Write-Host "Prepared verified daemon payload at $outputPath"
