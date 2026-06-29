# WinCMP v2.0.5
This release introduces new features, updates, and fixes to WinCMP.

## What's Changed

### Added
- **Onboarding State Persistence**: The completed states of onboarding guide bubbles are now saved to the `wincmp.json` config, preventing them from showing repeatedly on app restarts.

### Changed
- **Resource Monitor Optimization**: Automatically freezes background CPU/RAM utilization polling when the window is hidden or minimized to the tray, minimizing idle CPU overhead.
- **Flicker-Free Startup**: Set the window to start hidden and show programmatically after complete loading to eliminate initial white-screen flickering.
- **Synchronized Theme Transitions**: Synced transition animations between the sidebar and main layout to prevent visual tearing during theme switches.
- **Default Settings Adjusted**: Updated default theme to `sketch` and default font size to `large` to provide a better out-of-the-box appearance.

### Fixed
- **System Tray Freezing**: Locked the Windows tray event loop to a dedicated OS thread to prevent Go routine scheduling from freezing the tray menu and Windows message pump.

## Getting Started
1. Download `wincmp-v2.0.5-win-x64.zip`.
2. Extract the archive to any folder on your system.
3. Double-click `WinCMP_v2.0.5.exe` to launch the control panel.