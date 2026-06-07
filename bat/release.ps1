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
$RequiredFiles = @("readme.md", "CHANGELOG.md", "CHANGELOG_zh.md", "LICENSE")
$MissingFiles = @()

foreach ($file in $RequiredFiles) {
    $checkPath = Join-Path $TargetDir $file
    if (-not (Test-Path $checkPath)) {
        $MissingFiles += $file
    }
}

if ($MissingFiles.Count -gt 0) {
    Write-Host "    ⚠️ Warning! Missing files: $($MissingFiles -join ', ')" -ForegroundColor Yellow
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

# 9. Generate GitHub Release Suggestion Document
Write-Host "[9] Generating GitHub Release suggestion document..." -ForegroundColor Gray

$ChangelogPath = Join-Path $ProjectRoot "packaging\wincmp\CHANGELOG.md"
$ReleaseNotesContent = ""

if (Test-Path $ChangelogPath) {
    $ChangelogLines = Get-Content $ChangelogPath -Encoding utf8
    $StartIndex = -1
    $EndIndex = -1
    
    # Find the line matching current version, e.g., "## [1.2.5]" or "## 1.2.5"
    for ($i = 0; $i -lt $ChangelogLines.Count; $i++) {
        $line = $ChangelogLines[$i]
        if ($line -match "^##\s+\[?$Version\]?") {
            $StartIndex = $i + 1
            break
        }
    }
    
    if ($StartIndex -ne -1) {
        # Find next heading as end index
        for ($i = $StartIndex; $i -lt $ChangelogLines.Count; $i++) {
            $line = $ChangelogLines[$i]
            if ($line -match "^##\s+") {
                $EndIndex = $i - 1
                break
            }
        }
        if ($EndIndex -eq -1) {
            $EndIndex = $ChangelogLines.Count - 1
        }
        
        $NotesLines = $ChangelogLines[$StartIndex..$EndIndex]
        $ReleaseNotesContent = ($NotesLines -join "`n").Trim()
    }
}

if (-not $ReleaseNotesContent) {
    $ReleaseNotesContent = "- Maintenance updates and stability improvements."
}

# Construct release suggestion template
$ReleaseDocTemplate = @'
# GitHub Release Suggestion (v{0})

## Release Title
WinCMP v{0}

## Tag Name
v{0}

## Release Notes (Markdown)
---
## WinCMP v{0}

This release introduces new features, updates, and fixes to WinCMP.

### What's Changed
{1}

### Getting Started
1. Download `wincmp-v{0}-win-x64.zip`.
2. Extract the archive to any folder on your system.
3. Double-click `WinCMP_v{0}.exe` to launch the control panel.
---
'@

$ReleaseDocContent = $ReleaseDocTemplate -f $Version, $ReleaseNotesContent
$ReleaseNotesFile = Join-Path $ReleaseParentDir "release_notes_v$Version.md"
$utf8NoBom = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllText($ReleaseNotesFile, $ReleaseDocContent, $utf8NoBom)
Write-Host "    -> Generated GitHub Release notes: $ReleaseNotesFile" -ForegroundColor Green

# Return to root
Set-Location -Path $ProjectRoot

Write-Host "===================================================" -ForegroundColor Green
Write-Host "✨ Automated release completed successfully!" -ForegroundColor Green
Write-Host "Saved to: $ZipFile" -ForegroundColor Green
Write-Host "===================================================" -ForegroundColor Green
