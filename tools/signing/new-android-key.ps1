param([string]$Directory = "$env:LOCALAPPDATA\BIMCC\Signing\Android")
$ErrorActionPreference = 'Stop'
$openssl = (Get-Command openssl.exe -ErrorAction Stop).Source
if ((Test-Path -LiteralPath $Directory) -and (Get-ChildItem -Force -LiteralPath $Directory | Select-Object -First 1)) { throw 'Signing directory is not empty; never replace an established update key.' }
New-Item -ItemType Directory -Path $Directory -Force | Out-Null
$sid = [System.Security.Principal.WindowsIdentity]::GetCurrent().User.Value
& icacls.exe $Directory /inheritance:r /grant:r "*${sid}:(OI)(CI)F" '*S-1-5-18:(OI)(CI)F' | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'Could not restrict signing directory permissions.' }
$bytes = New-Object byte[] 32
$rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
$rng.GetBytes($bytes); $rng.Dispose()
$env:HEADSCALE_KEY_PASSWORD = [Convert]::ToBase64String($bytes)
try {
    $config = Join-Path (Split-Path $openssl) '../ssl/openssl.cnf'
    & $openssl req -config $config -x509 -newkey rsa:3072 -sha256 -days 10000 -subj '/CN=BIMCC HeadscaleClient Android/O=BIMCC' -passout env:HEADSCALE_KEY_PASSWORD -keyout "$Directory\key.pem" -out "$Directory\certificate.pem" 2>$null
    if ($LASTEXITCODE -ne 0) { throw 'Android certificate generation failed.' }
    & $openssl pkcs12 -export -name headscaleclient -keypbe PBE-SHA1-3DES -certpbe PBE-SHA1-3DES -macalg SHA256 -inkey "$Directory\key.pem" -in "$Directory\certificate.pem" -passin env:HEADSCALE_KEY_PASSWORD -passout env:HEADSCALE_KEY_PASSWORD -out "$Directory\headscaleclient.p12"
    if ($LASTEXITCODE -ne 0) { throw 'Android keystore export failed.' }
    $protected = ConvertTo-SecureString $env:HEADSCALE_KEY_PASSWORD -AsPlainText -Force | ConvertFrom-SecureString
    [IO.File]::WriteAllText("$Directory\password.dpapi", $protected)
    & $openssl x509 -in "$Directory\certificate.pem" -noout -fingerprint -sha256
    Write-Output "Private key retained outside the repository: $Directory"
} finally { Remove-Item Env:HEADSCALE_KEY_PASSWORD }
