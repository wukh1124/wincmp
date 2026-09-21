# WinCMP v2.1.4

Release date: 2026-09-21

This release introduces new features, updates, and fixes to WinCMP.

## What's Changed

### Dependencies
- **PHP 7.4 and Redis extension**: Added support for PHP 7.4.33 (NTS x64) and PECL Redis 5.3.7 extension, automatically mapped to Laravel 6.x–8.x project recommendations.

### Added
- **Reveal log in File Explorer**: Added a "Reveal in File Explorer" context menu action in terminal logs to open the folder containing today's log file with the file highlighted.
- **Terminal log line count tooltip**: Hovering over category tabs now displays the full service name and current buffered line count (e.g., `Caddy - 4 lines total`) while keeping the tab labels clean.
- **Automatic dependency update check**: Entering the Dependency Manager automatically triggers a silent background update check with cooldown throttling to avoid redundant network requests.

### Changed
- **Database explorer layout refinements**:
  - Moved the MariaDB and Redis switcher to the top title bar aligned to the right, showing connection address and status via tooltip on hover.
  - Relocated MariaDB "Refresh" and "Open in HeidiSQL" buttons directly to the database and table list header rows.
  - Relocated Redis key search to the top of the Keys list, "Refresh" button to the Keys header, and DB selector to the left of the key detail header.
  - Standardized header row heights across both MariaDB and Redis panels to 48px to eliminate layout jitter during tab switching.
- **Settings descriptions enhanced**: Streamlined toggle titles and added comprehensive description hints under every setting option for a clean, cohesive layout.

### Fixed
- **Missing Caddy snippets on fresh install**: Fixed an issue where standalone executables running for the first time did not automatically extract embedded `snippets/common.caddy` and `snippets/php-upstream.caddy`, causing Caddy startup errors.
- **Terminal log open console flicker**: Eliminated console window flicker when opening log files with the default editor, and resolved editor launch failures on specific system paths.
- **Carbon theme context menu transparency**: Resolved background transparency issues in the Carbon dark theme where context menus and dropdowns suffered from readability and text bleed-through, introducing solid background surfaces with drop shadows.

## Getting Started
1. Download `wincmp-v2.1.4-win-x64.zip`.
2. Extract it to any folder on your system.
3. Double-click `WinCMP_v2.1.4.exe` to launch the control panel.
