# Changelog

## [2.0.4] 2026-06-19

### Added
- **Monorepo Project Support**: Added a Monorepo helper checkbox during project creation to automatically adapt default project names and domain aliases.
- **Quick Settings Onboarding Guide**: Added step-by-step onboarding guide bubbles for sidebar Quick Settings (Theme/Language/Font).
- **Custom Start Command Configuration**: Refactored project properties to introduce a "Use Custom Command" option, dynamically displaying read-only default commands as a reference.
- **Sidebar Lock/Unlock**: Added a lock/unlock button to prevent the sidebar from collapsing automatically when switching pages.

### Changed
- **Auto-Update Check Optimization**: Optimized background auto-update check intervals and cached status to improve startup performance and bandwidth efficiency.
- **Onboarding Bubble Flow & Visuals**: Changed onboarding guide bubbles to trigger sequentially to prevent overlap, and enhanced readability contrast and layout stability across all themes.
- **Micro-interactions for Buttons & UI**: Added a subtle border color change effect when hovering over all major action buttons and optimized the hover style for danger buttons.
- **Theme Renaming**: Renamed the "Claude" theme to "Cream".
- **Enhanced Update Notification Visuals**: Added a pulsing red dot animation to the sidebar update badge for clearer visual notification.

### Fixed
- **Project List Header Penetration**: Fixed an issue where the header was penetrated by scrolling content in the project list, keeping it sticky and opaque across all themes.
- **Custom Framework Start Command Unlock**: Fixed a bug where the start command input field for the Custom Framework could not be edited.
- **Status Indicator Dot Alignment**: Fixed layout shifting of status indicator dots caused by font scaling, locking them to a fixed 8px size.

## [2.0.3] 2026-06-15

### Added
- **SHA-256 Dependency Integrity Verification**: Implemented automatic SHA-256 integrity verification after downloading and before extracting dependencies. Corrupted files are automatically deleted, with clear user instructions provided for manual resolution or remote configuration fetching.
- **Onboarding Guide**: Added interactive step-by-step onboarding popovers for project actions and the dependency manager to help users quickly get familiar with the UI.
- **Custom Hosts Error Alerts**: Replaced native browser dialogs with a custom React Alert component for Hosts file write failures, adding a "Do not remind again" option to prevent repetitive alerts.

### Changed
- **Hosts Sync Optimization**: Optimized hosts file automatic update logic to only sync hosts for enabled projects.
- **Simplified Dependency Check**: Streamlined startup dependency checks to scan for Caddy only.
- **Theme Improvements & Renaming**: Renamed the "xAI" dark theme to "Carbon"; optimized the "Sketch" hand-drawn theme UI to resolve text legibility issues in alert banners and project lists against the graph grid background.

## [2.0.2] 2026-06-12

### Added
- **Theme Support**: Integrated dark/light/system theme switching.
- **Process Guard & Port Cleanup**: Introduced Windows Job Object to terminate entire runtime process trees instantly.
- **Custom Command Safety**: Restricted custom executable/script executions to the active project root directory.
- **Global Environment Auto-Detection**: Added automatic sensing of global Node.js/Bun installations.
- **Single-Instance Protection**: Added process check to prevent concurrent runs with legacy v1 version.

## [2.0.1] 2026-06-08

### Added
- **Automatic SSL CA Cert Configuration**: Added automatic downloading and configuration of `cacert.pem` for PHP SSL requests when it is missing in the local environment.
- **Automatic Version Checker & One-Click Updater**: Added background checks for new releases on GitHub every 6 hours, a dedicated "Version Update" tab in the sidebar with a red notification badge, and seamless self-rename/restart/cleanup logic for passwordless exe update on Windows.
- **Release Automation Optimization**: Updated `release.ps1` to export a standalone `WinCMP_v*.exe` alongside the release zip for faster direct updates.

### Changed
- **Dependency Optimization**: Completely removed the deprecated `fyne.io/fyne/v2` dependency and legacy resource monitoring GUI code, reducing final binary size.
- **Wails Build Improvements**: Restored Wails build templates (such as `icon.ico` and manifest files) into the Git repository to resolve build failure caused by incorrect `.gitignore` patterns.

## [2.0.0] 2026-06-07

### Added
- **Full GUI Core Refactoring**: Migrated to **Wails v2** + **React 18** + **TypeScript** for better performance and lower resource usage.
- **Dark Professional Design**: Premium high-density dark UI built with Tailwind CSS and reactive Zustand stores.
- **Improved Tools**: Added high-performance real-time log console, TanStack Table projects manager, and TablePlus-like database viewer.

### Removed
- **Removed Legacy GUI**: Completely removed the old Go Fyne implementation (archived to the `legacy_fyne/` directory).

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