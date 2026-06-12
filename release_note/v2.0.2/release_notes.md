# WinCMP v2.0.2
This release introduces new features, updates, and fixes to WinCMP.

## What's Changed

### Added
- **Theme Support**: Integrated dark/light/system theme switching.
- **Process Guard & Port Cleanup**: Introduced Windows Job Object to terminate entire runtime process trees instantly.
- **Custom Command Safety**: Restricted custom executable/script executions to the active project root directory.
- **Global Environment Auto-Detection**: Added automatic sensing of global Node.js/Bun installations.
- **Single-Instance Protection**: Added process check to prevent concurrent runs with legacy v1 version.

## Getting Started
1. Download `wincmp-v2.0.2-win-x64.zip`.
2. Extract the archive to any folder on your system.
3. Double-click `WinCMP_v2.0.2.exe` to launch the control panel.