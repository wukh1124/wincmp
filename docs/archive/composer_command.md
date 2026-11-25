# WinCMP Composer 使用指南

## 目錄

1. [簡介](#1-簡介)
2. [目錄結構](#2-目錄結構)
3. [快速開始](#3-快速開始)
4. [Composer 版本與 PHP 版本對應](#4-composer-版本與-php-版本對應)
5. [常用指令詳解](#5-常用指令詳解)
6. [Laravel 專案操作實例](#6-laravel-專案操作實例)
7. [依賴管理進階](#7-依賴管理進階)
8. [常見問題與解決方案](#8-常見問題與解決方案)
9. [速查表](#9-速查表)

---

## 1. 簡介

Composer 是 PHP 的依賴管理工具，用於管理 Laravel 等 PHP 專案的套件依賴。WinCMP 支援在同一系統中同時管理**多個 PHP 版本**與**單一或多個 Composer 版本**，並透過專案對應的 `php.exe` 執行 `composer.phar`。

### 1.1 WinCMP 的目錄結構

```text
WinCMP bin/
├── composer/
│   └── composer-2.11.2/
│       ├── composer.phar
│       └── composer.bat        # 由 WinCMP 下載安裝時自動產生
├── php/
│   ├── php-8.5.13/
│   │   ├── php.exe
│   │   └── php-cgi.exe
│   ├── php-8.4.28/
│   ├── php-8.3.37/
│   ├── php-8.2.36/
│   ├── php-7.4.33/
│   └── php-7.3.33/
```

---

## 2. 目錄結構

### 2.1 Composer 版本掃描規則

WinCMP 會自動掃描 `bin/composer/` 目錄下格式為 `composer-<版本號>/` 的資料夾（透過「依賴庫管理」下載時會自動建立 `composer.phar` 與 `composer.bat`）：

```text
bin/composer/
├── composer-2.11.2/
│   ├── composer.phar
│   └── composer.bat         # 目前 scanner.go 依此檔案識別版本
└── composer.phar            # 不識別（需置於 composer-<版本號>/ 子目錄中）
```

> 若手動放入其他版本的 `composer.phar`（例如 `bin/composer/composer-2.2.25/`），需一併建立內容為 `@php "%~dp0composer.phar" %*` 的 `composer.bat` 才能被目前的掃描器辨識。

### 2.2 執行原理

WinCMP 使用指定版本的 PHP CLI 直接執行 `composer.phar`：

```cmd
# 基本格式
<PHP路徑>\php.exe <Composer路徑>\composer.phar <指令>

# 範例（於 WinCMP 根目錄執行）
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar install
```

---

## 3. 快速開始

### 3.1 基本語法

```cmd
<PHP路徑>\php.exe <Composer路徑>\composer.phar <指令> [選項]
```

### 3.2 查看版本

```cmd
:: 查看 Composer 版本
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar -V

:: 查看 PHP 版本（驗證環境）
bin\php\php-8.2.36\php.exe -v
```

### 3.3 建立新專案

```cmd
:: 建立 Laravel 12 專案（使用 PHP 8.2+ 與 Composer 2.11）
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar create-project laravel/laravel www\my-laravel-app "^12.0"

:: 建立 Laravel 11 專案（使用 PHP 8.2+ 與 Composer 2.11）
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar create-project laravel/laravel www\my-laravel-app "^11.0"
```

---

## 4. Composer 版本與 PHP 版本對應

### 4.1 官方相容性說明

* **Composer 2.3+ ～ 2.11.x（最新穩定版）**：官方最低需求為 **PHP 7.2.5**，完整支援 **PHP 7.3 / 7.4 / 8.0 / 8.1 / 8.2 / 8.3 / 8.4 / 8.5**。
* **Composer 2.2 LTS**：最低支援 **PHP 5.3.2**（僅在維護 PHP 7.2.4 以下舊系統時需要）。

因此，WinCMP 預設安裝單一最新版 Composer（如 v2.11.2）即可覆蓋所有內建 PHP 版本（PHP 7.3 ～ 8.5）。

### 4.2 版本相容性矩陣

| PHP 版本 | Composer 2.2 LTS | Composer 2.11.x（最新） |
|----------|------------------|-------------------------|
| PHP 8.5  | 不建議           | 支援                    |
| PHP 8.4  | 不建議           | 支援                    |
| PHP 8.3  | 支援             | 支援                    |
| PHP 8.2  | 支援             | 支援                    |
| PHP 7.4  | 支援             | 支援                    |
| PHP 7.3  | 支援             | 支援                    |
| PHP <= 7.2.4 | 支援         | 不支援（需 2.2 LTS）    |

### 4.3 選擇建議

* **一般專案（PHP 7.3 ～ 8.5）**：統一使用最新 Composer 2.x，搭配專案設定的對應 PHP 版本執行。
* **Laravel 12**：PHP 8.2+、Composer 2.x。
* **Laravel 11**：PHP 8.2+、Composer 2.x。
* **Laravel 10**：PHP 8.1+（WinCMP 可用 PHP 8.2+）、Composer 2.x。
* **跨版本部署注意**：若本機與正式機 PHP 版本不同，建議於專案 `composer.json` 設定 `config.platform.php` 鎖定解析基準。

---

## 5. 常用指令詳解

### 5.1 安裝依賴

```cmd
:: 依 composer.lock 精準安裝依賴
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar install

:: 正式環境安裝（排除 dev 依賴並最佳化自動載入）
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar install --no-dev --optimize-autoloader

:: 優先下載 dist 壓縮包
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar install --prefer-dist
```

### 5.2 更新依賴

```cmd
:: 更新所有依賴並寫入 composer.lock
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar update

:: 僅更新特定套件及其連帶依賴
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar update laravel/framework -W

:: 模擬更新（不實際改動檔案）
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar update --dry-run
```

### 5.3 新增套件

```cmd
:: 新增一般套件
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar require guzzlehttp/guzzle

:: 新增開發環境套件
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar require --dev barryvdh/laravel-debugbar

:: 新增指定版本約束
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar require "laravel/sanctum:^4.0"
```

### 5.4 移除套件

```cmd
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar remove guzzlehttp/guzzle
```

### 5.5 檢視、診斷與安全

```cmd
:: 顯示已安裝的套件列表
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar show

:: 顯示特定套件資訊
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar show laravel/framework

:: 顯示可更新的過期套件
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar outdated

:: 檢查已知安全漏洞
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar audit

:: 驗證 composer.json 格式
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar validate

:: 重建類別自動載入（Laravel 常用）
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar dump-autoload -o

:: 清除 Composer 快取
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar clear-cache
```

---

## 6. Laravel 專案操作實例

以下範例假設已進入專案根目錄（例如 `www\my-laravel-app`，因此透過 `..\..\bin\...` 指向 WinCMP 根目錄的執行檔）：

### 6.1 安裝現有 Laravel 專案依賴

```cmd
cd www\my-laravel-app
..\..\bin\php\php-8.2.36\php.exe ..\..\bin\composer\composer-2.11.2\composer.phar install
```

### 6.2 更新 Laravel 專案依賴

```cmd
cd www\my-laravel-app

:: 完整更新
..\..\bin\php\php-8.2.36\php.exe ..\..\bin\composer\composer-2.11.2\composer.phar update

:: 只更新特定第三方套件（不更動其他套件）
..\..\bin\php\php-8.2.36\php.exe ..\..\bin\composer\composer-2.11.2\composer.phar update vendor/package-name
```

### 6.3 搭配 Laravel Artisan 指令

`vendor:publish`、`optimize:clear`、`key:generate` 等屬於 Laravel Artisan 指令，應直接以 `php.exe artisan` 執行，而非透過 Composer：

```cmd
cd www\my-laravel-app

:: 複製環境設定檔並產生應用程式金鑰
copy .env.example .env
..\..\bin\php\php-8.2.36\php.exe artisan key:generate

:: 發布套件設定檔
..\..\bin\php\php-8.2.36\php.exe artisan vendor:publish

:: 清除設定、路由與檢視快取
..\..\bin\php\php-8.2.36\php.exe artisan optimize:clear
```

### 6.4 完整新專案初始化流程

```cmd
:: 1. 於 WinCMP 根目錄建立新專案
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar create-project laravel/laravel www\demo-app

:: 2. 進入專案目錄
cd www\demo-app

:: 3. 新增額外套件
..\..\bin\php\php-8.2.36\php.exe ..\..\bin\composer\composer-2.11.2\composer.phar require laravel/sanctum

:: 4. 產生金鑰並執行資料庫遷移
..\..\bin\php\php-8.2.36\php.exe artisan key:generate
..\..\bin\php\php-8.2.36\php.exe artisan migrate
```

---

## 7. 依賴管理進階

### 7.1 Composer 腳本

```cmd
:: 查看 composer.json 中定義的可用腳本
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar run-script --list

:: 執行自訂腳本
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar run-script post-autoload-dump
```

### 7.2 平台需求處理

```cmd
:: 安裝時忽略平台需求（如 PHP 版本或擴充模組檢查）
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar install --ignore-platform-reqs

:: 更新時忽略平台需求
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar update --ignore-platform-reqs
```

### 7.3 鎖定檔驗證與套件追蹤

```cmd
:: 驗證 composer.lock 是否與 composer.json 同步（常用於 CI）
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar validate --strict

:: 查詢某個套件為何被安裝（相依鏈）
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar why symfony/http-foundation

:: 查詢某個套件為何無法升級至指定版本
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar why-not laravel/framework ^12.0
```

---

## 8. 常見問題與解決方案

### 8.1 PHP 版本不相容

```text
錯誤訊息：
Your requirements could not be resolved to an installable set of packages.

解決方案：
1. 確認專案 composer.json 中的 PHP 版本與擴充模組需求
2. 切換為相容的 PHP 版本執行 Composer
3. 若僅需暫時略過平台檢查，可加上 --ignore-platform-reqs
```

### 8.2 記憶體不足

```text
錯誤訊息：
Fatal error: Allowed memory size of 1610612736 bytes exhausted

解決方案：
1. 解除該次執行的 PHP 記憶體限制：
bin\php\php-8.2.36\php.exe -d memory_limit=-1 bin\composer\composer-2.11.2\composer.phar install

2. 或搭配減少腳本與優先下載壓縮包：
bin\php\php-8.2.36\php.exe -d memory_limit=-1 bin\composer\composer-2.11.2\composer.phar install --no-scripts --prefer-dist
```

### 8.3 網路連線問題

```text
錯誤訊息：
Could not fetch from https://repo.packagist.org, retrying (2 retries left)...

解決方案：
1. 透過環境變數設定代理：
set HTTP_PROXY=http://127.0.0.1:7890
set HTTPS_PROXY=http://127.0.0.1:7890

2. 檢查網路或切換鏡像（如需還原官方來源，可執行 config --unset repos.packagist）：
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar config -g repo.packagist composer https://mirrors.cloud.tencent.com/composer/
```

### 8.4 權限或快取鎖定問題

```text
錯誤訊息：
Could not delete /path/to/vendor/...

解決方案：
1. 確認沒有其他終端或防毒軟體正在鎖定 vendor 目錄
2. 清除 vendor 目錄後重新執行 install：
rd /s /q www\my-app\vendor
bin\php\php-8.2.36\php.exe bin\composer\composer-2.11.2\composer.phar install
```

---

## 9. 速查表

### 9.1 常用場景速查

| 場景 | 建議 PHP 版本 | Composer 版本 | 指令 |
|------|---------------|---------------|------|
| Laravel 12 新專案 | PHP 8.2 ～ 8.5 | Composer 2.11.x | `create-project laravel/laravel "^12.0"` |
| Laravel 11 專案 | PHP 8.2 ～ 8.4 | Composer 2.11.x | `install` |
| 正式環境部署 | 依專案設定 | Composer 2.11.x | `install --no-dev --optimize-autoloader` |
| 舊專案維護 | PHP 7.3 / 7.4 | Composer 2.11.x | `install --no-dev` |

### 9.2 常用選項速查

| 選項 | 說明 |
|------|------|
| `--no-dev` | 不安裝開發依賴（`require-dev`） |
| `--optimize-autoloader` / `-o` | 轉換 PSR-0/4 自動載入為效能更佳的 classmap |
| `--prefer-dist` | 優先下載 dist 壓縮包 |
| `--prefer-source` | 優先使用 source（git） |
| `--ignore-platform-reqs` | 忽略 PHP 版本與擴充模組平台檢查 |
| `--no-scripts` | 不執行 `composer.json` 內的 scripts |
| `--no-interaction` / `-n` | 停用互動式提問 |
| `--dry-run` | 僅模擬執行，不實際安裝或修改檔案 |
| `-W` / `--with-all-dependencies` | 更新指定套件時一併允許更新其相關依賴 |
