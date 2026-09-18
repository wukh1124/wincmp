# WinCMP v2.1.1
This release introduces new features, updates, and fixes to WinCMP.

## What's Changed

### Fixed
- **Terminal Working Directory Fallback**: Terminal startup no longer fails when a project path is missing or has been moved. WinCMP now resolves a valid working directory and shell absolute path before launching ConPTY.
- **Database Explorer Selection Style**: Selected schema items no longer use a solid accent fill. The list now uses a light surface with accent text; the sketch theme uses an opaque light-blue paper color to keep items readable over grid lines.

### Dependencies
- **Default Dependency Updates**: Updated default dependency versions for Caddy, Composer, HeidiSQL, Mailpit, Node.js, PHP 8.2/8.3/8.4, and php_redis, and completed missing SHA-256 checksums for more reliable download verification.

## Getting Started
1. Download `wincmp-v2.1.1-win-x64.zip`.
2. Extract the archive to any folder on your system.
3. Double-click `WinCMP_v2.1.1.exe` to launch the control panel.
