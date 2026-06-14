# WinCMP v2.0.3
This release introduces new features, updates, and fixes to WinCMP.

## What's Changed

### Added
- **Onboarding Guide**: Added interactive step-by-step onboarding popovers for project actions and the dependency manager to help users quickly get familiar with the UI.
- **Custom Hosts Error Alerts**: Replaced native browser dialogs with a custom React Alert component for Hosts file write failures, adding a "Do not remind again" option to prevent repetitive alerts.

### Changed
- **Hosts Sync Optimization**: Optimized hosts file automatic update logic to only sync hosts for enabled projects.
- **Simplified Dependency Check**: Streamlined startup dependency checks to scan for Caddy only.
- **Theme Improvements & Renaming**: Renamed the "xAI" dark theme to "Carbon"; optimized the "Sketch" hand-drawn theme UI to resolve text legibility issues in alert banners and project lists against the graph grid background.

## Getting Started
1. Download `wincmp-v2.0.3-win-x64.zip`.
2. Extract the archive to any folder on your system.
3. Double-click `WinCMP_v2.0.3.exe` to launch the control panel.