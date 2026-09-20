# WinCMP v2.1.3

Release date: 2026-09-20

This release introduces new features, updates, and fixes to WinCMP.

## What's Changed

### Added
- **Terminal log context menu**: Right-click the log area to copy the selection, copy a line, select all, or open today's log file for the current category (system/Caddy/MariaDB map to `logs/wincmp-*.log`; Node / Custom maps to `logs/runtime-*.log`). The open action is disabled when no file exists; hover shows the file name and path.
- **Runtime project log menu indicators**: The Node / Custom project selector is now a custom dropdown that shows a dot when another project has new logs.

### Changed
- **Database explorer toolbar**: MariaDB and Redis action buttons are aligned; HeidiSQL is hidden on the Redis tab. Connection address and the DB selector sit next to the tabs, and key search moved into the Keys header to remove the middle info bar.
- **Terminal logs tab label**: `Node / Bun` renamed to `Node / Custom` to match available runtimes.
- **Settings copy**: “Check for updates periodically” is now “Auto-check version updates (GitHub)”, with a note that WinCMP also checks on startup and then every 6 hours.
- **Dependency manager check updates**: After fetching remote recommended versions, WinCMP also rescans the local environment so manually installed extensions (such as PHP Redis) appear immediately.
- **Redis key value panel**: Keys and key-value headers share a fixed height; the copy control is smaller, and long key names/values display more completely.

### Removed
- **Redis flush current DB**: Removed the destructive FLUSHDB UI and backend API.
- **Redis delete key**: Removed single-key delete UI and backend API from the key detail panel.

### Fixed
- **Redis key search**: Input without glob characters is normalized to `*query*`; empty SCAN pages continue paging so matches are not falsely reported as missing.
- **Redis refresh icon when service is down**: Connection probing no longer keeps the refresh icon spinning while Redis is stopped.
- **Version update notices**: With auto-check enabled, WinCMP checks on every app startup and then every 6 hours. Detected updates show a breathing dot on the Version Update sidebar item, and the state is persisted so restarts keep the badge.

## Getting Started
1. Download `wincmp-v2.1.3-win-x64.zip`.
2. Extract it to any folder on your system.
3. Double-click `WinCMP_v2.1.3.exe` to launch the control panel.
