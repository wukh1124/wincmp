# WinCMP Automated Release Script
# Ensure Console output encoding is UTF-8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$ErrorActionPreference = "Stop"

Write-Host "===================================================" -ForegroundColor Cyan
Write-Host "     WinCMP Automated Release Wizard (Wails) " -ForegroundColor Cyan
Write-Host "===================================================" -ForegroundColor Cyan

# 1. Get project paths
$ScriptDir = $PSScriptRoot
$ProjectRoot = Split-Path -Path $ScriptDir -Parent
Set-Location -Path $ProjectRoot

Write-Host "[1] Reading Version from VERSION..." -ForegroundColor Gray
$VersionPath = Join-Path $ProjectRoot "VERSION"
if (-not (Test-Path $VersionPath)) {
    Write-Error "VERSION file not found! Make sure you run this script within the WinCMP project."
}

$Version = (Get-Content $VersionPath -Raw).Trim()
if ($Version.StartsWith("v")) {
    $Version = $Version.Substring(1)
}
Write-Host "    -> Version detected: v$Version" -ForegroundColor Green

# 2. Compile release build using Wails
Write-Host "[2] Compiling release build with Wails..." -ForegroundColor Gray

$wailsCheck = Get-Command "wails" -ErrorAction SilentlyContinue
if (-not $wailsCheck) {
    Write-Error "Wails CLI not found! Please install it using 'go install github.com/wailsapp/wails/v2/cmd/wails@latest'."
}

$BuildFailed = $false
try {
    Write-Host "    -> Running 'wails build -clean -ldflags ""-s -w -X main.AppVersion=v$Version"" -o wincmp.exe'..." -ForegroundColor DarkGray
    wails build -clean -ldflags "-s -w -X main.AppVersion=v$Version" -o wincmp.exe
    Write-Host "    -> Wails build succeeded!" -ForegroundColor Green
} catch {
    $BuildFailed = $true
    Write-Error "Wails compilation failed! Please check your configuration."
}

if ($BuildFailed) {
    exit 1
}

# 3. Prepare release folder structure
$ReleaseParentDir = Join-Path (Split-Path -Path $ProjectRoot -Parent) "wincmp-release-only"
$ReleaseDirName = "wincmp_v$Version"
$TargetDir = Join-Path $ReleaseParentDir $ReleaseDirName

Write-Host "[3] Preparing clean release directory..." -ForegroundColor Gray
Write-Host "    -> Release Parent: $ReleaseParentDir" -ForegroundColor DarkGray
Write-Host "    -> Target Directory: $TargetDir" -ForegroundColor DarkGray

if (-not (Test-Path $ReleaseParentDir)) {
    New-Item -ItemType Directory -Path $ReleaseParentDir -Force | Out-Null
}

if (Test-Path $TargetDir) {
    Write-Host "    -> Target directory already exists, cleaning old files..." -ForegroundColor Yellow
    Remove-Item -Path $TargetDir -Recurse -Force
}
New-Item -ItemType Directory -Path $TargetDir -Force | Out-Null

# 4. Copy template files
Write-Host "[4] Copying release template..." -ForegroundColor Gray
$TemplateDir = Join-Path $ProjectRoot "packaging\wincmp"
if (-not (Test-Path $TemplateDir)) {
    Write-Error "Template directory packaging\wincmp not found!"
}
Copy-Item -Path "$TemplateDir\*" -Destination $TargetDir -Recurse -Force

# 5. Copy and rename executable
Write-Host "[5] Copying and renaming executable..." -ForegroundColor Gray
$BuiltExe = Join-Path $ProjectRoot "build\bin\wincmp.exe"
$TargetExe = Join-Path $TargetDir "WinCMP_v$Version.exe"

if (-not (Test-Path $BuiltExe)) {
    Write-Error "Could not find built wincmp.exe at $BuiltExe"
}
Copy-Item -Path $BuiltExe -Destination $TargetExe -Force
Write-Host "    -> Created executable: WinCMP_v$Version.exe" -ForegroundColor Green

# Copy standalone executable to release parent directory for auto updates
$StandaloneExe = Join-Path $ReleaseParentDir "WinCMP_v$Version.exe"
Copy-Item -Path $BuiltExe -Destination $StandaloneExe -Force
Write-Host "    -> Created standalone executable for updates: $StandaloneExe" -ForegroundColor Green

# 6. Clean redundant files (.gitkeep, .example, logs, backups, etc.)
Write-Host "[6] Cleaning redundant and test files..." -ForegroundColor Gray

