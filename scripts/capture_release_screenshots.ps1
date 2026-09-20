# WinCMP Release Screenshot Capture
# Run this script from the WinCMP project root after starting `wails dev`
# (http://localhost:34115)
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$ErrorActionPreference = "Stop"

function Resolve-ProjectRoot {
    $candidates = New-Object System.Collections.Generic.List[string]

    $scriptDir = $null
    if (-not [string]::IsNullOrWhiteSpace($PSScriptRoot)) {
        $scriptDir = $PSScriptRoot
    }
    elseif (-not [string]::IsNullOrWhiteSpace($PSCommandPath)) {
        $scriptDir = Split-Path -Parent $PSCommandPath
    }
    elseif ($MyInvocation.MyCommand.Path) {
        $scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
    }

    if ($scriptDir) {
        $candidates.Add($scriptDir)
        $candidates.Add((Split-Path -Path $scriptDir -Parent))
        $walk = $scriptDir
        for ($i = 0; $i -lt 4; $i++) {
            $parent = Split-Path -Path $walk -Parent
            if (-not $parent -or $parent -eq $walk) { break }
            $candidates.Add($parent)
            $walk = $parent
        }
    }

    $cwd = (Get-Location).Path
    $candidates.Add($cwd)
    $walk = $cwd
    for ($i = 0; $i -lt 3; $i++) {
        $parent = Split-Path -Path $walk -Parent
        if (-not $parent -or $parent -eq $walk) { break }
        $candidates.Add($parent)
        $walk = $parent
    }

    foreach ($dir in $candidates) {
        if ([string]::IsNullOrWhiteSpace($dir)) { continue }
        $verPath = Join-Path $dir "VERSION"
        if (Test-Path -LiteralPath $verPath) {
            $hasMarker = (Test-Path -LiteralPath (Join-Path $dir "wails.json")) -or (Test-Path -LiteralPath (Join-Path $dir "conf"))
            if ($hasMarker) {
                return @{ ProjectRoot = $dir; VersionPath = $verPath }
            }
        }
    }

    foreach ($dir in $candidates) {
        if ([string]::IsNullOrWhiteSpace($dir)) { continue }
        $verPath = Join-Path $dir "VERSION"
        if (Test-Path -LiteralPath $verPath) {
            return @{ ProjectRoot = $dir; VersionPath = $verPath }
        }
    }

    return $null
}

$ProjectRoot = $null
$VersionPath = $null
$resolved = Resolve-ProjectRoot
if ($null -eq $resolved) {
    Write-Host "===================================================" -ForegroundColor Cyan
    Write-Host "     WinCMP Release Screenshot Capture            " -ForegroundColor Cyan
    Write-Host "===================================================" -ForegroundColor Cyan
    Write-Host "[Error] Cannot locate project root (VERSION not found)." -ForegroundColor Red
    Write-Host "  PSScriptRoot  = '$PSScriptRoot'" -ForegroundColor Yellow
    Write-Host "  PSCommandPath = '$PSCommandPath'" -ForegroundColor Yellow
    Write-Host "  Get-Location  = '$(Get-Location)'" -ForegroundColor Yellow
    Write-Host "Please run from the WinCMP project root:" -ForegroundColor Yellow
    Write-Host "  powershell -ExecutionPolicy Bypass -File .\scripts\capture_release_screenshots.ps1" -ForegroundColor Yellow
    exit 1
}

$ProjectRoot = $resolved.ProjectRoot
$VersionPath = $resolved.VersionPath
Set-Location -Path $ProjectRoot

Write-Host "===================================================" -ForegroundColor Cyan
Write-Host "     WinCMP Release Screenshot Capture            " -ForegroundColor Cyan
Write-Host "===================================================" -ForegroundColor Cyan
Write-Host "[0] Project root: $ProjectRoot" -ForegroundColor DarkGray

# 1) Read target version
$NewVersion = (Get-Content -LiteralPath $VersionPath -Raw).Trim()
if ($NewVersion.StartsWith("v")) {
    $NewVersion = $NewVersion.Substring(1)
}
Write-Host "[1] Target version: v$NewVersion" -ForegroundColor Green

