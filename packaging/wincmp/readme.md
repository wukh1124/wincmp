# WinCMP

![Platform](https://img.shields.io/badge/Platform-Windows%2010%2F11-0078D6?style=for-the-badge&logo=windows)
![License](https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge)

Portable local development control panel for Windows: **Caddy + MariaDB + PHP + Mailpit + Redis**, plus project runtimes and a built-in terminal. No installer; core services run without admin rights.

Source & issues: [github.com/wukh1124/wincmp](https://github.com/wukh1124/wincmp)

---

## Features

- Start/stop Caddy, MariaDB, PHP-CGI, Mailpit, Redis from one panel
- Multi-version PHP with load balancing (default 3 FastCGI processes, adjustable)
- Project presets (Laravel, Next.js, Nuxt, Astro, Vite, Django/FastAPI/Flask, …)
- Runtimes: Node.js / Bun / Python / Go / Custom (Background or Terminal)
- Built-in terminal drawer + DB Explorer (HeidiSQL)
- One-click dependency downloader into `/bin`
- Fully portable — move `conf/` + `bin/` + `data/` to another PC

---

## Quick start

### Requirements

- Windows 10 / 11 (64-bit)
- ~500MB free (1GB+ recommended if you install several services/versions)
- 4GB+ RAM recommended

### Install

1. Download the latest **`wincmp-v*-win-x64.zip`** from [Releases](https://github.com/wukh1124/wincmp/releases/latest).
2. Extract anywhere (example: `D:\wincmp`).
3. Run **`WinCMP_vX.Y.Z.exe`**.
4. If core binaries are missing, use the in-app **dependency downloader** (recommended), or place them under `bin/` yourself.

### Manual `bin/` layout (optional)

```text
wincmp/bin/
├── caddy/          # caddy.exe
├── mariadb/        # mariadbd.exe in versioned folder
├── php/            # php-x.x.x-nts-.../php-cgi.exe (multi-version OK)
├── mailpit/
├── redis/
├── node/  bun/  composer/  heidisql/   # optional
```

Do not delete `conf/`, `data/`, or `logs/` — they hold config and MariaDB data.

---

## Usage

### Services

Dashboard services: **Caddy** (80/443), **MariaDB** (3306), **PHP** (version picker), **Mailpit** (1025 / 8025), **Redis** (6379).

### New project

1. **Add Project**
2. Set name + root path
3. Pick project type / preset
4. Optional domain (default `local-{name}.test`)
5. **Create** — WinCMP generates Caddy config for you

### Access

- Local: `http://localhost` or your project domain
- DB: panel **Database** page, or **Open in HeidiSQL**

### Tray

Minimize to tray (optional in Settings). Tray menu can start/stop services quickly.

---

## Settings notes

- **Ports** — change in Settings UI or `conf/wincmp.json` (Caddy 80/443, MariaDB 3306).
- **PHP versions** — one folder per version under `bin/php/` (e.g. `php-8.3.33-nts-...`).
- **Runtimes** — Node/Bun scanned from `bin/`; Python/Go from system PATH; Custom commands support `%PORT%`, `%HOST%`, `%PROJECT_DIR%`, `%BIN_DIR%`.
- **SSL** — Caddy local CA by default; put custom certs in `conf/ssl/`.

---

## FAQ

**Executable not found**  
Install Caddy / MariaDB / PHP via the dependency downloader, or place binaries under `bin/`.

**Port already in use**  
Change the port in Settings, or stop the other app (XAMPP/WAMP/IIS, etc.).

**Database connection failed**  
Start MariaDB; check `conf/my.ini`. External MySQL/MariaDB can be used from Settings if needed.

**Runtime failed to start**  
Python/Go must work in PATH (`python -V`, `go version`). Node/Bun must exist under `bin/node/` or `bin/bun/`.

**Backup databases**  
Copy `data/mariadb/`.

**Move to another PC**  
Copy the whole folder, or at least `conf/` + `bin/` + `data/`.

---

## Links

- [Caddy](https://caddyserver.com/)
- [MariaDB](https://mariadb.org/)
- [PHP](https://www.php.net/)
- [Mailpit](https://mailpit.axllent.org/)
- [Redis](https://redis.io/)
- [Node.js](https://nodejs.org/) / [Bun](https://bun.sh/)

## License & Trademark Disclaimer

- **Core Application**: [MIT License](LICENSE)
- **Third-Party Notices**: See [THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md) for licenses and notices of integrated and managed components.
- **Trademarks**: WinCMP is an independent open-source project. All product names, logos, and brands (such as Caddy, MariaDB, Redis, PHP, Node.js, Composer, HeidiSQL, Mailpit) are property of their respective owners. WinCMP is not affiliated with, endorsed by, or sponsored by them.

