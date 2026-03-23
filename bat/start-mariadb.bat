@echo off
chcp 65001 > nul

cd /d "%~dp0.."
set "ROOT=%CD%"

set "MARIADB_BIN=%ROOT%\bin\mariadb\mariadb-11.4.10\bin"
set "MARIADB_DATA=%ROOT%\data\mariaDB"
set "MARIADB_CONF=%ROOT%\conf\my.ini"

echo.
echo [1/3] 檢查設定檔...
if not exist "%MARIADB_CONF%" (
    echo [錯誤] 找不到 %MARIADB_CONF%
    echo        請先建立 conf\my.ini 設定檔。
    pause
    exit /b 1
)
echo       OK：%MARIADB_CONF%

echo.
echo [2/3] 檢查資料目錄初始化狀態...
if not exist "%MARIADB_DATA%\mysql\" (
    echo       資料目錄尚未初始化，開始初始化...
    if not exist "%MARIADB_DATA%" mkdir "%MARIADB_DATA%"

    "%MARIADB_BIN%\mariadb-install-db.exe" --datadir="%MARIADB_DATA%"

    if errorlevel 1 (
        echo [錯誤] 初始化失敗！請檢查路徑或權限。
        pause
        exit /b 1
    )
    echo       初始化完成！Root 帳號為空密碼。
) else (
    echo       已初始化，略過。
)

echo.
echo [3/3] 啟動 MariaDB Server（非服務模式）...
echo       設定檔 : %MARIADB_CONF%
echo       資料目錄: %MARIADB_DATA%
echo       按 Ctrl+C 可停止伺服器
echo.
"%MARIADB_BIN%\mariadbd.exe" --defaults-file="%MARIADB_CONF%" --console

pause