# Remove .gitkeep files
$Gitkeeps = Get-ChildItem -Path $TargetDir -Filter ".gitkeep" -Recurse -Force
if ($Gitkeeps) {
    $Gitkeeps | Remove-Item -Force
    Write-Host "    -> Cleaned $($Gitkeeps.Count) .gitkeep files" -ForegroundColor DarkGray
}

# Remove .example files
$Examples = Get-ChildItem -Path $TargetDir -Filter "*.example" -Recurse -Force
if ($Examples) {
    $Examples | Remove-Item -Force
    Write-Host "    -> Cleaned $($Examples.Count) .example files" -ForegroundColor DarkGray
}

# Empty logs
$LogsPath = Join-Path $TargetDir "logs"
if (Test-Path $LogsPath) {
    Get-ChildItem -Path $LogsPath -File -Force | Remove-Item -Force
    Write-Host "    -> Cleared logs/ directory" -ForegroundColor DarkGray
}

# Clean data subfolders contents but keep the folder structure
$DataPath = Join-Path $TargetDir "data"
if (Test-Path $DataPath) {
    $SubDirs = Get-ChildItem -Path $DataPath -Directory -Recurse -Force
    foreach ($dir in $SubDirs) {
        Get-ChildItem -Path $dir.FullName -File -Recurse -Force | Remove-Item -Force
    }
    Write-Host "    -> Cleared data/ subdirectory contents" -ForegroundColor DarkGray
}

# 7. Verify required release files
Write-Host "[7] Verifying required documentation..." -ForegroundColor Gray
$RequiredFiles = @("readme.md", "LICENSE")
$MissingFiles = @()

foreach ($file in $RequiredFiles) {
    $checkPath = Join-Path $TargetDir $file
    if (-not (Test-Path $checkPath)) {
        $MissingFiles += $file
    }
}

if ($MissingFiles.Count -gt 0) {
    Write-Host "    [Warning] Missing files: $($MissingFiles -join ', ')" -ForegroundColor Yellow
} else {
    Write-Host "    -> All required documentation verified!" -ForegroundColor Green
}

# 8. Compress release files
Write-Host "[8] Compressing release package..." -ForegroundColor Gray

# Look for 7z.exe
$7zPaths = @(
    "C:\Program Files\7-Zip\7z.exe",
    "C:\Program Files (x86)\7-Zip\7z.exe"
)
$7zExe = $null

foreach ($p in $7zPaths) {
    if (Test-Path $p) {
        $7zExe = $p
        break
    }
}

if (-not $7zExe) {
    $cmdCheck = Get-Command "7z" -ErrorAction SilentlyContinue
    if ($cmdCheck) {
        $7zExe = "7z"
    }
}

$Arch = "x64"
$ZipFile = Join-Path $ReleaseParentDir "wincmp-v$Version-win-$Arch.zip"
if (Test-Path $ZipFile) {
    Remove-Item -Path $ZipFile -Force
}

if ($7zExe) {
    Write-Host "    -> 7-Zip found. Compressing to .zip..." -ForegroundColor DarkGray
    
    # Change location to keep relative path structure in archive
    Set-Location -Path $ReleaseParentDir
    & $7zExe a -tzip $ZipFile $ReleaseDirName -mx5 | Out-Null
    
    Write-Host "    -> Successfully generated: $ZipFile" -ForegroundColor Green
} else {
    # Fallback to Compress-Archive
    Write-Host "    -> 7-Zip not found. Using PowerShell Compress-Archive for .zip fallback..." -ForegroundColor Yellow
    
    Set-Location -Path $ReleaseParentDir
    Compress-Archive -Path $ReleaseDirName -DestinationPath $ZipFile -Force
    Write-Host "    -> Successfully generated: $ZipFile" -ForegroundColor Green
}
# 9. Validate Release Notes (release_note 為唯一內容來源)
Write-Host "[9] Validating Release Notes in release_note directory..." -ForegroundColor Gray

$ReleaseNoteDir = Join-Path $ProjectRoot "release_note"
$VersionDir = Join-Path $ReleaseNoteDir "v$Version"
$EnReleaseFile = Join-Path $VersionDir "release_notes.md"
$ZhReleaseFile = Join-Path $VersionDir "release_notes_zh.md"
$utf8NoBom = New-Object System.Text.UTF8Encoding($false)

if (-not (Test-Path $VersionDir)) {
    New-Item -ItemType Directory -Path $VersionDir -Force | Out-Null
    Write-Host "    -> Created release note directory: $VersionDir" -ForegroundColor DarkGray
}

