# WinCMP v2.1.0
This release introduces new features, updates, and fixes to WinCMP.

## What's Changed

### Added
- **Redis Server & PHP Redis Extension Support**: Added one-click installation and service lifecycle management for Redis (default port 6379); automatically configures `php_redis` extension on PHP setup with version compatibility checks.
- **Project Drag-and-Drop Reordering**: Supported intuitive project list reordering via a dedicated drag handle to prevent accidental triggers, with persistent ordering saved in `wincmp.json`.
- **PHP OPcache & Core Setting Auto-Migration**: Automatically validates and optimizes OPcache configurations (including JIT and memory limits) and enables recommended extensions on PHP startup.
- **Open Install Directory Shortcut**: Retains action buttons after dependency download completion and adds an "Open Install Directory" shortcut for direct folder access.
- **Release Management & Pre-flight Validator**: Introduced automated release pre-flight validation scripts and standardized dual-language changelog tooling.

### Changed
- **Compact Single-Line Terminal Log Header**: Redesigned the terminal log header into a streamlined single-line layout inspired by VS Code, integrating service tabs, clearing, and log filtering.
- **Responsive Layout & Font Scaling**: Switched project table headers to rem-based scaling to prevent text wrapping on high-DPI displays, and optimized core service card spacing.
- **Background Performance Optimization**: Removed unused dashboard status overview sections and obsolete port conflict polling to minimize background idle CPU usage.

### Fixed
- **PHP Ini Optimization Trigger**: Fixed an issue where PHP ini optimizations were not reliably applied on first startup by enforcing validation on every PHP process launch.
- **Dependency Dropdown Hover & Dashboard Icons**: Fixed hover visual contrast on dependency dropdown actions and restored the PHP process selector arrow icon.
- **Onboarding Guide Bubble Blur**: Resolved visual blur effects around onboarding guide bubbles under specific theme backgrounds.

## Getting Started
1. Download `wincmp-v2.1.0-win-x64.zip`.
2. Extract the archive to any folder on your system.
3. Double-click `WinCMP_v2.1.0.exe` to launch the control panel.