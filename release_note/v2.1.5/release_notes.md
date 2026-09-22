# WinCMP v2.1.5

Release date: 2026-09-22

This release introduces new features, updates, and fixes to WinCMP.

## What's Changed

### Added
- **Terminal log unread count tooltips**: Category tabs and runtime project tooltips now indicate unread log lines (e.g., `Caddy - 4 lines total (2 unread)`), instantly clearing counters upon switching tabs or projects.
- **Immediate updater checks and manual refresh**: Added a "Check Updates" button with smooth spinner animations in the Version Update view, automatically fetching the latest release if over 60 seconds have elapsed and enabling cache-bypass to eliminate redundant multi-step upgrades.

### Changed
- **Dependency manager update UX refinements**: Silently checks for updates in the background when opening the manager, guarantees smooth transition spinners on manual checks, eliminates blocking alert modals, and adds subtle fade transitions when configuration contents change.

### Fixed
- **Windows Explorer file path and reveal fallback**: Resolved an issue where Explorer defaulted to the Documents folder due to argument quotation escapes when revealing files; native command lines are now used, with safe fallback to opening the containing directory if log files have not yet been created.
- **Updater notification badge flicker on launch**: Kept the sidebar update badge calm on launch until background verification confirms a new version, and automatically purged stale update flags upon startup and post-update restarts to eliminate distracting badge flash.

## Getting Started
1. Download `wincmp-v2.1.5-win-x64.zip`.
2. Extract it to any folder on your system.
3. Double-click `WinCMP_v2.1.5.exe` to launch the control panel.
