# Caddy 2 入門與調試指南 (Windows 11)

本文件說明如何在 Windows 11 環境下手動運行與調試 Caddy 2，協助理解 `wincmp` 的底層運行邏輯。

## 1. 環境準備
推薦使用 [Scoop](https://scoop.sh/) 安裝 Caddy：
```powershell
scoop install caddy
```

## 2. 手動運行 PHP-CGI
`wincmp` 預設使用多端口運行 PHP 以模擬並發。調試時可手動啟動特定版本的 PHP：

**路徑範例 (PHP 8.2):** `C:\wincmp\bin\php\php-8.2.30-Win32-vs16-x64`

### 手動啟動指令：
```powershell
cd /d "C:\wincmp\bin\php\php-8.2.30-Win32-vs16-x64"
.\php-cgi.exe -b 127.0.0.1:9821
.\php-cgi.exe -b 127.0.0.1:9822
.\php-cgi.exe -b 127.0.0.1:9823
```

### 批量啟動腳本 (batch):
```batch
@echo off
set PHP_PATH="C:\wincmp\bin\php\php-8.2.30-Win32-vs16-x64"
start /b "" %PHP_PATH%\php-cgi.exe -b 127.0.0.1:9821
start /b "" %PHP_PATH%\php-cgi.exe -b 127.0.0.1:9822
start /b "" %PHP_PATH%\php-cgi.exe -b 127.0.0.1:9823
echo PHP-CGI 已啟動於端口 9821, 9822, 9823
pause
```

## 3. Caddyfile 操作指令
Caddyfile 通常存放於 `C:\wincmp\conf\Caddyfile`。

| 動作 | 指令 | 說明 |
| :--- | :--- | :--- |
| **語法檢查** | `caddy validate --config Caddyfile` | 運行前必做，檢查語法錯誤 |
| **格式化** | `caddy fmt --overwrite` | 自動美化 Caddyfile 縮排 |
| **前台運行** | `caddy run --config Caddyfile` | **調試推薦**，直接在終端看 Log |
| **啟動背景** | `caddy start --config Caddyfile` | 在背景啟動 Caddy |
| **熱重載** | `caddy reload --config Caddyfile` | 修改設定後無縫更新 |
| **停止** | `caddy stop` | 停止背景運行的 Caddy |

## 4. Windows 服務化 (正式環境)
使用 `nssm` 將 Caddy 掛載為系統服務：
- **Path**: `C:\wincmp\bin\caddy\caddy.exe`
- **Arguments**: `run --config Caddyfile --adapter caddyfile`
- **Startup directory**: `C:\wincmp\conf`

```powershell
nssm install Caddy
nssm start Caddy
```

## 5. 參考 Caddyfile 範例
針對 `wincmp` 專案結構的典型配置：

```caddy
# 自動跳轉 HTTP to HTTPS
http://local.domain.xyz {
    redir https://{host}{uri}
}

https://local.domain.xyz {
    # SSL 憑證路徑 (建議使用正斜線 /)
    tls C:/wincmp/conf/ssl/domain.xyz.chained.crt C:/wincmp/conf/ssl/domain.xyz.key

    # 專案根目錄
    root * C:/wincmp/www/your_project/public

    # 壓縮與 PHP 處理
    encode zstd gzip
    php_fastcgi 127.0.0.1:9821  # 需對應 PHP-CGI 啟動的端口
    
    file_server
    
    # 調試用日誌
    log {
        output file C:/wincmp/logs/caddy/project_access.log
    }
}
```

> **提示**：若 Caddy 無法啟動，請先檢查 80 與 443 端口是否被其他服務（如 IIS 或 Apache）佔用。