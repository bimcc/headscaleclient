$ErrorActionPreference = 'Stop'
. "$PSScriptRoot/windows-upgrade.ps1" -Mode Library
$testRoot = Join-Path ([IO.Path]::GetTempPath()) ('headscale-upgrade-' + [guid]::NewGuid())
New-Item -ItemType Directory -Path $testRoot | Out-Null
try {
    $old = Join-Path $testRoot 'old'
    $current = Join-Path $testRoot 'current'
    New-Item -ItemType Directory -Path "$old/daemon/licenses", $current | Out-Null
    $installs = @(
        [PSCustomObject]@{ Key = 'HeadscaleClient ContributorsHeadscaleClient'; Directory = $old; Exists = $true },
        [PSCustomObject]@{ Key = 'BIMCC., Ltd.HeadscaleClient'; Directory = $current; Exists = $true }
    )
    if ((Select-ProductDirectory $installs) -ne $current) { throw 'Must prefer the newer publisher registration regardless of enumeration order.' }
    $installs[1].Exists = $false
    if ((Select-ProductDirectory $installs) -ne $old) { throw 'A stale registry record must not create a second copy.' }
    foreach ($relative in @('headscaleclient.exe', 'uninstall.exe', 'daemon/tailscaled.exe', 'daemon/licenses/TAILSCALE-LICENSE.txt')) {
        [IO.File]::WriteAllText((Join-Path $old $relative), 'fixture')
    }
    [IO.File]::WriteAllText("$old/user-notes.txt", 'preserve')
    [IO.File]::WriteAllText("$old/daemon/state.backup", 'preserve')
    [IO.File]::WriteAllText("$current/headscaleclient.exe", 'new')
    Remove-OldProductFiles $old $current
    if (Test-Path "$old/headscaleclient.exe") { throw 'Legacy executable remains.' }
    if (-not (Test-Path "$old/user-notes.txt") -or -not (Test-Path "$old/daemon/state.backup")) { throw 'Unrelated data was removed.' }
    Remove-OldProductFiles $current $current
    if (-not (Test-Path "$current/headscaleclient.exe")) { throw 'Current executable was removed.' }
    foreach ($broad in @([IO.Path]::GetPathRoot($testRoot), $env:ProgramFiles, $env:USERPROFILE)) {
        $rejected = $false
        try { $null = Assert-ProductDirectory $broad } catch { $rejected = $true }
        if (-not $rejected) { throw "Broad directory was accepted: $broad" }
    }
    Write-Output 'Windows upgrade tests passed: identity selection, stale records, exact-file cleanup, retained data, and unsafe paths.'
} finally {
    # This uniquely generated fixture is the only recursive removal in the test.
    if ($testRoot -notlike (Join-Path ([IO.Path]::GetTempPath()) 'headscale-upgrade-*')) { throw 'Invalid fixture cleanup path.' }
    Remove-Item -LiteralPath $testRoot -Recurse -Force
}
