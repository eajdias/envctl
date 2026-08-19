<#
.SYNOPSIS
    Bootstrap installer and runner for win11-new.
.DESCRIPTION
    Downloads the latest release of win11-new, extracts it, and executes the provisioner.
    Can be run via:
        irm https://raw.githubusercontent.com/eajdias/win11-new/main/bootstrap.ps1 | iex
    Or with parameters:
        & ([scriptblock]::Create((irm https://raw.githubusercontent.com/eajdias/win11-new/main/bootstrap.ps1))) -Subsystem lsp
#>

[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [string]$Command = "run",

    [Parameter(Position = 1)]
    [string]$Subsystem = "all",

    [string]$Version = "latest",
    [switch]$DryRun,
    [switch]$Force
)

$ErrorActionPreference = "Stop"

Write-Host "================================================================" -ForegroundColor Cyan
Write-Host "  🚀 Windows 11 PRO / MSYS2 / OpenCode Provisioner Bootstrap" -ForegroundColor Cyan
Write-Host "================================================================" -ForegroundColor Cyan

# 1. Architecture Check
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
if ($arch -ne "amd64") {
    Write-Error "Unsupported architecture: $arch. win11-new requires Windows 64-bit."
    exit 1
}

# 2. Check if local compiled win11-new exists in current dir
$LocalExe = Join-Path (Get-Location) "win11-new.exe"
$TargetExe = $null

if (Test-Path $LocalExe -and -not $Force) {
    Write-Host "[*] Found local win11-new binary at $LocalExe" -ForegroundColor Green
    $TargetExe = $LocalExe
} else {
    # 3. Destination folder
    $InstallDir = Join-Path $env:LOCALAPPDATA "win11-new"
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }
    $TargetExe = Join-Path $InstallDir "win11-new.exe"

    # Download from GitHub Releases
    $Repo = "eajdias/win11-new"
    $DownloadUrl = if ($Version -eq "latest") {
        "https://github.com/$Repo/releases/latest/download/win11-new-windows-amd64.zip"
    } else {
        "https://github.com/$Repo/releases/download/$Version/win11-new-windows-amd64.zip"
    }

    $ZipPath = Join-Path $env:TEMP "win11-new.zip"

    Write-Host "[*] Downloading win11-new ($Version) from GitHub..." -ForegroundColor Yellow
    try {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12 -bor [Net.SecurityProtocolType]::Tls13
        Invoke-WebRequest -Uri $DownloadUrl -OutFile $ZipPath -UseBasicParsing
        Write-Host "[*] Extracting package..." -ForegroundColor Yellow
        Expand-Archive -Path $ZipPath -DestinationPath $InstallDir -Force
        Remove-Item $ZipPath -Force -ErrorAction SilentlyContinue
        Write-Host "[+] Download complete: $TargetExe" -ForegroundColor Green
    } catch {
        Write-Warning "Could not download pre-built binary: $_"
        # Fallback: check if Go is installed locally to compile
        $GoCmd = Get-Command "go" -ErrorAction SilentlyContinue
        if ($GoCmd) {
            Write-Host "[*] Go toolchain detected. Attempting to build from source..." -ForegroundColor Yellow
            $SourceDir = Join-Path $env:TEMP "win11-new-source"
            if (Test-Path $SourceDir) { Remove-Item -Recurse -Force $SourceDir }
            git clone --depth 1 "https://github.com/$Repo.git" $SourceDir
            Push-Location $SourceDir
            go build -ldflags "-s -w" -o $TargetExe ./cmd/win11-new
            Pop-Location
            Remove-Item -Recurse -Force $SourceDir -ErrorAction SilentlyContinue
            Write-Host "[+] Build from source complete!" -ForegroundColor Green
        } else {
            Write-Error "Failed to acquire win11-new binary and Go is not installed. Please install Go or check GitHub release."
            exit 1
        }
    }
}

# 4. Execute win11-new with arguments
$ArgsList = @()
if ($Command) { $ArgsList += $Command }
if ($Subsystem -and $Command -eq "run") { $ArgsList += $Subsystem }
if ($DryRun) { $ArgsList += "--dry-run" }

Write-Host "[*] Launching: $TargetExe $($ArgsList -join ' ')" -ForegroundColor Cyan
& $TargetExe @ArgsList
