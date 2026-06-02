@echo off
chcp 65001 > nul

rem ===================================================
rem  WinCMP Automated Release Wizard
rem ===================================================

powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0bat\release.ps1"

echo.
pause
