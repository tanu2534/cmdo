# CMDO Installer for Windows
# Usage: iwr -useb https://raw.githubusercontent.com/tanu2534/cmdo-release/main/install.ps1 | iex

param(
    [switch]$NoAddToPath,
    [switch]$DryRun
)

$ErrorActionPreference = "Stop"

$BASE_URL = "https://raw.githubusercontent.com/tanu2534/cmdo-release/main"
$INSTALL_NAME = "cmdo"

Write-Host ""
Write-Host "  CMDO Installer for Windows" -ForegroundColor Cyan
Write-Host ""

# Check PowerShell version
if ($PSVersionTable.PSVersion.Major -lt 5) {
    Write-Host "[!] Error: PowerShell 5+ required" -ForegroundColor Red
    exit 1
}

Write-Host "[OK] Windows detected" -ForegroundColor Green

# Detect architecture
function Get-Architecture {
    if ([Environment]::Is64BitOperatingSystem) {
        $arch = [Environment]::GetEnvironmentVariable("PROCESSOR_ARCHITECTURE")
        if ($arch -eq "AMD64") {
            return "amd64"
        } elseif ($arch -eq "ARM64") {
            return "arm64"
        } else {
            return "amd64" # Default to amd64
        }
    } else {
        return "386"
    }
}

# Check for existing installation
function Check-ExistingInstallation {
    try {
        $null = Get-Command cmdo -ErrorAction Stop
        $version = (& cmdo --version 2>$null)
        if ($version) {
            Write-Host "[*] Existing CMDO installation detected: $version" -ForegroundColor Yellow
            return $true
        }
    } catch {
        return $false
    }
    return $false
}

# Download binary
function Download-Binary {
    param(
        [string]$Arch
    )
    
    $binaryName = "cmdo-windows-$Arch.exe"
    $downloadUrl = "$BASE_URL/$binaryName"
    $tempFile = Join-Path $env:TEMP "cmdo-$Arch-$PID.exe"
    
    Write-Host "[*] Detected architecture: $Arch" -ForegroundColor Yellow
    Write-Host "[*] Downloading $binaryName..." -ForegroundColor Yellow
    
    try {
        $ProgressPreference = 'SilentlyContinue'
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tempFile -UseBasicParsing
        $ProgressPreference = 'Continue'
        
        if (-not (Test-Path $tempFile)) {
            throw "Download failed: file not created"
        }
        
        $fileInfo = Get-Item $tempFile
        if ($fileInfo.Length -lt 1KB) {
            throw "Downloaded file is too small (likely an error page)"
        }
        
        Write-Host "[OK] Downloaded successfully ($([math]::Round($fileInfo.Length/1MB, 2)) MB)" -ForegroundColor Green
        return $tempFile
    } catch {
        Write-Host "[!] Error downloading binary: $_" -ForegroundColor Red
        if (Test-Path $tempFile) {
            Remove-Item $tempFile -Force
        }
        Write-Host ""
        Write-Host "Please check:" -ForegroundColor Yellow
        Write-Host "  1. Internet connection" -ForegroundColor Gray
        Write-Host "  2. GitHub URL is accessible: $downloadUrl" -ForegroundColor Gray
        Write-Host "  3. Binary exists in repository" -ForegroundColor Gray
        exit 1
    }
}

# Install binary
function Install-Binary {
    param(
        [string]$TempFile
    )
    
    # Determine installation directory
    $installDir = Join-Path $env:USERPROFILE ".local\bin"
    
    # Create directory if it doesn't exist
    if (-not (Test-Path $installDir)) {
        Write-Host "[*] Creating installation directory: $installDir" -ForegroundColor Yellow
        New-Item -ItemType Directory -Force -Path $installDir | Out-Null
    }
    
    $installPath = Join-Path $installDir "$INSTALL_NAME.exe"
    
    # Backup existing installation if present
    if (Test-Path $installPath) {
        $backupPath = "$installPath.bak-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
        Write-Host "[*] Backing up existing installation to: $backupPath" -ForegroundColor Yellow
        Move-Item -Path $installPath -Destination $backupPath -Force
    }
    
    # Install
    Write-Host "[*] Installing to: $installPath" -ForegroundColor Yellow
    Copy-Item -Path $TempFile -Destination $installPath -Force
    
    # Cleanup
    Remove-Item $TempFile -Force
    
    Write-Host "[OK] Binary installed successfully" -ForegroundColor Green
    return @{
        Path = $installPath
        Dir = $installDir
    }
}

