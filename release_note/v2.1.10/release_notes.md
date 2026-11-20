# WinCMP v2.1.10

Release date: 2026-11-20

This release switches WinCMP to a fixed executable filename to preserve Windows taskbar pins across updates, and updates the bundled dependencies catalog.

## What's Changed

### Changed
- **Fixed executable filename (`WinCMP.exe`)**: The release archive now ships `WinCMP.exe` instead of a versioned filename (`WinCMP_vX.Y.Z.exe`). The in-app updater also replaces the binary using `WinCMP.exe`, so Windows taskbar pins and shortcuts remain valid after upgrading.

### Dependencies
- **Updated bundled dependency catalog**: Refreshed versions and SHA-256 checksums in `conf/dependencies.json` and `docs/dependencies.md`:
  - Caddy: `2.11.7` -> `2.11.9`
  - MariaDB: `11.4.13` -> `11.4.14`
  - PHP: `8.2.35` -> `8.2.36`, `8.3.36` -> `8.3.37`, `8.4.27` -> `8.4.28`, `8.5.12` -> `8.5.13`
  - Mailpit: `1.31.6` -> `1.34.0`
  - Node.js: `24.22.1` -> `24.22.3`
  - Composer: `2.10.3` -> `2.11.2`
  - HeidiSQL: `12.21` -> `12.22`
  - CA Certificates (`cacert`): `2026-10-20` -> `2026-11-05`

## Getting Started
1. Download `wincmp-v2.1.10-win-x64.zip`.
2. Extract the archive into any folder on your system.
3. Double-click `WinCMP.exe` to launch the control panel.
4. Upgrading from v2.1.9: the updater verifies the downloaded executable against `checksums.txt` with SHA-256 and replaces the current binary as `WinCMP.exe`.
