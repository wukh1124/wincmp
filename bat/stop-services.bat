@echo off
:: 設定編碼為 UTF-8 以正確顯示中文
chcp 65001 >nul
title 關閉網頁伺服器環境 (Caddy, MariaDB, PHP)

echo ===========================================
echo   正在強制關閉 Web 環境相關進程...
echo ===========================================

:: 1. 關閉 PHP-CGI
taskkill /F /IM php-cgi.exe /T 2>nul
if %errorlevel%==0 (echo [成功] PHP-CGI 已關閉) else (echo [跳過] 未發現運行的 PHP-CGI)

:: 2. 關閉 Caddy
taskkill /F /IM caddy.exe /T 2>nul
if %errorlevel%==0 (echo [成功] Caddy 已關閉) else (echo [跳過] 未發現運行的 Caddy)

:: 3. 關閉 MariaDB (核心進程名為 mysqld.exe)
taskkill /F /IM mysqld.exe /T 2>nul
if %errorlevel%==0 (echo [成功] MariaDB 已關閉) else (echo [跳過] 未發現運行的 MariaDB)

echo -------------------------------------------
echo   所有指定進程已處理完畢。
echo ===========================================
pause
