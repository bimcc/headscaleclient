param(
    [ValidateSet('Discover', 'Prepare', 'Complete', 'Library')][string]$Mode = 'Discover',
    [string]$InstallDirectory
)
$ErrorActionPreference = 'Stop'
$script:ProductKey = 'io.headscaleclient.desktop'
$script:ProductKeys = @($script:ProductKey, 'BIMCC., Ltd.HeadscaleClient', 'HeadscaleClient ContributorsHeadscaleClient')
$script:UninstallRoot = 'SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall'

function Assert-ProductDirectory([string]$Directory) {
    if (-not [IO.Path]::IsPathRooted($Directory) -or $Directory.StartsWith('\\')) { throw 'Invalid product directory.' }
    $path = [IO.Path]::GetFullPath($Directory).TrimEnd('\')
    $forbidden = @([IO.Path]::GetPathRoot($path).TrimEnd('\'), $env:WINDIR, $env:ProgramFiles, $env:ProgramW6432,
        ${env:ProgramFiles(x86)}, $env:USERPROFILE, $env:APPDATA, $env:LOCALAPPDATA,
        (Join-Path $env:ProgramFiles 'BIMCC'))
    if ($forbidden -contains $path) { throw 'Refusing a shared system/user directory as product directory.' }
    if (Test-Path -LiteralPath $path) {
        $item = Get-Item -LiteralPath $path
        for ($ancestor = $item; $null -ne $ancestor; $ancestor = $ancestor.Parent) {
            if (($ancestor.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) { throw 'Product directory must not traverse a junction or symbolic link.' }
        }
    }
    return $path
}

function Get-ProductInstalls {
    foreach ($view in @([Microsoft.Win32.RegistryView]::Registry64, [Microsoft.Win32.RegistryView]::Registry32)) {
        $registry = [Microsoft.Win32.RegistryKey]::OpenBaseKey([Microsoft.Win32.RegistryHive]::LocalMachine, $view)
        try {
            foreach ($name in $script:ProductKeys) {
                $key = $registry.OpenSubKey("$script:UninstallRoot\$name")
                if ($null -eq $key) { continue }
                try {
                    if ($key.GetValue('DisplayName') -ne 'HeadscaleClient') { throw "Unexpected product identity in $name" }
                    $directory = [string]$key.GetValue('InstallLocation')
                    if (-not $directory) {
                        $uninstall = [string]$key.GetValue('UninstallString')
                        if ($uninstall -match '^"(.+)\\uninstall\.exe"$') { $directory = $Matches[1] }
                        elseif ($uninstall -match '^([A-Za-z]:\\.+)\\uninstall\.exe$') { $directory = $Matches[1] }
                        else { throw "Cannot safely resolve installed product directory in $name" }
                    }
                    $directory = Assert-ProductDirectory $directory
                    $exe = Join-Path $directory 'headscaleclient.exe'
                    $exists = Test-Path -LiteralPath $exe
                    if ($exists) {
                        $productName = (Get-Item -LiteralPath $exe).VersionInfo.ProductName
                        # Early released binaries had no version resources. Only
                        # the two exact historic registry identities may use that fallback.
                        if ($productName -ne 'HeadscaleClient' -and ($productName -or $name -eq $script:ProductKey)) {
                            throw "Installed executable has unexpected product metadata: $exe"
                        }
                    }
                    [PSCustomObject]@{ Key = $name; View = $view; Directory = $directory; Exists = $exists }
                } finally { $key.Dispose() }
            }
        } finally { $registry.Dispose() }
    }
}

function Select-ProductDirectory($Installs) {
    # Prefer stable/current publisher records, then the original publisher.
    foreach ($name in $script:ProductKeys) {
        foreach ($install in $Installs) { if ($install.Key -eq $name -and $install.Exists) { return $install.Directory } }
    }
    return ''
}

function Remove-OldProductFiles([string]$Directory, [string]$CurrentDirectory) {
    $Directory = Assert-ProductDirectory $Directory
    if ($Directory -eq (Assert-ProductDirectory $CurrentDirectory)) { return }
    # Explicit product files only. Never run an old uninstaller (which may
    # recursively remove its directory) or remove account/core state directories.
    $files = @('headscaleclient.exe', 'uninstall.exe', 'daemon\tailscaled.exe', 'daemon\tailscale.exe',
        'daemon\wintun.dll', 'daemon\provenance.json', 'daemon\licenses\TAILSCALE-LICENSE.txt',
        'daemon\licenses\WINTUN-PREBUILT-LICENSE.txt')
    foreach ($file in $files) {
        $path = Join-Path $Directory $file
        if (Test-Path -LiteralPath $path) {
            $null = Assert-ProductDirectory (Split-Path -Parent $path)
            if ((Get-Item -LiteralPath $path).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Refusing a linked product file.' }
            Remove-Item -LiteralPath $path -Force
        }
    }
    foreach ($relative in @('daemon\licenses', 'daemon', '')) {
        $path = if ($relative) { Join-Path $Directory $relative } else { $Directory }
        if ((Test-Path -LiteralPath $path) -and -not (Get-ChildItem -Force -LiteralPath $path | Select-Object -First 1)) {
            Remove-Item -LiteralPath $path
        }
    }
}

if ($Mode -eq 'Library') { return }
try {
    $installs = @(Get-ProductInstalls)
    if ($Mode -eq 'Discover') { Write-Output (Select-ProductDirectory $installs); exit 0 }
    $InstallDirectory = Assert-ProductDirectory $InstallDirectory
    $selected = Select-ProductDirectory $installs
    if ($selected -and $selected -ne $InstallDirectory) { throw 'Updates must reuse the registered product directory.' }
    if ($Mode -eq 'Prepare') {
        $serviceKey = 'HKLM:\SYSTEM\CurrentControlSet\Services\Tailscale'
        $image = (Get-ItemProperty -LiteralPath $serviceKey -ErrorAction SilentlyContinue).ImagePath
        $image = if ($image) { $image.Trim('"') } else { '' }
        foreach ($install in $installs) {
            if ($install.Directory -eq $InstallDirectory -or $image -ne (Join-Path $install.Directory 'daemon\tailscaled.exe')) { continue }
            $service = Get-Service Tailscale
            if ($service.Status -ne 'Stopped') { Stop-Service Tailscale; $service.WaitForStatus('Stopped', [TimeSpan]::FromSeconds(20)) }
            & "$env:WINDIR\System32\sc.exe" delete Tailscale | Out-Null
            if ($LASTEXITCODE -ne 0) { throw 'Could not migrate the legacy managed network service.' }
            $service.Dispose()
            for ($attempt = 0; $attempt -lt 40 -and (Test-Path $serviceKey); $attempt++) { Start-Sleep -Milliseconds 250 }
            if (Test-Path $serviceKey) { throw 'The old managed service is still being removed; close service management tools and retry.' }
            break
        }
    } else {
        foreach ($install in $installs) {
            Remove-OldProductFiles $install.Directory $InstallDirectory
            if ($install.Key -eq $script:ProductKey -and $install.View -eq [Microsoft.Win32.RegistryView]::Registry64) { continue }
            $registry = [Microsoft.Win32.RegistryKey]::OpenBaseKey([Microsoft.Win32.RegistryHive]::LocalMachine, $install.View)
            try { $registry.DeleteSubKeyTree("$script:UninstallRoot\$($install.Key)", $false) } finally { $registry.Dispose() }
        }
    }
} catch { Write-Output $_.Exception.Message; exit 1 }
