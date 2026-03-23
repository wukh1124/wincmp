@echo off
chcp 65001 > nul
setlocal EnableDelayedExpansion

REM 1. Get the absolute path of the script directory
set "SCRIPT_DIR=%~dp0"

REM 2. Set the absolute path of the exclude list
set "EXCLUDE_FILE=%SCRIPT_DIR%zip-exclude.txt"

REM 3. Go to the project root directory
cd /d "%SCRIPT_DIR%.."
set "PROJECT_ROOT=%CD%"

REM 4. Get the project root folder name
for %%I in (.) do set "PROJECT_DIR=%%~nxI"

REM 5. Safely calculate file count and size
REM    (Write the complex PowerShell script to a temp file first to avoid CMD parsing issues)
echo Scanning project directory information, please wait...
set "PS_TEMP=%SCRIPT_DIR%temp_calc.ps1"
echo $files = Get-ChildItem -Path '%PROJECT_ROOT%' -File -Recurse -Force -ErrorAction SilentlyContinue > "%PS_TEMP%"
echo $count = @($files).Count >> "%PS_TEMP%"
echo $size = ($files ^| Measure-Object -Property Length -Sum).Sum >> "%PS_TEMP%"
echo if ($null -eq $size) { $size = 0 } >> "%PS_TEMP%"
echo $sizeMB = [math]::Round($size / 1MB, 2) >> "%PS_TEMP%"
REM Fix: append an extra | so any trailing \r is pushed into an unused third token
echo Write-Output "$count|$sizeMB|" >> "%PS_TEMP%"

for /f "tokens=1,2 delims=|" %%A in ('powershell -NoProfile -ExecutionPolicy Bypass -File "%PS_TEMP%"') do (
    set "FILE_COUNT=%%A"
    set "TOTAL_SIZE_MB=%%B"
)

REM Delete temp file
del "%PS_TEMP%"

REM 6. Get the datetime string from PowerShell
for /f "delims=" %%a in ('powershell -NoProfile -Command "Get-Date -Format 'yyyyMMdd_HHmmss'"') do set "DATETIME=%%a"

REM 7. Build output file name
set "ZIPFILE=%SCRIPT_DIR%wincmp_%DATETIME%.7z"

REM 8. Show information to the user for confirmation
cls
echo ===================================================
echo WinCMP Project Packaging Wizard
echo ===================================================
echo [Project Name] : %PROJECT_DIR%
echo [Root Path]    : %PROJECT_ROOT%
REM Fix: use delayed expansion to avoid \r or special characters breaking echo
echo [File Count]   : About !FILE_COUNT! files
echo [Raw Size]     : About !TOTAL_SIZE_MB! MB
echo [Output File]  : %ZIPFILE%
if exist "%EXCLUDE_FILE%" (
    set "HAS_EXCLUDE=1"
    echo [Exclude List] : Loaded bat\zip-exclude.txt, contents below:
    echo ---------------------------------------------------
    REM New: display the exclude list content
    type "%EXCLUDE_FILE%"
    echo.
    echo ---------------------------------------------------
) else (
    set "HAS_EXCLUDE=0"
    echo [Exclude List] : Warning! bat\zip-exclude.txt not found, full directory will be compressed!
)
echo ===================================================
echo.

REM Ask user whether to continue
set "USER_CONFIRM="
set /p "USER_CONFIRM=Start compression? (Y/N): "
REM Fix: use delayed expansion to avoid syntax errors caused by empty input or hidden characters
if /i not "!USER_CONFIRM!"=="Y" (
    echo.
    echo Packaging cancelled.
    pause
    exit /b
)

echo.
echo Starting compression, please wait...

REM Check 7z
set "SEVEN_ZIP=C:\Program Files\7-Zip\7z.exe"
if not exist "%SEVEN_ZIP%" set "SEVEN_ZIP=7z"

REM ==========================================
REM 9. Move up one more level before compression
cd ..
REM ==========================================

REM Run compression
if "%HAS_EXCLUDE%"=="1" (
    "%SEVEN_ZIP%" a -t7z "%ZIPFILE%" "%PROJECT_DIR%" -r -x@"%EXCLUDE_FILE%" -mx5
) else (
    "%SEVEN_ZIP%" a -t7z "%ZIPFILE%" "%PROJECT_DIR%" -r -mx5
)

if %errorlevel%==0 (
    echo.
    echo ===================================================
    echo Compression completed successfully!
    echo File saved to: %ZIPFILE%
    echo ===================================================
) else (
    echo.
    echo Compression failed, error code: %errorlevel%
)

REM Switch back to the original script directory after compression
cd /d "%SCRIPT_DIR%"
pause
