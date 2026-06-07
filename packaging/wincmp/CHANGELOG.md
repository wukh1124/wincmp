# Changelog

## [2.0.0] 2026-06-03

### Added
- **Full GUI Core Refactoring**: Ported the desktop application framework from Go Fyne to **Wails v2** + **React 18** + **TypeScript**.
- **Brand New Design System**: Implemented a premium, high-density **Dark Professional** theme using Tailwind CSS and curated HSL color mappings.
- **High-Performance Log Console**: Integrated instant, high-performance logs renderer supporting real-time stdout/stderr streams from Caddy, MariaDB, PHP, Mailpit, and Runtime.
- **Improved Projects & DB Explorer**: Rebuilt user interface using TanStack Table with smooth slide-over drawers for configuration editing and TablePlus-like database tables viewer.
- **Zustand State Management**: Established clean and reactive frontend stores for service, projects, database, settings, and terminal states.

### Changed
- Archived legacy code: Moved previous Go Fyne implementation code to the `legacy_fyne/` directory.
- Removed obsolete files: Cleaned up unused Go files including `ui_runtime.go` and `bundled_icon.go`.

## [1.2.6] 2026-06-03

### Added
- Added "Display Language" setting under "Settings" supporting switching between Traditional Chinese (`zh-TW`) and English (`en-US`).
- Added "Auto Restart WinCMP" button in the prompt dialog after changing display language.

### Changed
- Adjusted dependency configurations in `conf/dependencies.json` to only allow launching the latest Caddy and MariaDB versions.

## [1.2.5] 2026-06-02

### Added
- Added automatic downloading and extraction of core dependencies (supporting Caddy, MariaDB, PHP 7.3/8.2/8.3, Composer, HeidiSQL, Node.js, etc.) with a download progress UI.
- Added dependency integrity checking and a warning dialog upon application startup.
- Added `conf/dependencies.json` configuration file to externalize dependency versions and download URLs from code.
- Added "Fetch Latest Recommended Versions" (Fetch) feature to dynamically update dependency configurations from a remote GitHub repository.
- Added a fallback dialog when system Hosts update fails, offering one-click copying of Hosts rules and opening the Hosts file in Notepad with Administrator (UAC) privileges.

### Changed
- Optimized the Dependency Manager UI layout by coloring "Download" and "Reinstall" buttons differently and adjusting vertical spacing.

### Fixed
- Fixed directory naming format after automatically downloading MariaDB and Node.js, and auto-generated `composer.bat`.

## [1.2.4] - 2026-04-20

### Fixed
- Fixed the issue where "Open Project Directory" under Edit Project and "hosts" under Settings could not be opened.

## [1.2.3] - 2026-04-16