# 2) Backup version = previous published release (before this release)
# 不再使用 release_info.json；以 git tag 取得上一版
$PrevVersion = $null
try {
    Push-Location $ProjectRoot
    $tags = @(git tag -l --sort=-v:refname)
    Pop-Location
    foreach ($t in $tags) {
        if (-not $t) { continue }
        $clean = $t.TrimStart('v')
        if ($clean -and ($clean -ne $NewVersion)) {
            $PrevVersion = $clean
            break
        }
    }
}
catch {
    Write-Host "    [Warn] Failed to read git tags" -ForegroundColor Yellow
}

if (-not $PrevVersion) {
    $PrevVersion = "unknown"
}
Write-Host "[2] Backup folder will use previous version: v$PrevVersion" -ForegroundColor Green

# 3) Backup current screenshots to screenshot/backup/v{prev}/
$ShotRoot = Join-Path $ProjectRoot "screenshot"
$BackupDir = Join-Path (Join-Path $ShotRoot "backup") ("v" + $PrevVersion)

$themeDirs = @("dark", "sketch")
$hasExisting = $false
foreach ($td in $themeDirs) {
    $src = Join-Path $ShotRoot $td
    if (Test-Path -LiteralPath $src) {
        $hasExisting = $true
        break
    }
}

if ($hasExisting) {
    Write-Host "[3] Backing up current screenshots -> screenshot/backup/v$PrevVersion/ ..." -ForegroundColor Gray
    if (Test-Path -LiteralPath $BackupDir) {
        Write-Host "    -> Backup dir exists, cleaning..." -ForegroundColor Yellow
        Remove-Item -LiteralPath $BackupDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $BackupDir -Force | Out-Null
    foreach ($td in $themeDirs) {
        $src = Join-Path $ShotRoot $td
        if (Test-Path -LiteralPath $src) {
            $dst = Join-Path $BackupDir $td
            Copy-Item -LiteralPath $src -Destination $dst -Recurse -Force
            Write-Host "    -> Copied $td" -ForegroundColor DarkGray
        }
    }
}
else {
    Write-Host "[3] No existing screenshots found, skip backup." -ForegroundColor Yellow
}

# 4) Check Wails dev server
Write-Host "[4] Checking Wails dev server at http://localhost:34115 ..." -ForegroundColor Gray
$targetUrl = "http://localhost:34115"
$devReady = $false
try {
    $resp = Invoke-WebRequest -Uri $targetUrl -TimeoutSec 3 -UseBasicParsing
    if ($resp.StatusCode -eq 200) {
        $devReady = $true
    }
}
catch {
    $devReady = $false
}

if (-not $devReady) {
    Write-Host "    [Error] Wails dev server is not reachable." -ForegroundColor Red
    Write-Host "    Please run in another terminal first:" -ForegroundColor Yellow
    Write-Host "      wails dev" -ForegroundColor Yellow
    Write-Host "    Then re-run from project root:" -ForegroundColor Yellow
    Write-Host "      powershell -ExecutionPolicy Bypass -File .\scripts\capture_release_screenshots.ps1" -ForegroundColor Yellow
    exit 1
}
Write-Host "    -> Wails dev server is ready." -ForegroundColor Green

# 5) Run Playwright capture
Write-Host "[5] Running capture.cjs (Playwright)..." -ForegroundColor Gray
$FrontendDir = Join-Path $ProjectRoot "frontend"
if (-not (Test-Path -LiteralPath $FrontendDir)) {
    Write-Error "frontend directory not found at $FrontendDir"
}
Set-Location -Path $FrontendDir
node scripts/capture.cjs
if ($LASTEXITCODE -ne 0) {
    Write-Error "Screenshot capture failed."
}

Set-Location -Path $ProjectRoot
Write-Host "===================================================" -ForegroundColor Green
Write-Host "[Success] Screenshots updated for v$NewVersion" -ForegroundColor Green
Write-Host "Previous set backed up to: screenshot/backup/v$PrevVersion/" -ForegroundColor Green
Write-Host "===================================================" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "  1. Review screenshot/dark and screenshot/sketch" -ForegroundColor White
Write-Host "  2. git add screenshot/ VERSION release_note/ conf/dependencies.json" -ForegroundColor White
Write-Host "  3. Commit, push, then tag v$NewVersion" -ForegroundColor White