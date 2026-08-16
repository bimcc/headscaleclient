param(
    [Parameter(Mandatory = $true)]
    [string]$DaemonDirectory,

    [Parameter(Mandatory = $true)]
    [string]$ApplicationBinary,

    [Parameter(Mandatory = $true)]
    [ValidateSet("AMD64", "ARM64")]
    [string]$ArchitectureFlag,

    [Parameter(Mandatory = $true)]
    [string]$ProjectFile
)

$ErrorActionPreference = "Stop"

$command = Get-Command makensis.exe -ErrorAction SilentlyContinue
$makensis = if ($null -ne $command) { $command.Source } else { $null }
if (-not $makensis) {
    $programFilesX86 = [Environment]::GetEnvironmentVariable("ProgramFiles(x86)")
    if ($programFilesX86) {
        $candidate = Join-Path $programFilesX86 "NSIS\makensis.exe"
        if (Test-Path -LiteralPath $candidate) {
            $makensis = $candidate
        }
    }
}
if (-not $makensis) {
    throw "makensis.exe was not found. Install NSIS 3 and reopen the terminal."
}

& $makensis "-DARG_DAEMON_DIR=$DaemonDirectory" "-DARG_WAILS_$($ArchitectureFlag)_BINARY=$ApplicationBinary" $ProjectFile
if ($LASTEXITCODE -ne 0) {
    throw "makensis.exe failed with exit code $LASTEXITCODE."
}