### Fixed
- Fixed Caddy configuration fallback to incorrect domains when domains contain underscores (Caddyfile now uses the user's input directly without safe-filtering fallback).
- Enhanced error messaging for Hosts update failures, explicitly listing domains with invalid characters and prompting users to add them to hosts manually.

## [1.2.2] - 2026-04-16

### Fixed
- Fixed Windows Hosts file writing issue.
- Fixed Terminal Logs tab index mismatch (incorrect mapping for Mailpit/PHP/Runtime tabs).
- Fixed the issue where Terminal Logs automatically switched to the Runtime tab on application startup (added an initialization lock mechanism).
- Fixed the issue where the tab switch was ineffective when new content arrived in Runtime Log (now only triggers when conditional checks pass).
- Fixed the issue where log content did not automatically scroll to the bottom after switching tabs (moved scrolling execution to happen after tab switching).

### Changed
- Optimized Terminal Logs tab auto-scrolling: scrolls to the bottom on the target tab during tab switching.

## [1.2.1] - 2026-04-16

### Added
- Added Mailpit email testing service integration (added Mailpit service control buttons and settings dialog in Dashboard).
- Added a Mailpit tab to Terminal Logs.
- Added system PATH fallback for Runtime (automatically detects Node.js/Bun in the system PATH when execution files are missing in `bin/`).

### Changed
- Upgraded Go version to 1.26.2.
- Reordered Terminal Logs tabs to: System / Caddy / MariaDB / Mailpit / PHP / Runtime.

### Fixed
- Fixed Entry component blocking wheel scroll events on parent VScroll.
- Fixed abnormal filename when project name contains special characters (special characters are now automatically replaced with hyphens).
- Fixed expired logs remaining undeleted after Caddy Timberjack stops on Windows.
- Fixed non-Custom Runtimes failing to clear startup commands and leftover MariaDB status tags.
- Fixed path backslashes in Windows being falsely flagged as shell injection characters when UseWinCMPBin=false.

### Dependencies
- Mailpit 1.29.6

## [1.2.0] - 2026-04-13

### Added
- Added Runtime development environment execution (expanded support from Node.js only to Bun, Python, Go, and Custom).
- Added a project log filter button to the Runtime tab to quickly switch to the corresponding project's Terminal Logs.
- Added a one-click copy link button next to the Domain field.
- Added automatic project type detection (Preset system), supporting Laravel, Next.js, Nuxt, Astro, Vite, Python (Django/FastAPI/Flask), PocketBase, Go API, etc.
- Automatically migrated legacy Node.js projects to the new Runtime architecture.
- Added tooltip text to System Tray icon.

### Changed
- Rename Node.js to Runtime, Node.js Port to Runtime Port, and Node.js Projects to Projects Runtime.
- Rename Node Version to Runtime, with options now including Auto, Node.js, Bun, Python, Go Air, Go Run, and Custom.
- Switched to using RSS (WorkingSetSize) to show RAM usage of WinCMP.
- Hovering over the Monitor area in the bottom info bar now displays a custom tooltip showing Stack Total and detailed service information.
- Supported configuring MariaDB to use an external MariaDB/MySQL server, custom path, and custom port.
- Supported Background/Terminal mode selection for Runtime execution.
- Optimized Terminal Logs log limitation (500 lines or 200KB characters).
- Optimized page transitions and rapid tab switching performance (added debounce mechanism).

### Fixed
- Fixed page lag and performance bottlenecks (reduced OS Stat calls in Projects, lazy loaded DB Explorer and Node.js components).
- Fixed Settings MaxLogRetention to automatically delete expired log files.
- Fixed insufficient text contrast for logs under dark mode in Terminal Logs.

### Security
- For detailed audit report, see `doc/audit_report_v1.2.0.md`.

### Dependencies
- Bun 1.3.11

---

## [1.1.3] - 2026-04-09

### Fixed
- Added tooltip text to System Tray icon.

---

## [1.1.2] - 2026-04-02

### Changed
- Switched to using RSS (WorkingSetSize) to show RAM usage of WinCMP, reflecting actual physical memory usage (differences from Windows Task Manager may still exist).
- Hovering over the Monitor area in the bottom info bar now displays a custom tooltip showing Stack Total and detailed service information (e.g., Caddy, MariaDB, PHP-CGI, Node.js).

---

## [1.1.1] - 2026-03-30

### Added
- Added Monitor to bottom status bar, displaying CPU and RAM usage of WinCMP.

### Changed
- Added MariaDB settings to support external MariaDB/MySQL, custom path, and custom port.

### Fixed
- Added Terminal Logs log limitation (500 lines or 200KB characters).
- Fixed page lag and performance issues (reduced OS Stat calls in Projects using precalculated functions, lazy loaded DB Explorer and Node.js, removed unnecessary delays, and ignored rapid consecutive tab clicks until the current tab finishes loading).
- Fixed Settings MaxLogRetention to automatically delete expired `error-*.log`, `node-*.log`, and `wincmp-*.log` files.

---

## [1.1.0] - 2026-03-26

### Added
- Supported startup/reverse proxy for Node.js projects.
- Added button to open log file in Terminal Logs.

### Changed
- Optimized PHP version prompts when starting Caddy.
- Improved PHP version detection for Laravel projects.
- Improved Node project detection.
- Improved MariaDB initialization prompts.
- Renamed setting `auto_start` to `restore_last_state` in `wincmp.json`.
- Switched terminal log text to light gray in dark mode.

### Dependencies
- Composer 1.10.10 / 2.9.3
- Node 24.14.1

---

## [1.0.0] - 2026-03-23

### Added
- **WinCMP** portable Windows development panel core framework.
- One-click startup/stop and hot-reload support for Caddy server.
- MariaDB database management interface (connection testing, backup).
- PHP multi-version load balancing (7.3/8.2/8.3).
- Fast project creation and environment isolation tools.

### Dependencies
- Caddy 2.11.1
- Heidisql 12.16
- MariaDB 11.4.10
- PHP 7.3.33 / 8.2.30 / 8.3.28