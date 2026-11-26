# WinCMP v2.1.11

Release date: 2026-11-26

This update adds a built-in Composer console for PHP projects, clearer dashboard service alerts, and smoother automatic updates and multi-PHP compatibility.

## What's Changed

### Added
- **Built-in Composer Console for PHP Projects**: Open a dedicated Composer panel directly from PHP and Laravel project menus, automatically using the project's PHP version with quick command hints and searchable run history.
- **Prioritized Dashboard Service Alerts**: When projects are enabled, the dashboard highlights the single most important issue first—missing Caddy, missing PHP versions, or Caddy/PHP not running yet—with a direct Download or Start button.
- **Python HTTP Project Type**: Added a `Python HTTP` option to the project type selector with a preconfigured HTTP server command (default port `8000`) for quick local serving.

### Changed
- **Cleaner Project Action Menu**: Combined row action buttons into a single dropdown menu so project lists stay tidy and easier to use on smaller windows.
- **No More Button Flicker on Startup or Tab Switching**: Service status buttons now load smoothly without briefly flashing between states when launching WinCMP or switching tabs.

### Fixed
- **More Reliable Auto-Update Restarts**: Updating and restarting from an older version no longer fails due to leftover ports or busy processes, and previously running services are reliably restored afterward.
- **No False Crash Logs on Manual Stop**: Stopping Caddy, MariaDB, PHP, or project runtimes manually no longer logs misleading unexpected-exit errors.
- **Better Multi-Version PHP & Composer Compatibility**: Running different PHP versions or Composer scripts no longer fails if an extension in your shared settings is unavailable for that version, and Composer progress bars now update cleanly instead of repeating lines.
- **Dashboard Alert Visual Polish**: Removed the duplicate warning icon next to the alert title and softened the yellow warning background in the Sketch theme.
- **Respect Selected Python Framework on Launch**: Fixed an issue where starting a Python project without a custom command ignored the selected framework and re-detected from files, ensuring the launched command matches the preview.

### Dependencies
- **Added PHP 8.6.0**: PHP `8.6.0` is now available for one-click download and installation in the Dependency Manager.

## Getting Started
1. Download `wincmp-v2.1.11-win-x64.zip`.
2. Extract the archive into any folder on your system.
3. Double-click `WinCMP.exe` to launch the control panel.
4. Upgrading from v2.1.10 (or earlier): use the built-in updater for a smooth restart while keeping your previously active services.
