<#
.SYNOPSIS
    Bootstrap installer and runner for envctl (Windows 11 PRO).
.DESCRIPTION
    Downloads the latest release of envctl, extracts it, and executes the provisioner.
    Can be run via:
        irm https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.ps1 | iex
    Or with parameters:
        & ([scriptblock]::Create((irm https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.ps1))) -Subsystem lsp
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
Write-Host "  🚀 envctl: Development Environment Provisioner Bootstrap" -ForegroundColor Cyan
Write-Host "================================================================" -ForegroundColor Cyan

# 1. Architecture Check
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
if ($arch -ne "amd64") {
    Write-Error "Unsupported architecture: $arch. envctl requires 64-bit Windows."
    exit 1
}

# 2. Check if local compiled envctl exists in current dir
$LocalExe = Join-Path (Get-Location) "envctl.exe"
$TargetExe = $null

if (Test-Path $LocalExe -and -not $Force) {
    Write-Host "[*] Found local envctl binary at $LocalExe" -ForegroundColor Green
    $TargetExe = $LocalExe
} else {
    # 3. Destination folder
    $InstallDir = Join-Path $env:LOCALAPPDATA "envctl"
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }
    $TargetExe = Join-Path $InstallDir "envctl.exe"

    # Download from GitHub Releases
    $Repo = "eajdias/envctl"
    $DownloadUrl = if ($Version -eq "latest") {
        "https://github.com/$Repo/releases/latest/download/envctl-windows-amd64.zip"
    } else {
        "https://github.com/$Repo/releases/download/$Version/envctl-windows-amd64.zip"
    }

    $ZipPath = Join-Path $env:TEMP "envctl.zip"

    Write-Host "[*] Downloading envctl ($Version) from GitHub..." -ForegroundColor Yellow
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
            $SourceDir = Join-Path $env:TEMP "envctl-source"
            if (Test-Path $SourceDir) { Remove-Item -Recurse -Force $SourceDir }
            git clone --depth 1 "https://github.com/$Repo.git" $SourceDir
            Push-Location $SourceDir
            go build -ldflags "-s -w" -o $TargetExe ./cmd/envctl
            Pop-Location
            Remove-Item -Recurse -Force $SourceDir -ErrorAction SilentlyContinue
            Write-Host "[+] Build from source complete!" -ForegroundColor Green
        } else {
            Write-Error "Failed to acquire envctl binary and Go is not installed. Please install Go or check GitHub release."
            exit 1
        }
    }
}

# 4. Execute envctl with arguments
$ArgsList = @()
if ($Command) { $ArgsList += $Command }
if ($Subsystem -and $Command -eq "run") { $ArgsList += $Subsystem }
if ($DryRun) { $ArgsList += "--dry-run" }

Write-Host "[*] Launching: $TargetExe $($ArgsList -join ' ')" -ForegroundColor Cyan
& $TargetExe @ArgsList
