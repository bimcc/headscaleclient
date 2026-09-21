param([string]$PublicCertificate = "$PSScriptRoot/../../bin/signing/HeadscaleClient-test.cer")
$ErrorActionPreference = 'Stop'
$name = 'HeadscaleClient internal test signing'
$certificate = Get-ChildItem Cert:/CurrentUser/My | Where-Object {
    $_.FriendlyName -eq $name -and $_.HasPrivateKey -and $_.NotAfter -gt (Get-Date).AddDays(30)
} | Select-Object -First 1
if (-not $certificate) {
    $certificate = New-SelfSignedCertificate -Type CodeSigningCert -Subject 'CN=BIMCC HeadscaleClient (Self-signed Test)' `
        -FriendlyName $name -CertStoreLocation Cert:/CurrentUser/My -KeyAlgorithm RSA -KeyLength 3072 `
        -HashAlgorithm SHA256 -KeyExportPolicy NonExportable -NotAfter (Get-Date).AddYears(3)
}
New-Item -ItemType Directory -Path (Split-Path $PublicCertificate) -Force | Out-Null
Export-Certificate -Cert $certificate -FilePath $PublicCertificate -Force | Out-Null
Write-Output "Test certificate thumbprint: $($certificate.Thumbprint)"
Write-Output "Public certificate: $([IO.Path]::GetFullPath($PublicCertificate))"
Write-Output 'Private key stays in CurrentUser/My. No trusted-root store was modified. This is not publicly trusted signing.'