# release_notes 為唯一對外內容來源（由 wincmp-release skill 事先撰寫）
if (-not (Test-Path $EnReleaseFile) -or (Get-Item $EnReleaseFile).Length -lt 20) {
    $EnTemplate = @"
# WinCMP v$Version

Release date: $((Get-Date).ToString('yyyy-MM-dd'))

This release introduces new features, updates, and fixes to WinCMP.

## What's Changed

- Maintenance updates and stability improvements.

## Getting Started
1. Download ``wincmp-v$Version-win-x64.zip``.
2. Extract the archive to any folder on your system.
3. Double-click ``WinCMP_v$Version.exe`` to launch the control panel.
"@
    [System.IO.File]::WriteAllText($EnReleaseFile, $EnTemplate, $utf8NoBom)
    Write-Host "    [Warn] English release notes missing/empty, wrote stub: $EnReleaseFile" -ForegroundColor Yellow
} else {
    Write-Host "    -> English release notes present: $EnReleaseFile" -ForegroundColor Green
}

if (-not (Test-Path $ZhReleaseFile) -or (Get-Item $ZhReleaseFile).Length -lt 20) {
    $ZhTemplate = @"
# WinCMP v$Version

發布日期：$((Get-Date).ToString('yyyy-MM-dd'))

此版本為 WinCMP 帶來了新的功能、更新與修正。

## What's Changed

- 維護更新與穩定性優化。

## Getting Started
1. 下載 ``wincmp-v$Version-win-x64.zip``。
2. 解壓縮至您系統中的任何資料夾。
3. 按兩下 ``WinCMP_v$Version.exe`` 啟動控制面板。
"@
    [System.IO.File]::WriteAllText($ZhReleaseFile, $ZhTemplate, $utf8NoBom)
    Write-Host "    [Warn] Chinese release notes missing/empty, wrote stub: $ZhReleaseFile" -ForegroundColor Yellow
} else {
    Write-Host "    -> Chinese release notes present: $ZhReleaseFile" -ForegroundColor Green
}

# 9.1 Notes 須含發布日期行（不再寫 release_info.json；版號以 VERSION 為準）
function Get-ReleaseDateFromNotes {
    param([string]$Text)
    if ([string]::IsNullOrWhiteSpace($Text)) { return $null }
    $patterns = @(
        '(?m)^Release\s*date\s*[:\uFF1A]\s*(\d{4}-\d{2}-\d{2})',
        '(?m)^(?:\u767C\u5E03\u65E5\u671F|\u53D1\u5E03\u65E5\u671F)\s*[:\uFF1A]\s*(\d{4}-\d{2}-\d{2})'
    )
    foreach ($p in $patterns) {
        $m = [regex]::Match($Text, $p)
        if ($m.Success) { return $m.Groups[1].Value }
    }
    return $null
}

$EnNotesText = if (Test-Path $EnReleaseFile) { Get-Content -LiteralPath $EnReleaseFile -Raw -Encoding UTF8 } else { "" }
$ZhNotesText = if (Test-Path $ZhReleaseFile) { Get-Content -LiteralPath $ZhReleaseFile -Raw -Encoding UTF8 } else { "" }
$NotesDate = Get-ReleaseDateFromNotes -Text $EnNotesText
if (-not $NotesDate) { $NotesDate = Get-ReleaseDateFromNotes -Text $ZhNotesText }
if (-not $NotesDate) {
    Write-Host "    [Warn] Release notes missing date line (Release date: YYYY-MM-DD / 發布日期：YYYY-MM-DD)" -ForegroundColor Yellow
} else {
    Write-Host "    -> Release date from notes: $NotesDate" -ForegroundColor Green
}

# 若舊檔仍存在則移除，避免與 VERSION 重複維護
$InfoJsonPath = Join-Path $ReleaseNoteDir "release_info.json"
if (Test-Path -LiteralPath $InfoJsonPath) {
    Remove-Item -LiteralPath $InfoJsonPath -Force
    Write-Host "    -> Removed obsolete release_info.json" -ForegroundColor Yellow
}

# Return to root
Set-Location -Path $ProjectRoot

Write-Host "===================================================" -ForegroundColor Green
Write-Host "[Success] Automated release completed successfully!" -ForegroundColor Green
Write-Host "Saved to: $ZipFile" -ForegroundColor Green
Write-Host "===================================================" -ForegroundColor Green

