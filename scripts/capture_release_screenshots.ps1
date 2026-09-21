# WinCMP Release Screenshot Capture
# (相容包裝：建議直接執行 `node scripts/capture.js`)
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
Set-Location -Path $ProjectRoot

Write-Host "===================================================" -ForegroundColor Cyan
Write-Host "     WinCMP Release Screenshot Capture (Wrapper)  " -ForegroundColor Cyan
Write-Host "===================================================" -ForegroundColor Cyan
Write-Host "[提示] 截圖腳本已現代化遷移至純 Node.js！" -ForegroundColor Yellow
Write-Host "日後推薦直接在終端執行：" -ForegroundColor White
Write-Host "  node scripts/capture.js" -ForegroundColor Green
Write-Host "  node scripts/capture.js --sync-website (截圖完自動同步至官網目錄)" -ForegroundColor Green
Write-Host "===================================================" -ForegroundColor Cyan
Write-Host ""

node scripts/capture.js
if ($LASTEXITCODE -ne 0) {
    Write-Error "Screenshot capture failed."
}