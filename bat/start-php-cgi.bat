@echo off
chcp 65001 > nul
setlocal enabledelayedexpansion

REM 切換到專案根目錄 (bat 的上一層)
cd /d "%~dp0.."

set PHP_VER=php-8.2.30
set PHP_DIR=bin\php\%PHP_VER%
set PHP_INI=conf\php\php.ini

echo.
echo 啟動 PHP-CGI %PHP_VER% (3 個行程做負載平衡)...
echo   設定檔: %PHP_INI%
echo   擴充目錄: %PHP_DIR%\ext

REM 修正 start 指令：第一個 "" 是標題，避免路徑被當成標題
start "" /b "%PHP_DIR%\php-cgi.exe" -c "%PHP_INI%" -d "extension_dir=%PHP_DIR%\ext" -b 127.0.0.1:38200
start "" /b "%PHP_DIR%\php-cgi.exe" -c "%PHP_INI%" -d "extension_dir=%PHP_DIR%\ext" -b 127.0.0.1:38201
start "" /b "%PHP_DIR%\php-cgi.exe" -c "%PHP_INI%" -d "extension_dir=%PHP_DIR%\ext" -b 127.0.0.1:38202

echo PHP-CGI %PHP_VER% 已啟動: 38200, 38201, 38202
echo.
pause
