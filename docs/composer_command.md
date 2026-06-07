# WinCMP Composer 使用指南

## 目錄

1. [簡介](#1-簡介)
2. [目錄結構](#2-目錄結構)
3. [快速開始](#3-快速開始)
4. [Composer 版本與 PHP 版本對應](#4-composer-版本與-php-版本對應)
5. [常用指令詳解](#5-常用指令詳解)
6. [Laravel 專案操作實例](#6-laravel-專案操作實例)
7. [依賴管理进阶](#7-依賴管理进阶)
8. [常見問題與解決方案](#8-常見問題與解決方案)

---

## 1. 簡介

Composer 是 PHP 的依賴管理工具，用於管理 Laravel 等 PHP 專案的函式庫依賴。WinCMP 支援在同一系統中同時管理**多個 PHP 版本**與**多個 Composer 版本**，讓你可以針對不同專案需求選擇最適合的組合。

### 1.1 WinCMP 的 Composer 支援

```
WinCMP bin/
├── composer/
│   ├── composer-2.8.6/
│   │   └── composer.phar
│   ├── composer-2.4.24/
│   │   └── composer.phar
│   └── composer-1.10.25/
│       └── composer.phar
├── php/
│   ├── php-8.2.30/
│   │   └── php-cgi.exe
│   ├── php-8.1.32/
│   │   └── php-cgi.exe
│   └── php-7.4.33/
│       └── php-cgi.exe
```

---

## 2. 目錄結構

### 2.1 Composer 版本掃描規則

WinCMP 會自動掃描 `bin/composer/` 目錄，識別格式為 `composer-<版本號>/` 的資料夾：

```
bin/composer/
├── composer-2.8.6/          # ✅ 識別
│   └── composer.phar        # ✅ 識別 (需要 composer.bat 或直接 php composer.phar)
├── composer-2.4.24/        # ✅ 識別
├── composer-1.10.25/        # ✅ 識別
└── composer.phar            # ❌ 不識別 (需放在版本子目錄中)
```

### 2.2 執行原理

WinCMP 使用以下方式執行 Composer：

```cmd
# 方式一：使用 PHP 直接執行 composer.phar
<PHP路徑>\php.exe <Composer路徑>\composer.phar <指令>

# 範例
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar install
```

### 2.3 建議的 Composer + PHP 組合

| Composer 版本 | 建議 PHP 版本 | 說明 |
|--------------|---------------|------|
| Composer 2.8.x | PHP 8.2+ | 最新穩定版，支援 PHP 8.3 |
| Composer 2.4-2.7 | PHP 8.1+ | 長期維護版本 |
| Composer 2.0-2.3 | PHP 7.4+ | 過渡版本 |
| Composer 1.10.x | PHP 7.3-7.4 | 舊專案相容 |

---

## 3. 快速開始

### 3.1 基本語法

```cmd
<PHP路徑>\php.exe <Composer路徑>\composer.phar <指令> [選項]
```

### 3.2 查看版本

```cmd
:: 查看 Composer 版本
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar -V

:: 查看 PHP 版本 (驗證環境)
bin\php\php-8.2.30\php.exe -v
```

### 3.3 建立新專案

```cmd
:: 建立 Laravel 10 專案 (使用 PHP 8.2 + Composer 2.8)
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar create-project laravel/laravel www\my-laravel-app "10.*"

:: 建立 Laravel 11 專案 (使用 PHP 8.2 + Composer 2.8)
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar create-project laravel/laravel www\my-laravel-app "^11.0"

:: 建立 Laravel 9 專案 (使用 PHP 8.1 + Composer 2.4)
bin\php\php-8.1.32\php.exe bin\composer\composer-2.4.24\composer.phar create-project laravel/laravel www\my-laravel-app "9.*"
```

---

## 4. Composer 版本與 PHP 版本對應

### 4.1 版本相容性矩陣

| PHP 版本 | Composer 1.10 | Composer 2.4 | Composer 2.8 |
|----------|---------------|--------------|--------------|
| PHP 8.3  | ❌ | ✅ | ✅ |
| PHP 8.2  | ❌ | ✅ | ✅ |
| PHP 8.1  | ❌ | ✅ | ✅ |
| PHP 8.0  | ❌ | ✅ | ⚠️ 需 2.4+ |
| PHP 7.4  | ✅ | ✅ | ❌ |
| PHP 7.3  | ✅ | ❌ | ❌ |

### 4.2 選擇建議

- **新專案**：使用 PHP 8.2+ 、Composer 2.8
- **Laravel 10**：PHP 8.1+ 、Composer 2.4+
- **Laravel 9**：PHP 8.0+ 、Composer 2.4+
- **Laravel 8**：PHP 7.4+ 、Composer 2.0+
- **舊專案維護**：依據現有 `composer.lock` 的 Composer 版本選擇

---

## 5. 常用指令詳解

### 5.1 安裝依賴

```cmd
:: 基本安裝 (讀取 composer.lock)
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar install

:: 安裝並忽略 composer.lock (升級依賴)
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar update

:: 安裝指定環境的依賴
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar install --no-dev      # 排除開發依賴
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar install --prefer-dist # 優先下載 dist 壓縮包
```

### 5.2 更新依賴

```cmd
:: 更新所有依賴
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar update

:: 更新特定套件
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar update laravel/framework

:: 更新特定套件至指定版本
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar update laravel/framework "^10.0"
```

### 5.3 新增套件

```cmd
:: 新增穩定版套件
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar require guzzlehttp/guzzle

:: 新增開發版套件
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar require --dev phpunit/phpunit

:: 新增特定版本
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar require "laravel/framework:^10.0"
```

### 5.4 移除套件

```cmd
:: 移除套件
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar remove guzzlehttp/guzzle
```

### 5.5 其他常用指令

```cmd
:: 顯示已安裝的套件列表
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar show

:: 顯示特定套件資訊
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar show laravel/framework

:: 檢查安全性問題
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar audit

:: 驗證 composer.json 格式
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar validate

:: 清除快取
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar clear-cache
```

---

## 6. Laravel 專案操作實例

### 6.1 建立新 Laravel 專案

```cmd
:: 進入專案目錄
cd www

:: 使用 PHP 8.2 + Composer 2.8 建立 Laravel 11
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar create-project laravel/laravel my-app "^11.0"
```

### 6.2 安裝現有 Laravel 專案依賴

```cmd
:: 進入 Laravel 專案目錄
cd www\my-laravel-app

:: 使用與專案相容的 PHP + Composer 版本安裝
bin\php\php-8.2.30\php.exe ..\..\bin\composer\composer-2.8.6\composer.phar install
```

### 6.3 更新 Laravel 專案

```cmd
:: 完整更新 (包含框架核心)
cd www\my-laravel-app
bin\php\php-8.2.30\php.exe ..\..\bin\composer\composer-2.8.6\composer.phar update

:: 只更新第三方套件 (保留框架版本)
bin\php\php-8.2.30\php.exe ..\..\bin\composer\composer-2.8.6\composer.phar update --no-interaction
```

### 6.4 Laravel 專屬指令

```cmd
:: 發布設定檔
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar laravel:publish

:: 清除快取
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar laravel:clear-compiled
```

### 6.5 完整工作流程

```cmd
:: 1. 建立新專案
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar create-project laravel/laravel www\demo-app

:: 2. 進入專案目錄
cd www\demo-app

:: 3. 安裝依賴
..\..\bin\php\php-8.2.30\php.exe ..\..\bin\composer\composer-2.8.6\composer.phar install

:: 4. 新增額外套件
..\..\bin\php\php-8.2.30\php.exe ..\..\bin\composer\composer-2.8.6\composer.phar require laravel/sanctum

:: 5. 複製環境設定檔
copy .env.example .env

:: 6. 產生應用程式金鑰
bin\php\php-8.2.30\php.exe artisan key:generate
```

---

## 7. 依賴管理进阶

### 7.1 Composer 脚本

```cmd
:: 查看可用脚本
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar run-script

:: 執行自訂脚本
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar run-script post-autoload-dump
```

### 7.2 離線安裝

```cmd
:: 匯出本機套件到 vendor.bak
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar archive --dir=vendor.bak --file=dependencies

:: 離線安裝 (需要有 vendor.bak)
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar install --no-dev --prefer-dist --ignore-platform-reqs
```

### 7.3 平台需求處理

```cmd
:: 安裝時忽略平台需求 (PHP 擴展等)
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar install --ignore-platform-reqs

:: 更新時忽略平台需求
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar update --ignore-platform-reqs
```

### 7.4 鎖定特定版本

```cmd
:: 將依賴版本鎖定在 composer.lock
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar install --locked

:: 顯示需要更新的套件
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar outdated
```

---

## 8. 常見問題與解決方案

### 8.1 PHP 版本不相容

```
錯誤訊息：
Your requirements could not be resolved to an installable set of packages.

解決方案：
1. 確認專案的 composer.json 中的 PHP 版本要求
2. 選擇相容的 PHP 版本執行 Composer
3. 或使用 --ignore-platform-reqs 忽略平台需求
```

### 8.2 Composer 版本過舊

```
錯誤訊息：
Composer 1.x is not supported. Please update to Composer 2.x.

解決方案：
使用 Composer 2.x 版本
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar self-update
```

### 8.3 記憶體不足

```
錯誤訊息：
Fatal error: Allowed memory size of 1610612736 bytes exhausted

解決方案：
1. 增加 PHP 記憶體限制
bin\php\php-8.2.30\php.exe -d memory_limit=-1 bin\composer\composer-2.8.6\composer.phar install

2. 或使用 Composer 的記憶體優化選項
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar install --no-scripts --prefer-dist
```

### 8.4 網路連線問題

```
錯誤訊息：
Could not fetch from https://repo.packagist.org, retrying (2 retries left)...

解決方案：
1. 使用代理
set HTTP_PROXY=http://127.0.0.1:7890
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar install

2. 或使用中國鏡像
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar config repo.packagist composer https://packagist.org

3. 或配置騰訊雲鏡像
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar config repo.packagist composer https://mirrors.cloud.tencent.com/composer/
```

### 8.5 權限問題

```
錯誤訊息：
Could not delete /path/to/vendor/.cache

解決方案：
1. 以系統管理員身份執行
2. 或手動刪除 vendor 目錄後重新安裝
rd /s /q www\my-app\vendor
bin\php\php-8.2.30\php.exe bin\composer\composer-2.8.6\composer.phar install
```

---

## 9. 速查表

### 9.1 常用組合速查

| 場景 | PHP 版本 | Composer 版本 | 指令 |
|------|----------|---------------|------|
| Laravel 11 新專案 | PHP 8.2 | Composer 2.8 | `create-project laravel/laravel "^11.0"` |
| Laravel 10 新專案 | PHP 8.1 | Composer 2.4 | `create-project laravel/laravel "10.*"` |
| Laravel 9 專案 | PHP 8.0 | Composer 2.4 | `install` |
| 舊專案維護 | PHP 7.4 | Composer 1.10 | `install --no-dev` |

### 9.2 選項速查

| 選項 | 說明 |
|------|------|
| `--no-dev` | 不安裝開發依賴 |
| `--prefer-dist` | 優先下載 dist 壓縮包 |
| `--prefer-source` | 優先使用 source (git) |
| `--ignore-platform-reqs` | 忽略平台需求 |
| `--no-scripts` | 不執行 scripts |
| `--no-interaction` | 不等待互動輸入 |
| `--locked` | 使用已鎖定的版本 |

---

> 💡 **提示**：WinCMP 會自動掃描 `bin/composer/` 目錄中的版本，你也可以手動下載不同版本的 Composer 到該目錄中。
