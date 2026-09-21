# WinCMP v2.1.4

Release date: 2026-09-22

This release introduces new features, updates, and fixes to WinCMP.

## What's Changed

### Dependencies
- **PHP 7.4 and Redis extension**: Added support for PHP 7.4.33 (NTS x64) and PECL Redis 5.3.7 extension, automatically mapped to Laravel 6.x–8.x project recommendations.

### Security
- **Download and update defense**: Enforced official domain whitelisting, mandatory SHA-256 hash checks, and PE format validation for updater and dependencies, with automatic rollback and diagnostic links on failure.

### Added
- **Reveal log in File Explorer**: Added a "Reveal in File Explorer" context menu action in terminal logs to highlight today's log file.
- **Terminal log line count tooltip**: Hovering over category tabs displays the full service name and current buffered line count (e.g., `Caddy - 4 lines total`).
- **Automatic dependency checks & download links**: Automatically checks for updates upon entering the Dependency Manager, and added an action to copy official download links for manual troubleshooting.

### Changed
- **Database explorer layout refinements**: Restructured MariaDB/Redis switchers, toolbar actions, and search placements, standardizing header height to 48px to eliminate layout jitter.
- **Settings descriptions enhanced**: Streamlined toggle titles and added comprehensive description hints under each setting option.

### Fixed
- **Dashboard navigation flicker**: Eliminated button and status flickering during view transitions using module caching and adaptive skeletons.
- **Windows file locking & process warnings**: Added retry logic for directory renames to overcome transient file lock errors on Windows, and suppressed false-positive termination warnings when stopping runtime services.
- **Missing Caddy snippets on fresh install**: Fixed an issue where standalone executables running for the first time did not automatically extract embedded snippets, causing Caddy startup errors.
- **Terminal log & context menu display**: Eliminated console window flicker when opening log files, and resolved background transparency bleed-through in the Carbon dark theme.

## Getting Started
1. Download `wincmp-v2.1.4-win-x64.zip`.
2. Extract it to any folder on your system.
3. Double-click `WinCMP_v2.1.4.exe` to launch the control panel.
