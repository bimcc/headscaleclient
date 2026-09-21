# Integration test exclusively for an empty disposable GitHub Windows runner.
$ErrorActionPreference = 'Stop'
if ($env:GITHUB_ACTIONS -ne 'true') { throw 'Run only on a disposable GitHub Actions runner.' }
. "$PSScriptRoot/windows-upgrade.ps1" -Mode Library
if (@(Get-ProductInstalls).Count -or (Get-Service Tailscale -ErrorAction SilentlyContinue)) { throw 'Runner is not empty; refusing to alter an existing installation/service.' }
$repo = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
$installer = Join-Path $repo 'bin/headscaleclient-amd64-installer.exe'
$current = Join-Path $env:ProgramFiles 'BIMCC/HeadscaleClient'
$legacy = Join-Path $env:ProgramFiles 'HeadscaleClient Contributors/HeadscaleClient'
foreach ($directory in @($current, $legacy)) {
    if (Test-Path -LiteralPath $directory) { throw 'Fixture directory already exists.' }
    New-Item -ItemType Directory -Path $directory -Force | Out-Null
    Copy-Item (Join-Path $repo 'bin/headscaleclient.exe') (Join-Path $directory 'headscaleclient.exe')
    [IO.File]::WriteAllText((Join-Path $directory 'uninstall.exe'), 'Never execute this legacy uninstaller fixture')
}
[IO.File]::WriteAllText((Join-Path $legacy 'user-notes.txt'), 'must survive upgrade')
$registry = [Microsoft.Win32.RegistryKey]::OpenBaseKey([Microsoft.Win32.RegistryHive]::LocalMachine, [Microsoft.Win32.RegistryView]::Registry64)
try {
    foreach ($entry in @(@('BIMCC., Ltd.HeadscaleClient',$current,'0.2.1'), @('HeadscaleClient ContributorsHeadscaleClient',$legacy,'0.1.0'))) {
        $key = $registry.CreateSubKey("$script:UninstallRoot\$($entry[0])")
        try {
            $key.SetValue('DisplayName','HeadscaleClient'); $key.SetValue('DisplayVersion',$entry[2])
            $key.SetValue('UninstallString', '"' + $entry[1] + '\uninstall.exe"')
        } finally { $key.Dispose() }
    }
} finally { $registry.Dispose() }
for ($iteration = 0; $iteration -lt 2; $iteration++) {
    $process = Start-Process -FilePath $installer -ArgumentList '/S' -PassThru -WindowStyle Hidden
    if (-not $process.WaitForExit(120000)) { $process.Kill(); throw 'Installer timed out.' }
    if ($process.ExitCode -ne 0) { throw "Installer exit code: $($process.ExitCode)" }
    $installs = @(Get-ProductInstalls)
    if ($installs.Count -ne 1 -or $installs[0].Key -ne $script:ProductKey -or $installs[0].Directory -ne $current) { throw 'Upgrade left multiple registrations or moved the install.' }
    if (Test-Path (Join-Path $legacy 'headscaleclient.exe')) { throw 'Legacy executable remains runnable.' }
    if (-not (Test-Path (Join-Path $legacy 'user-notes.txt'))) { throw 'Unrelated files were deleted.' }
    if ((Get-Service Tailscale).Status -ne 'Running') { throw 'Managed service is not running.' }
    $expected = (Get-Content (Join-Path $repo 'VERSION') -Raw).Trim()
    if ((Get-Item (Join-Path $current 'headscaleclient.exe')).VersionInfo.ProductVersion -ne $expected) { throw 'Installed product version is incorrect.' }
}
$uninstall = Start-Process -FilePath (Join-Path $current 'uninstall.exe') -ArgumentList '/S',"_?=$current" -PassThru -WindowStyle Hidden
if (-not $uninstall.WaitForExit(120000)) { $uninstall.Kill(); throw 'Uninstaller timed out.' }
if ($uninstall.ExitCode -ne 0 -or @(Get-ProductInstalls).Count) { throw 'Uninstall left a registered product.' }
Write-Output 'Passed real silent install over two legacy records, repair-in-place, service startup, file preservation and uninstall.'
