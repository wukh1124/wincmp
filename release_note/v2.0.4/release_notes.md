# WinCMP v2.0.4
This release introduces new features, updates, and fixes to WinCMP.

## What's Changed

### Added
- **Monorepo Project Support**: Added a Monorepo helper checkbox during project creation to automatically adapt default project names and domain aliases.
- **Quick Settings Onboarding Guide**: Added step-by-step onboarding guide bubbles for sidebar Quick Settings (Theme/Language/Font).
- **Custom Start Command Configuration**: Refactored project properties to introduce a "Use Custom Command" option, dynamically displaying read-only default commands as a reference.
- **Sidebar Lock/Unlock**: Added a lock/unlock button to prevent the sidebar from collapsing automatically when switching pages.

### Changed
- **Auto-Update Check Optimization**: Optimized background auto-update check intervals and cached status to improve startup performance and bandwidth efficiency.
- **Onboarding Bubble Flow & Visuals**: Changed onboarding guide bubbles to trigger sequentially to prevent overlap, and enhanced readability contrast and layout stability across all themes.
- **Micro-interactions for Buttons & UI**: Added a subtle border color change effect when hovering over all major action buttons and optimized the hover style for danger buttons.
- **Theme Renaming**: Renamed the "Claude" theme to "Cream".
- **Enhanced Update Notification Visuals**: Added a pulsing red dot animation to the sidebar update badge for clearer visual notification.

### Fixed
- **Project List Header Penetration**: Fixed an issue where the header was penetrated by scrolling content in the project list, keeping it sticky and opaque across all themes.
- **Custom Framework Start Command Unlock**: Fixed a bug where the start command input field for the Custom Framework could not be edited.
- **Status Indicator Dot Alignment**: Fixed layout shifting of status indicator dots caused by font scaling, locking them to a fixed 8px size.

## Getting Started
1. Download `wincmp-v2.0.4-win-x64.zip`.
2. Extract the archive to any folder on your system.
3. Double-click `WinCMP_v2.0.4.exe` to launch the control panel.