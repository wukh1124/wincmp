# WinCMP 🚀

![Go Version](https://img.shields.io/badge/Go-1.26.2+-00ADD8?style=for-the-badge&logo=go)
![Wails Version](https://img.shields.io/badge/Wails-v2.12.0-red?style=for-the-badge&logo=wails)
![React Version](https://img.shields.io/badge/React-v18-blue?style=for-the-badge&logo=react)
![Platform](https://img.shields.io/badge/Platform-Windows_11-0078D6?style=for-the-badge&logo=windows)
![License](https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge)

**WinCMP** is a modern, portable local development environment control panel designed specifically for Windows. 
The name is derived from **Win**dows + **C**addy + **M**ariaDB + **P**HP.

Inspired by XAMPP and Laragon, WinCMP aims to provide a more lightweight, **portable (no installation required)**, and **mostly admin-privilege-free** development solution (excluding optional Hosts file modifications). Built with Go core and the Wails v2 framework, it features a premium React 18 frontend with extremely low resource usage, fast startup speeds, and beautiful visual aesthetics.

---

## 📸 Preview

![WinCMP Dashboard](screenshot/dashboard.png)

---

## ✨ Features

- 🪶 **Extremely Lightweight**: Statically compiled in Go + Wails, leveraging the native OS web engine (WebView2) without Electron dependencies.
- 🛡️ **No Admin Privileges Needed for Core Services**: Fully supports running under restricted environments without modifying system environment variables or writing to the registry. *(Note: Automatic writing to the Windows `hosts` file for custom domains is optional and requires Administrator elevation/UAC prompt).*
- 🎨 **Modern UI/UX**: Premium Dark Professional theme with smooth sidebar navigation, real-time status monitoring, and interactive micro-animations.
- 🔄 **PHP Multi-Process Load Balancing**: Leverages Caddy's upstream mechanism to run multiple FastCGI processes for each PHP version.
- 📂 **Automated Project Management**: Visually manage Laravel, Next.js, Nuxt, Astro, Vite, Python, Go, and other projects. Automatically detects frameworks and generates configurations.
- 🚀 **Runtime Multi-Environment Execution**: Supports Node.js, Bun, Python, Go (Air/Run), and Custom development environments, with options to start in Background or Terminal mode.
- 💻 **Project Integrated Interactive Terminal**: Spawns a beautiful drawer-based Terminal (PowerShell, CMD, Git Bash, WSL) at the project root using Windows ConPTY and `xterm.js` with interactive CLI and auto-completion support.
- 📜 **Isolated Environments**: Dynamically injects `PATH` when launching subprocesses to ensure PHP and its extensions run in the correct binary environments.

---

## 📁 Project Architecture & Directory Layout

To achieve "plug-and-play" simplicity, WinCMP strictly adheres to the following directory structure:

```text
wincmp/
├── main.go                  # Application entry point: initializes and starts Wails
├── app.go                   # Wails lifecycle manager (startup, shutdown) and monitor triggers
├── bridge.go                # Wails & Go Binding API (frontend-backend RPC endpoints)
├── downloader_bridge.go     # Wails dependency downloader binding API
├── wincmp.json              # WinCMP global & project configurations (UI data source)
├── conf/                    # Configuration center
│   ├── ssl/                 # SSL Certificates (crt/key)
│   ├── snippets/            # Shared Caddy configuration snippets
│   ├── sites/               # Dynamically generated project Caddyfiles
│   ├── Caddyfile            # Caddy entry point (Imports snippets & sites)
│   └── my.ini               # MariaDB initialization config
├── bin/                     # Binary executables directory (pre-included or auto-downloaded)
│   ├── caddy/               # caddy-x.xx.x/caddy.exe
│   ├── mariadb/             # mariadb-x.x.x/bin/mariadbd.exe
│   ├── php/                 # php-x.x.x/php-cgi.exe
│   ├── node/                # node-x.x.x/npm.cmd
│   ├── bun/                 # bun-x.x.x/bun.exe
│   ├── composer/            # composer-x.x.x/composer.bat
│   ├── heidisql/            # heidisql-x.xx/heidisql.exe
│   └── mailpit/             # mailpit-x.xx.x/mailpit.exe
├── data/                    # Data storage
│   └── mariadb/             # Default MariaDB data directory
├── logs/                    # Service execution logs (grouped by date)
├── www/                     # Default web projects root directory
├── internal/                # Core logic (independent of GUI)
│   ├── config/              # JSON configuration reader/writer
│   ├── scanner/             # Dynamic version scanning for the bin directory
│   ├── process/             # Subprocess lifecycle management (Manager)
│   ├── detect/              # Laravel project detection (confidence-score based)
│   ├── preset/              # Project preset system (framework detection & command templates)
│   ├── hosts/               # Windows Hosts file manager
│   ├── port/                # Port occupation checker
│   ├── resource/            # Resource monitoring (CPU/RAM/Subprocess stack)
│   ├── crypto/              # MariaDB password encryption
│   └── singleinstance/      # Single instance lock + window focus helper
└── frontend/                # Frontend React + TSX project
    ├── src/                 # Frontend source files (Dashboard, Projects, etc.)
    └── tailwind.config.js   # Tailwind style configurations
```

---

## 🛠️ Architecture & Under-the-Hood Logic

### 1. PHP Process Management & Port Mapping
WinCMP utilizes a **3-version-sequence** pattern to assign service ports, ensuring different versions of PHP can run concurrently without conflicts:
- **Naming Convention**: `3<Major><Minor><Sequence 00-99>`
  - PHP 7.3 → `37300`, `37301`, `37302`
  - PHP 8.2 → `38200`, `38201`, `38202`
- **Load Balancing**: Each PHP version starts 3 `php-cgi` processes by default. WinCMP defines `php_fastcgi 127.0.0.1:38200 127.0.0.1:38201 ...` in Caddyfile to balance the requests.

### 2. Dynamic Caddy Configuration
When a user updates project settings in the UI:
1. `conf/wincmp.json` is updated.
2. The Go application rewrites `conf/sites/{project}.caddy`.
3. Calls `caddy reload` for a zero-downtime hot reload.

### 3. Dynamic Environment Variables Injection
To avoid modifying the system's global `PATH`, WinCMP prepends the corresponding binary directories to the subprocess's `Env` list via `os/exec` (e.g., when launching PHP), ensuring that extensions and dependencies locate their correct DLLs.

---

## 🚀 Development & Compilation

### 1. Prerequisites
- [Go 1.26.2+](https://go.dev/dl/)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation/): Make sure Wails v2 is installed on your system. Otherwise install it using `go install github.com/wailsapp/wails/v2/cmd/wails@latest`.
- C Compiler: MinGW-w64 (WinLibs) for compiling underlying native Windows bindings. Make sure `gcc -v` works.
- [Node.js](https://nodejs.org/): Node.js 18+ for compiling frontend components.

### 2. Development Hot Reload
```cmd
# Start Wails dev server (watches both Go backend and React frontend)
wails dev
```

### 3. Build Commands
```cmd
# Tidy Go modules & frontend packages
go mod tidy
cd frontend && npm install && cd ..

# Build with debug console and developer tools
wails build -debug

# Release build (no CMD window console, outputs wincmp.exe)
wails build -clean

# Production build with stripped symbols for size optimization
wails build -clean -ldflags "-s -w"

# Production build with dynamic version injection (e.g., v2.0.0)
wails build -ldflags "-X main.AppVersion=v2.0.0"
```

---

## 🗺️ Roadmap

### ✅ Completed
- [x] Modern UI prototype and project management interface (rebuilt with Wails + React 18).
- [x] Multi-tab system logs with log-rotation mechanism.
- [x] MariaDB scanning and database viewer.
- [x] Multi-process PHP load balancing for Caddy.
- [x] **Windows System Tray** minimization support.
- [x] **Resume Last Services**: Auto-starts services that were running when the app was closed (saved in `wincmp.json`).
- [x] **Service Uptime Tracker** (independent stats for Caddy, MariaDB, and PHP).
- [x] **Laravel Auto-Detection** (confidence score system, automatically routing to `public/`).
- [x] **Port Conflict Check** (runs checks before launch to minimize race conditions).
- [x] **Hosts File Auto-Management** (automatically syncs local domains after prompting for UAC elevation).
- [x] **Dark/Light Theme Toggle** (integrated with Tailwind CSS).
- [x] **Runtime Multi-Environment Support** (Node.js, Bun, Python, Go Air/Run, Custom).
- [x] **Framework Preset Auto-Detection** (Next.js, Nuxt, Astro, Vite, Django, FastAPI, Flask, PocketBase, Go API).
- [x] **Double Launch Mode for Runtimes** (Background / Terminal).
- [x] **Legacy Project Auto-Migration** (node_port → runtime_port, etc.).
- [x] **Mailpit Integration** (start/stop toggle on Dashboard and configuration dialog).
- [x] **Project Integrated Interactive Terminal** (integrated Windows ConPTY with `xterm.js` to support interactive CLI, auto-completion, and customization in Settings).

### ⏳ Planned
> **💡 For a detailed roadmap, technical analysis, and prioritization, please refer to the complete [Develop Task List](doc/develop_task_list.md).**

- **Deep OS Integration**: Windows auto-start on boot (`HKCU\Run`), one-click setup for Windows system environment path (`Path`).
- **Dev Toolchain**: Embedded Composer support (no `composer.phar` installation needed), PHP process watchdog with auto-recovery.
- **Advanced Service Manager**: HeidiSQL integration (preview & fast connection), automated downloader for service executables (Caddy/PHP/MariaDB multi-version).

---

## 📄 License

This project is licensed under the [MIT License](https://opensource.org/license/mit/).
Feel free to submit Pull Requests or open Issues to share your feedback!