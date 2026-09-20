# WinCMP v2.1.2

Release date: 2026-09-20

This release introduces new features, updates, and fixes to WinCMP.

## What's Changed

### Added
- **Built-in Redis Cache Explorer**: The database explorer now includes a Redis tab. You can switch across DB 0–15, page through keys with safe SCAN-based search, inspect key name/type/TTL/value, delete a single key, or flush the current database after confirmation. If Redis is not running, a guidance card takes you straight to the dashboard to start the service.
- **Automatic PHP Redis Extension Setup**: On startup WinCMP checks `php.ini`. If `extension=redis` is missing it backs up the file and appends the setting safely; if the line is commented out it is uncommented automatically. PHP cards on the dashboard show a Redis status badge, with one-click enable or a link to dependency management when needed.
- **Custom Redis Port**: Settings gains a Redis port option (default 6379) used for service start/stop and the built-in cache explorer connection.
- **Simplified Sidebar Collapse**: Removed the lock button and auto-collapse on tab switches. Collapse state is stored locally and kept across app reloads.

### Changed
- **Dashboard PHP Card Layout**: Restructured PHP FastCGI cards so version, port range, run status, and the Redis badge are easier to scan.
- **Dependency Manager Redis Status**: php_redis is shown as ready only when the DLL is installed and effectively enabled in `php.ini`, reducing false positives.

### Fixed
- **Runtime Start Command Safety**: Custom start commands that begin with `start` or detach via `cmd start` / `Start-Process` are rejected with guidance to use native foreground commands, keeping processes inside WinCMP lifecycle management.
- **Port Release Guarantees**: After killing a process that holds a port, WinCMP waits for the TCP port to fully free (up to about 3 seconds), reducing `EADDRINUSE` on project restart and accidental frontend port jumps. Startup also pre-checks whether the port is still occupied.

### Dependencies
- **Go Redis Client**: Added the official `go-redis/v9` module for the built-in Redis cache explorer.

## Getting Started
1. Download `wincmp-v2.1.2-win-x64.zip`.
2. Extract the archive to any folder on your system.
3. Double-click `WinCMP_v2.1.2.exe` to launch the control panel.