# Add to PATH
function Add-ToPath {
    param(
        [string]$Directory
    )
    
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $pathArray = $userPath -split ";" | Where-Object { $_ }
    
    # Check if already in PATH
    $alreadyInPath = $false
    foreach ($p in $pathArray) {
        if ($p -ieq $Directory) {
            $alreadyInPath = $true
            break
        }
    }
    
    if (-not $alreadyInPath) {
        Write-Host "[*] Adding to user PATH: $Directory" -ForegroundColor Yellow
        $newPath = "$userPath;$Directory"
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        
        # Update current session PATH
        $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
        
        Write-Host "[OK] Added to PATH (restart terminal if command not found)" -ForegroundColor Green
        return $true
    } else {
        Write-Host "[OK] Already in PATH" -ForegroundColor Green
        return $false
    }
}

# Verify installation
function Verify-Installation {
    param(
        [string]$InstallPath
    )
    
    Write-Host "[*] Verifying installation..." -ForegroundColor Yellow
    
    if (-not (Test-Path $InstallPath)) {
        Write-Host "[!] Error: Installation file not found" -ForegroundColor Red
        return $false
    }
    
    # Try to get version
    try {
        $version = (& $InstallPath --version 2>&1)
        if ($version) {
            Write-Host "[OK] Installation verified: $version" -ForegroundColor Green
            return $true
        }
    } catch {
        Write-Host "[!] Warning: Could not verify version, but binary exists" -ForegroundColor Yellow
        return $true
    }
    
    return $true
}

# Main installation flow
function Main {
    if ($DryRun) {
        Write-Host "[OK] Dry run mode" -ForegroundColor Green
        $arch = Get-Architecture
        Write-Host "[OK] Would install for: $arch" -ForegroundColor Green
        Write-Host "[OK] Download URL: $BASE_URL/cmdo-windows-$arch.exe" -ForegroundColor Green
        return
    }
    
    # Check for existing installation
    $isUpgrade = Check-ExistingInstallation
    
    # Detect architecture
    $arch = Get-Architecture
    
    # Download binary
    $tempFile = Download-Binary -Arch $arch
    
    # Install binary
    $installation = Install-Binary -TempFile $tempFile
    
    # Add to PATH
    if (-not $NoAddToPath) {
        $pathAdded = Add-ToPath -Directory $installation.Dir
    }
    
    # Verify installation
    $verified = Verify-Installation -InstallPath $installation.Path
    
    Write-Host ""
    Write-Host "===============================================" -ForegroundColor Cyan
    
    if ($isUpgrade) {
        Write-Host "  ✅ CMDO upgraded successfully!" -ForegroundColor Green
        $upgradeMessages = @(
            "Leveled up! Ready for action.",
            "Fresh code deployed. Let's go.",
            "Update complete. Now with more features.",
            "Upgraded and ready to roll.",
            "New version installed. Time to build!"
        )
        Write-Host "  $(Get-Random -InputObject $upgradeMessages)" -ForegroundColor Gray
    } else {
        Write-Host "  ✅ CMDO installed successfully!" -ForegroundColor Green
        $installMessages = @(
            "Welcome aboard! Time to automate.",
            "Installation complete. Let's build something.",
            "Ready to go! Your productivity just leveled up.",
            "All set! Point me at your tasks.",
            "Installed and ready. What are we building today?"
        )
        Write-Host "  $(Get-Random -InputObject $installMessages)" -ForegroundColor Gray
    }
    
    Write-Host "===============================================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Installation location: $($installation.Path)" -ForegroundColor Gray
    Write-Host ""
    
    # Check if command is available
    $cmdAvailable = $false
    try {
        $null = Get-Command cmdo -ErrorAction Stop
        $cmdAvailable = $true
    } catch {
        $cmdAvailable = $false
    }
    
    if ($cmdAvailable) {
        Write-Host "Next steps:" -ForegroundColor Cyan
        Write-Host "  1. Run: " -NoNewline
        Write-Host "cmdo setup" -ForegroundColor Yellow
        Write-Host "  2. Get help: " -NoNewline
        Write-Host "cmdo --help" -ForegroundColor Yellow
    } else {
        Write-Host "⚠️  Please restart your terminal, then run:" -ForegroundColor Yellow
        Write-Host "  cmdo setup" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "Or add to current session:" -ForegroundColor Gray
        Write-Host "  `$env:Path += `";$($installation.Dir)`"" -ForegroundColor Gray
    }
    
    Write-Host ""
}

# Run main
try {
    Main
} catch {
    Write-Host ""
    Write-Host "[!] Installation failed: $_" -ForegroundColor Red
    Write-Host ""
    Write-Host "For help, visit: https://github.com/tanu2534/cmdo-release" -ForegroundColor Gray
    exit 1
}
