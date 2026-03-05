#Requires -Version 5.1
<#
.SYNOPSIS
    Build the Rooam POS Agent Windows installer (MSI).

.DESCRIPTION
    1. Cross-compiles the Go agent binary for Windows/amd64.
    2. Runs WiX Toolset v4 to produce RooamPOSAgent-Setup.msi.

.NOTES
    Run this script from the installer\ directory:
        cd installer
        .\build.ps1
#>

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# ---------------------------------------------------------------------------
# Resolve paths
# ---------------------------------------------------------------------------
$InstallerDir = $PSScriptRoot
$RepoRoot     = Split-Path -Parent $InstallerDir
$OutDir       = Join-Path $InstallerDir 'out'
$AgentExe     = Join-Path $OutDir 'POSitouch-Integration.exe'
$MsiOut       = Join-Path $OutDir 'RooamPOSAgent-Setup.msi'
$WxsFile      = Join-Path $InstallerDir 'installer.wxs'

# ---------------------------------------------------------------------------
# Ensure output directory exists
# ---------------------------------------------------------------------------
if (-not (Test-Path $OutDir)) {
    New-Item -ItemType Directory -Path $OutDir | Out-Null
}

# ---------------------------------------------------------------------------
# Step 1 — Cross-compile Go agent binary
# ---------------------------------------------------------------------------
Write-Host "`n[build] Compiling Go agent binary..." -ForegroundColor Cyan

$env:GOOS   = 'windows'
$env:GOARCH = 'amd64'
$env:CGO_ENABLED = '0'

Push-Location $RepoRoot
try {
    & go build -o $AgentExe .
    if ($LASTEXITCODE -ne 0) {
        Write-Error "[build] go build failed (exit code $LASTEXITCODE)"
        exit 1
    }
} finally {
    Pop-Location
}

if (-not (Test-Path $AgentExe)) {
    Write-Error "[build] Expected binary not found: $AgentExe"
    exit 1
}

Write-Host "[build] Agent binary produced: $AgentExe" -ForegroundColor Green

# ---------------------------------------------------------------------------
# Step 2 — Build config_writer helper
# ---------------------------------------------------------------------------
Write-Host "`n[build] Compiling config_writer helper..." -ForegroundColor Cyan

$ConfigWriterSrc = Join-Path $InstallerDir 'config_writer'
$ConfigWriterExe = Join-Path $OutDir 'config_writer.exe'

Push-Location $RepoRoot
try {
    & go build -o $ConfigWriterExe $ConfigWriterSrc
    if ($LASTEXITCODE -ne 0) {
        Write-Error "[build] config_writer build failed (exit code $LASTEXITCODE)"
        exit 1
    }
} finally {
    Pop-Location
}

Write-Host "[build] config_writer produced: $ConfigWriterExe" -ForegroundColor Green

# ---------------------------------------------------------------------------
# Step 3 — Build MSI with WiX v4
# ---------------------------------------------------------------------------
Write-Host "`n[build] Running WiX Toolset to create MSI..." -ForegroundColor Cyan

if (-not (Get-Command 'wix' -ErrorAction SilentlyContinue)) {
    Write-Error @"
[build] 'wix' command not found.
Install WiX Toolset v4 with:
    dotnet tool install --global wix
Then ensure the .NET global tools directory is in your PATH.
"@
    exit 1
}

& wix build $WxsFile -o $MsiOut
if ($LASTEXITCODE -ne 0) {
    Write-Error "[build] wix build failed (exit code $LASTEXITCODE)"
    exit 1
}

if (-not (Test-Path $MsiOut)) {
    Write-Error "[build] Expected MSI not found: $MsiOut"
    exit 1
}

# ---------------------------------------------------------------------------
# Done
# ---------------------------------------------------------------------------
Write-Host "`n========================================" -ForegroundColor Green
Write-Host " Build succeeded!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host " MSI : $MsiOut"
Write-Host " EXE : $AgentExe"
Write-Host ""
