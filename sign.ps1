# sign.ps1 - Complete Signing Script

Write-Host "Starting signing process..." -ForegroundColor Green

# 1. Certificate ko Personal store se get karein
$cert = Get-ChildItem Cert:\CurrentUser\My -CodeSigningCert | Where-Object {$_.Subject -like "*CN=CMDO*"} | Select-Object -First 1

if (-not $cert) {
    Write-Host "Error: CMDO certificate not found!" -ForegroundColor Red
    exit 1
}

Write-Host "Certificate found: $($cert.Subject)" -ForegroundColor Yellow

# 2. EXE file ko sign karein
try {
    $result = Set-AuthenticodeSignature -FilePath "D:\Tanu\Projects\cmdo\cmdo.exe" -Certificate $cert -TimestampServer "http://timestamp.digicert.com"
    
    if ($result.Status -eq "Valid") {
        Write-Host "Successfully signed: cmdo.exe" -ForegroundColor Green
    } else {
        Write-Host "Signing status: $($result.Status)" -ForegroundColor Yellow
    }
} catch {
    Write-Host "Error signing file: $_" -ForegroundColor Red
    exit 1
}

# 3. Certificate ko Trusted Root mein install karein (if not already)
Write-Host "`nInstalling certificate to Trusted Root..." -ForegroundColor Green

# Certificate export karein
$cerPath = "D:\Tanu\Projects\cmdo\CMDO.cer"
Export-Certificate -Cert $cert -FilePath $cerPath -Force | Out-Null

# Check if running as Administrator
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

if ($isAdmin) {
    # Trusted Root mein import karein
    Import-Certificate -FilePath $cerPath -CertStoreLocation Cert:\LocalMachine\Root -ErrorAction SilentlyContinue | Out-Null
    Write-Host "Certificate installed to Trusted Root successfully!" -ForegroundColor Green
} else {
    Write-Host "Warning: Not running as Administrator. Cannot install to Trusted Root automatically." -ForegroundColor Yellow
    Write-Host "Please run this command as Administrator:" -ForegroundColor Yellow
    Write-Host "Import-Certificate -FilePath '$cerPath' -CertStoreLocation Cert:\LocalMachine\Root" -ForegroundColor Cyan
}

# 4. Verification
Write-Host "`nVerifying signature..." -ForegroundColor Green
$sig = Get-AuthenticodeSignature -FilePath "D:\Tanu\Projects\cmdo\cmdo.exe"
Write-Host "Signature Status: $($sig.Status)" -ForegroundColor $(if($sig.Status -eq "Valid"){"Green"}else{"Yellow"})

Write-Host "`n=== Process Complete ===" -ForegroundColor Green
Write-Host "Your cmdo.exe is now signed!" -ForegroundColor Cyan