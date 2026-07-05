# WinCMP v2.0.6
This release introduces new features, updates, and fixes to WinCMP.

## What's Changed

### Added
- **Port Occupancy Detection**: Automatically detects processes occupying the target port when starting a project, prompting the user and allowing safe termination directly from the UI.
- **Custom Start Command Persistence**: Persists custom startup command configurations in the `wincmp.json` profile, preventing settings from being lost when toggling the configuration options or reloading the app.

### Changed
- **UI & Tooltip Polishing**: Enhanced project settings panels by increasing label font sizes and help icon dimensions, and optimized tooltip positioning for better usability.
- **Localization Wording**: Refined UI translation strings and labels regarding custom start commands for clarity.

### Fixed
- **Race Condition in Concurrent Startup**: Implemented concurrency control to prevent multiple projects from starting simultaneously on the same port.

## Getting Started
1. Download `wincmp-v2.0.6-win-x64.zip`.
2. Extract the archive to any folder on your system.
3. Double-click `WinCMP_v2.0.6.exe` to launch the control panel.