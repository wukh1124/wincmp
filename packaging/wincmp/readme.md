# WinCMP 🚀

![Go Version](https://img.shields.io/badge/Go-1.26.2+-00ADD8?style=for-the-badge&logo=go)
![Wails Version](https://img.shields.io/badge/Wails-v2.12.0-red?style=for-the-badge&logo=wails)
![React Version](https://img.shields.io/badge/React-v18-blue?style=for-the-badge&logo=react)
![Platform](https://img.shields.io/badge/Platform-Windows_11-0078D6?style=for-the-badge&logo=windows)
![License](https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge)

**WinCMP** is a modern, portable local development environment control panel designed specifically for Windows.
The name stands for **Win**dows + **C**addy + **M**ariaDB + **P**HP.

Inspired by XAMPP and Laragon, WinCMP provides a lightweight, **portable (installation-free)**, and **no-administrator-privileges-required** local development solution. Built with a Go core and the Wails v2 framework, the frontend uses React 18 to deliver an elegant UI, low resource footprint, and lightning-fast startup speed.

---

## ✨ Key Features

- 🪶 **Lightweight**: Compiled with a Go core using the native Web rendering engine (WebView2). Zero Electron dependencies, fast startup, and minimal resource usage.
- 🛡️ **No Admin Privileges Needed**: Runs without system administrator privileges, does not modify system environment variables, and does not write to the registry. *(Note: Automatic host file updates require UAC elevation if enabled).*
- 🎨 **Modern UI/UX**: Built-in dark and light modes with an intuitive graphical interface for real-time service status monitoring.
- 🔄 **PHP Multi-Version Support**: Manage multiple PHP versions simultaneously, with automatic load balancing for improved performance.
- 🚀 **Runtime Multi-Environment Support**: Support for Node.js, Bun, Python, Go (Air/Run), and Custom runtimes.
- 📂 **Project Management**: Visually manage Laravel, Next.js, Nuxt, Astro, Vite, Python, Go, and other projects, featuring automatic framework detection and Caddyfile generation.
- 📜 **Fully Portable**: The entire environment is self-contained in a single folder, ready to be run from a USB drive.

---

## 📥 Quick Start

### System Requirements

- **OS**: Windows 10 / Windows 11 (64-bit)
- **Disk Space**: At least 500 MB available space
- **RAM**: 4 GB or more recommended

### Installation Steps

1. **Download WinCMP**
   - Download the latest `wincmp.zip` (Light version, contains the app binary only) from the Releases page.
   - Extract it to your desired location (e.g., `D:\wincmp`).

2. **Launch & Dependency Detection**
   - Double-click `wincmp.exe` to run the application.
   - **Dependency Missing Prompt**: Upon startup, WinCMP scans the `bin/` directory. If any critical dependencies (Caddy, PHP, MariaDB, etc.) are missing, a prompt will appear.
   - You can click **"Auto Download (Recommended)"** to automatically download and configure the recommended versions from official mirrors, or configure them manually.

3. **Manual Setup (Optional / Custom Versions)**
   If you wish to use your own binary versions, download them and place them in the following directory layout:
   ```
   wincmp/
   ├── bin/
   │   ├── caddy/          # Caddy binary (place caddy.exe here)
   │   ├── mariadb/        # MariaDB binaries (place mariadbd.exe directory here)
   │   ├── php/            # PHP runtimes (supports multiple versions, e.g., php-8.3.28-nts-Win32...)
   │   │   ├── php-8.2/
   │   │   └── php-8.3/
   │   ├── node/           # Node.js binary (optional)
   │   ├── bun/            # Bun binary (optional)
   │   ├── composer/       # Composer binary (optional)
   │   ├── heidisql/       # HeidiSQL binary (optional)
   │   └── mailpit/        # Mailpit binary (optional)
   ```

---

## 📁 Directory Structure

```text
wincmp/
├── wincmp.exe               # Main Executable
├── conf/                    # Configurations
│   ├── ssl/                 # SSL Certificates
│   ├── snippets/            # Shared Caddy snippets
│   ├── sites/               # Project Caddy files
│   ├── wincmp.json          # Main settings JSON
│   ├── Caddyfile            # Master Caddyfile
│   └── my.ini               # MariaDB Configuration
├── bin/                     # Service Binaries
│   ├── caddy/
│   ├── mariadb/
│   ├── php/
│   ├── node/                # Node.js (optional)
│   ├── bun/                 # Bun (optional)
│   ├── composer/            # Composer (optional)
│   ├── heidisql/            # HeidiSQL (optional)
│   └── mailpit/             # Mailpit (optional)
├── data/                    # Database Data Files
│   └── mariadb/
├── logs/                    # Process Logs
└── www/                     # Default Web Root Directory
```

**Note**: Do not delete `conf/`, `data/`, and `logs/` directories as they store your settings and database data.

---

## 🚀 Usage Guide

### Starting Services

1. Open the WinCMP application.
2. Select the services you want to start on the Dashboard:
   - **Caddy**: Web Server (Default ports: 80/443)
   - **MariaDB**: Database Server (Default port: 3306)
   - **PHP**: Choose the version to run
   - **Mailpit**: SMTP testing server (Default ports: 8025/1025, optional)
3. Click "Start" to run the services. Statuses will update in real time.

### Creating a New Project

1. Click "Add Project" in the WinCMP panel.
2. Set the Project Name and Root Path.
3. Select the Project Type (Laravel, Next.js, Nuxt, Astro, Vite, Python, Go API, PocketBase, Custom, etc.).
4. Set the domain name (optional, defaults to `local-{project-name}.test`).
5. Click "Create". WinCMP will automatically detect the framework and generate configurations.

### Accessing Your Site

- **Local Access**: `http://localhost`
- **Domain Access**: Access using the custom domain configured for the project.
- **Database Management**: Connect using HeidiSQL (bundled or external) or other MySQL clients.

### System Tray Features

- Minimize to Tray: Closes the window to the system tray (configurable in settings).
- Right-click the system tray icon to quickly start or stop all services.
- Service uptimes are displayed on the Dashboard.

---

## ⚙️ Advanced Settings

### Modifying Ports

If default ports are occupied, you can modify them in `conf/wincmp.json` or through Settings:
- Caddy HTTP Port: Default 80
- Caddy HTTPS Port: Default 443
- MariaDB Port: Default 3306

### PHP Multi-Version Configuration

Place different versions of PHP in separate subfolders under `bin/php/`:
```
bin/php/
├── php-8.2.30/
├── php-8.3.28/
```
WinCMP automatically scans all available PHP versions and displays only the latest patch version for each minor version.

### Runtime Environments

The Runtime Tab supports running custom script setups:
- **Node.js / Bun**: Drop them into `bin/node/` or `bin/bun/` to let WinCMP scan them automatically.
- **Python / Go**: Uses system PATH installations and automatically detects the environment.
- **Custom**: Define your own command-line script. Supports `%PORT%`, `%HOST%`, `%PROJECT_DIR%`, and `%BIN_DIR%` placeholders.

Runtimes support both **Background** (non-interactive, outputs to logs console) and **Terminal** (spawns an interactive cmd window) execution modes.

### SSL Certificates

By default, WinCMP uses Caddy's auto-generated local CA certificates. To use your own, place them in the `conf/ssl/` directory.

---

## 🛠️ FAQ

**Q: "Executable not found" error during startup?**  
A: Ensure you have placed Caddy, MariaDB, and PHP in their respective directories under `bin/` or clicked "Auto Download".

**Q: Port is already occupied?**  
A: Change the port settings in WinCMP or close other applications (like XAMPP/WAMP/IIS) using that port.

**Q: Database connection failed?**  
A: Ensure the MariaDB service is running. Check `conf/my.ini`. If using an external database, ensure the "Custom" database mode is enabled in Settings.

**Q: Runtime startup failed?**  
A: For Python/Go types, verify they are in your system PATH (`python -V` or `go version` works in terminal). For Node.js/Bun, verify they exist in `bin/node/` or `bin/bun/`.

**Q: How to back up databases?**  
A: The database files are located in `data/mariadb/`. You can copy this folder directly to back them up.

**Q: Can I carry this project to another computer?**  
A: Yes! WinCMP is fully portable. Just copy the entire folder to another PC.

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).

---

## 💡 Links

- **Caddy**: https://caddyserver.com/
- **MariaDB**: https://mariadb.org/
- **PHP**: https://www.php.net/
- **Mailpit**: https://mailpit.axllent.org/
- **Bun**: https://bun.sh/
- **Node.js**: https://nodejs.org/

---

*Thank you for using WinCMP! Please submit an issue if you have any questions.*
