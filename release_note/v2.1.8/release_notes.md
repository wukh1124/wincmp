# WinCMP v2.1.8

Release date: 2026-10-22

This release closes the lifecycle gaps of superseded PHP runtimes and makes the local build entry point predictable.

## What's Changed

### Fixed
- **Stale PHP cleanup no longer fails on locked DLLs**: Removing a replaced PHP runtime no longer stops at `Access is denied` when `php-cgi` still holds a DLL. The app stops the managed runtime, terminates leftover `php-cgi` processes inside that version directory, and retries the removal before reporting a failure.
- **Replaced runtimes stop during upgrade**: Installing a newer patch of the same PHP branch now stops the runtime it replaces and clears its auto-start entry immediately, so an invisible runtime can no longer keep serving in the background.
- **Restore-state pruning**: PHP versions whose directory is gone or that were replaced by a newer patch are dropped from the "restore last state" list, preventing phantom PHP starts on the next launch.

### Changed
- **Build entry point**: The Taskfile now refuses to build while WinCMP is running instead of silently terminating it, verifies that `build/bin` was really cleaned, and reads the version without requiring Git Bash. A new `task doctor` task reports Go, Wails and WebView2 versions.

## Getting Started
1. Download `wincmp-v2.1.8-win-x64.zip`.
2. Extract it into any folder on your system.
3. Double-click `WinCMP_v2.1.8.exe` to launch the control panel.
4. Upgrading from v2.1.7: if an older PHP patch is still shown as running after the upgrade, it is stopped automatically on launch; the cleanup entry disappears once the directory is removed.
