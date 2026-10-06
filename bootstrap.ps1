<#
.SYNOPSIS
    Bootstrap installer and runner for envctl (Windows 11 PRO).
.DESCRIPTION
    Downloads the latest release of envctl, extracts it, and executes the provisioner.
    Can be run via:
        irm https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.ps1 | iex
    Or with parameters:
        & ([scriptblock]::Create((irm https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.ps1))) -Subsystem lsp

    CONTRACT (mirrored in bootstrap.sh - keep both in sync):
      REPO=eajdias/envctl | VERSION from -Version (default latest) | asset name
      envctl-<os>-<arch>.tar.gz (linux) / envctl-windows-<arch>.zip (windows)
      from .goreleaser.yml | download ladder: local binary -> gh release download ->
      direct HTTPS (+GITHUB_TOKEN) -> go build from source | then persist install
      dir on PATH and exec.
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

# Fix encoding: ensure UTF-8 console for Unicode glyphs (emojis, checkmarks)
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
chcp 65001 | Out-Null

# Write-Status: status line without Write-Host (PSScriptAnalyzer
# PSAvoidUsingWriteHost — Write-Host bypasses stdout, so it can neither be
# captured nor redirected). Colors via inline ANSI escapes keep the UX while
# the message stays on the output stream.
$Script:AnsiColors = @{
    Cyan   = "$([char]27)[36m"
    Green  = "$([char]27)[32m"
    Yellow = "$([char]27)[33m"
    Reset  = "$([char]27)[0m"
}

function Write-Status {
    param([string]$Message = "", [string]$Color)
    if ($Color -and $Script:AnsiColors.ContainsKey($Color)) {
        Write-Output "$($Script:AnsiColors[$Color])$Message$($Script:AnsiColors.Reset)"
    } else {
        Write-Output $Message
    }
}

Write-Status "================================================================" -Color Cyan
Write-Status "  🚀 envctl: Development Environment Provisioner Bootstrap" -Color Cyan
Write-Status "================================================================" -Color Cyan

# 1. Architecture detection: 64-bit Windows, amd64 or arm64. The release ships
#    both envctl-windows-{amd64,arm64}.{exe,zip} (see .goreleaser.yml builds).
$arch = if (-not [Environment]::Is64BitOperatingSystem) {
    Write-Error "Unsupported architecture: 32-bit. envctl requires 64-bit Windows."
    exit 1
} elseif ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
    "arm64"
} else {
    "amd64"
}

# 2. Check if local compiled envctl exists in current dir
$LocalExe = Join-Path (Get-Location) "envctl.exe"
$TargetExe = $null

if (Test-Path $LocalExe -and -not $Force) {
    Write-Status "[*] Found local envctl binary at $LocalExe" -Color Green
    $TargetExe = $LocalExe
} else {
    # 3. Destination folder
    $InstallDir = Join-Path $env:LOCALAPPDATA "envctl"
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }
    $TargetExe = Join-Path $InstallDir "envctl.exe"

    # Persist the install dir on the user PATH so `envctl` resolves in any new
    # shell. Idempotent: only prepend when the dir is not already present, the
    # same guard the OpenCode installer uses for ~/.local/bin.
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if ($userPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable('Path', "$userPath;$InstallDir", 'User') | Out-Null
        Write-Status "[+] Added $InstallDir to the user PATH" -Color Green
    }

    # Download from GitHub Releases
    $Repo = "eajdias/envctl"
    $ZipPath = Join-Path $env:TEMP "envctl.zip"
    $downloaded = $false

    # 1. Try via GitHub CLI if available (handles private repository authentication)
    $ghCmd = Get-Command "gh" -ErrorAction SilentlyContinue
    if ($ghCmd) {
        Write-Status "[*] Downloading envctl via GitHub CLI..." -Color Yellow
        try {
            $tagArg = if ($Version -eq "latest") { @() } else { @($Version) }
            gh release download @tagArg --repo $Repo --pattern "envctl-windows-$arch.zip" --dir $env:TEMP --clobber
            $downloadedZip = Join-Path $env:TEMP "envctl-windows-$arch.zip"
            if (Test-Path $downloadedZip) {
                Expand-Archive -Path $downloadedZip -DestinationPath $InstallDir -Force
                Remove-Item $downloadedZip -Force -ErrorAction SilentlyContinue
                Write-Status "[+] Download complete via GitHub CLI: $TargetExe" -Color Green
                $downloaded = $true
            }
        } catch {
            Write-Warning "GitHub CLI download attempt failed: $_"
        }
    }

    # 2. Try direct WebRequest
    if (-not $downloaded) {
        $DownloadUrl = if ($Version -eq "latest") {
            "https://github.com/$Repo/releases/latest/download/envctl-windows-$arch.zip"
        } else {
            "https://github.com/$Repo/releases/download/$Version/envctl-windows-$arch.zip"
        }

        Write-Status "[*] Downloading envctl ($Version) from GitHub releases..." -Color Yellow
        try {
            [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12 -bor [Net.SecurityProtocolType]::Tls13
            $headers = @{}
            if ($env:GITHUB_TOKEN) {
                $headers["Authorization"] = "token $env:GITHUB_TOKEN"
            }
            Invoke-WebRequest -Uri $DownloadUrl -OutFile $ZipPath -Headers $headers -UseBasicParsing
            Write-Status "[*] Extracting package..." -Color Yellow
            Expand-Archive -Path $ZipPath -DestinationPath $InstallDir -Force
            Remove-Item $ZipPath -Force -ErrorAction SilentlyContinue
            Write-Status "[+] Download complete: $TargetExe" -Color Green
            $downloaded = $true
        } catch {
            Write-Warning "Direct web download failed: $_"
        }
    }

    # 3. Fallback: check if Go is installed locally to compile
    if (-not $downloaded) {
        $GoCmd = Get-Command "go" -ErrorAction SilentlyContinue
        if ($GoCmd) {
            Write-Status "[*] Go toolchain detected. Attempting to build from source..." -Color Yellow
            $SourceDir = Join-Path $env:TEMP "envctl-source"
            if (Test-Path $SourceDir) { Remove-Item -Recurse -Force $SourceDir }
            git clone --depth 1 "https://github.com/$Repo.git" $SourceDir
            Push-Location $SourceDir
            go build -ldflags "-s -w" -o $TargetExe ./cmd/envctl
            Pop-Location
            Remove-Item -Recurse -Force $SourceDir -ErrorAction SilentlyContinue
            Write-Status "[+] Build from source complete!" -Color Green
            $downloaded = $true
        } else {
            Write-Error "Failed to acquire envctl binary and Go is not installed. Please authenticate gh CLI or install Go."
            exit 1
        }
    }
}

# 4. Execute envctl with arguments
$ArgsList = @()
if ($Command) { $ArgsList += $Command }
if ($Subsystem -and $Command -eq "run") { $ArgsList += $Subsystem }
if ($DryRun) { $ArgsList += "--dry-run" }

Write-Status "[*] Launching: $TargetExe $($ArgsList -join ' ')" -Color Cyan
& $TargetExe @ArgsList

if ($LASTEXITCODE -eq 0 -and $Command -eq "run") {
    Write-Status ""
    Write-Status "⚠️  Provisionamento concluído. Reinicie o OpenCode para aplicar o novo shell PowerShell." -Color Yellow
    Write-Status "   Arquivos temporários dos agentes LLM agora usam C:\temp (ENVCTL_TEMP)." -Color Yellow
    Write-Status ""
}
