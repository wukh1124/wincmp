@echo off
chcp 65001 > nul

REM 切換到專案根目錄 (bat 的上一層)
cd /d "%~dp0.."

echo.
echo 啟動 Caddy...
echo 設定檔: conf\Caddyfile
echo 按 Ctrl+C 停止
echo.
bin\caddy\caddy-2.11.1\caddy.exe run --config conf\Caddyfile --adapter caddyfile --watch

pause
