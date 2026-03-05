#Requires -Version 5.1
<#
.SYNOPSIS
    Auto-detect POSitouch installation paths.

.DESCRIPTION
    Locates spcwin.exe on the local machine and extracts XML directory paths
    from spcwin.ini.  Returns a hashtable suitable for consumption by build.ps1
    or a WiX custom action.

.PARAMETER SpcwinHint
    Optional path to spcwin.exe (or the directory containing it).
    When supplied the default C:\SC location and the file-system scan are
    skipped and this path is tried first.

.OUTPUTS
    [hashtable] with keys:
        SpcwinPath      — full path to spcwin.exe
        SpcwinDir       — directory containing spcwin.exe
        XMLOutPath      — open-tickets XML directory
        XMLClosePath    — closed-tickets XML directory
        XMLInOrderPath  — inbound-order XML directory
        DetectionMethod — how the exe was found: "default", "hint", "scan", or "not_found"

.EXAMPLE
    $info = .\detect.ps1
    $info = .\detect.ps1 -SpcwinHint 'D:\POS\SC\spcwin.exe'
#>

[CmdletBinding()]
param(
    [Parameter(Mandatory = $false)]
    [string]$SpcwinHint = ''
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# ---------------------------------------------------------------------------
# Helper: read a key from a simple INI file (no section scoping needed)
# ---------------------------------------------------------------------------
function Get-IniValue {
    param(
        [string]$IniPath,
        [string]$Key
    )
    if (-not (Test-Path $IniPath)) { return $null }
    foreach ($line in Get-Content -LiteralPath $IniPath -ErrorAction SilentlyContinue) {
        if ($line -match "^\s*$([regex]::Escape($Key))\s*=\s*(.+)$") {
            return $Matches[1].Trim()
        }
    }
    return $null
}

# ---------------------------------------------------------------------------
# Helper: build result object from a found spcwin.exe path
# ---------------------------------------------------------------------------
function Build-Result {
    param(
        [string]$ExePath,
        [string]$Method
    )

    $dir    = Split-Path -Parent $ExePath
    $iniPath = Join-Path $dir 'spcwin.ini'

    # Try to read XML paths from the INI; fall back to conventional sub-dirs.
    $xmlOut     = Get-IniValue -IniPath $iniPath -Key 'XMLOutPath'
    $xmlClose   = Get-IniValue -IniPath $iniPath -Key 'XMLClosePath'
    $xmlInOrder = Get-IniValue -IniPath $iniPath -Key 'XMLInOrderPath'

    if ([string]::IsNullOrWhiteSpace($xmlOut))     { $xmlOut     = Join-Path $dir 'XML'      }
    if ([string]::IsNullOrWhiteSpace($xmlClose))   { $xmlClose   = Join-Path $dir 'XMLCLOSE' }
    if ([string]::IsNullOrWhiteSpace($xmlInOrder)) { $xmlInOrder = Join-Path $dir 'INORDER'  }

    return @{
        SpcwinPath      = $ExePath
        SpcwinDir       = $dir
        XMLOutPath      = $xmlOut
        XMLClosePath    = $xmlClose
        XMLInOrderPath  = $xmlInOrder
        DetectionMethod = $Method
    }
}

# ---------------------------------------------------------------------------
# 1. Try user-supplied hint first
# ---------------------------------------------------------------------------
if ($SpcwinHint -ne '') {
    $candidate = $SpcwinHint

    # If the hint is a directory, append the exe name.
    if (Test-Path -LiteralPath $candidate -PathType Container) {
        $candidate = Join-Path $candidate 'spcwin.exe'
    }

    if (Test-Path -LiteralPath $candidate -PathType Leaf) {
        Write-Verbose "spcwin.exe found via hint: $candidate"
        return Build-Result -ExePath $candidate -Method 'hint'
    }

    Write-Warning "SpcwinHint '$SpcwinHint' did not resolve to a valid spcwin.exe — continuing with auto-detection."
}

# ---------------------------------------------------------------------------
# 2. Check the well-known default location
# ---------------------------------------------------------------------------
$defaultPath = 'C:\SC\spcwin.exe'
if (Test-Path -LiteralPath $defaultPath -PathType Leaf) {
    Write-Verbose "spcwin.exe found at default location: $defaultPath"
    return Build-Result -ExePath $defaultPath -Method 'default'
}

# ---------------------------------------------------------------------------
# 3. Scan C:\ up to 4 directory levels deep
# ---------------------------------------------------------------------------
Write-Verbose 'spcwin.exe not at default location — scanning C:\ (depth ≤ 4)...'

function Find-SpcwinRecursive {
    param(
        [string]$Path,
        [int]$MaxDepth,
        [int]$CurrentDepth = 0
    )

    if ($CurrentDepth -gt $MaxDepth) { return $null }

    $exePath = Join-Path $Path 'spcwin.exe'
    if (Test-Path -LiteralPath $exePath -PathType Leaf) {
        return $exePath
    }

    try {
        $subDirs = Get-ChildItem -LiteralPath $Path -Directory -ErrorAction SilentlyContinue
    } catch {
        return $null
    }

    foreach ($sub in $subDirs) {
        $found = Find-SpcwinRecursive -Path $sub.FullName -MaxDepth $MaxDepth -CurrentDepth ($CurrentDepth + 1)
        if ($null -ne $found) { return $found }
    }

    return $null
}

$scanned = Find-SpcwinRecursive -Path 'C:\' -MaxDepth 4 -CurrentDepth 0
if ($null -ne $scanned) {
    Write-Verbose "spcwin.exe found by scan: $scanned"
    return Build-Result -ExePath $scanned -Method 'scan'
}

# ---------------------------------------------------------------------------
# 4. Not found — return a "not_found" result with empty paths
# ---------------------------------------------------------------------------
Write-Warning 'spcwin.exe was not found. Please supply the path manually via -SpcwinHint.'

return @{
    SpcwinPath      = ''
    SpcwinDir       = ''
    XMLOutPath      = ''
    XMLClosePath    = ''
    XMLInOrderPath  = ''
    DetectionMethod = 'not_found'
}
