# WinCMP v2.0.1
This release introduces new features, updates, and fixes to WinCMP.

## What's Changed

### Added
- **Automatic SSL CA Cert Configuration**: Added automatic downloading and configuration of `cacert.pem` for PHP SSL requests when it is missing in the local environment.

### Changed
- **Dependency Optimization**: Completely removed the deprecated `fyne.io/fyne/v2` dependency and legacy resource monitoring GUI code, reducing final binary size.
- **Wails Build Improvements**: Restored Wails build templates (such as `icon.ico` and manifest files) into the Git repository to resolve build failure caused by incorrect `.gitignore` patterns.

## Getting Started
1. Download `wincmp-v2.0.1-win-x64.zip`.
2. Extract the archive to any folder on your system.
3. Double-click `WinCMP_v2.0.1.exe` to launch the control panel.