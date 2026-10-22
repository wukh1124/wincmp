# WinCMP v2.1.7

Release date: 2026-10-22

This is a security release: the bundled PHP runtimes now carry the fixes for CVE-2026-109618 and CVE-2026-109619, plus PHP 8.5 support, PHP lifecycle badges and cleanup of superseded runtimes.

## What's Changed

### Security
- **Critical PHP vulnerability fixes (CVE-2026-109618 / CVE-2026-109619)**: Bundled PHP is updated to 8.5.12, 8.4.27, 8.3.36 and 8.2.35. Both flaws allow unauthenticated remote code execution (CVSS v4.0 9.2) and upstream urges every user to update immediately. Existing installs should open the dependency manager, click "Check Updates" and reinstall PHP, or install the new version directly.

### Added
- **PHP 8.5 runtime**: Added PHP 8.5.12 (NTS, VS17 x64) together with the matching php_redis 6.3.0 extension, downloadable from the dependency manager and configured into `ext/` automatically.
- **PHP lifecycle badges**: Service cards and the dependency manager now mark branches that reached end of life (EOL) or end support within 180 days; hover shows the end-of-support date and remaining days. PHP 7.3 / 7.4 no longer receive Windows security fixes upstream, so consider upgrading.
- **Superseded runtime detection and cleanup**: After a patch upgrade, directories left behind in `bin/php/` are listed with a "Clean up" action. Deletion shows the full path, asks for confirmation, and only targets directories replaced by a newer patch of the same branch.

### Dependencies
- **Default dependency updates**: Caddy 2.11.7, Node.js 24.22.1 (flagged as a security release upstream), MariaDB 11.4.13 and Mailpit 1.31.6; Composer 2.10.3, HeidiSQL 12.21 and Redis 5.0.14.1 are already current.
- **CA bundle pinned to an immutable snapshot**: Switched to `cacert-2026-10-20.pem`. The previously pinned rolling `cacert.pem` broke download verification whenever upstream refreshed the Mozilla store.

### Fixed
- **PHP same-branch version comparison**: Version comparison mistakenly used string ordering, so `8.4.9` was treated as newer than `8.4.10` and an older executable could be selected. The newest patch is now always used.
- **php_redis extension target**: Adding a new PHP branch can no longer copy the extension into another PHP version's `ext/` directory and break loading.
- **Archived dependency fallback**: When PHP moves an older release into `archives/`, the downloader and dependency check now follow the fallback URL instead of reporting a dead link.

## Getting Started
1. Download `wincmp-v2.1.7-win-x64.zip`.
2. Extract it into any folder on your system.
3. Double-click `WinCMP_v2.1.7.exe` to launch the control panel.
4. Upgrading from an older version: open "Dependency Manager" -> "Check Updates" -> reinstall PHP to pick up the patched build.
