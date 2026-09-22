# WinCMP v2.1.6

Release date: 2026-09-23

This release introduces comprehensive open-source compliance governance, third-party notices, trademark disclaimers, and enhanced transparency in the Dependency Manager.

## What's Changed

### Added
- **Open-source compliance & third-party notices**: Added `THIRD-PARTY-NOTICES.md` to repository root and release templates, cataloging licenses (SPDX) and official source repositories for managed services (Caddy, MariaDB, Redis, PHP, Node, etc.), Go modules, and npm packages; added trademark disclaimers across dual-language READMEs.
- **Dependency manager license badges & actions**: Added minimalist license badges (e.g. `GPL-2.0`, `Apache-2.0`, `BSD-3-Clause`) alongside service titles, and integrated license identifiers with direct "Source Code ↗" links into the action dropdowns.
- **In-app open-source notices modal**: Added an "Open Source Licenses & Trademarks" link in the dependency manager footer that opens an in-app notice dialog explaining WinCMP architecture, trademark disclaimers, and service licenses with full bilingual (zh-TW/en-US) support.
- **Standalone executable self-unpacking documentation**: Embedded `LICENSE`, `THIRD-PARTY-NOTICES.md`, and dual-language readmes into the binary via Go embed, automatically extracting them non-destructively upon initial startup in a clean folder to ensure full filesystem compliance for standalone exe users.

### Changed
- **Dependency configuration & documentation standardization**: Extended `conf/dependencies.json`, `internal/config/dependencies.go`, and `scripts/check_deps.go` with `license`, `homepage`, and `source_url` fields, and updated `docs/dependencies.md`.
- **Background update check cooldown debounce**: Added a 60-second cooldown period to the dependency manager window to avoid redundant background requests upon frequently toggling the modal.

### Fixed
- **Notice modal Carbon theme transparency & button contrast**: Resolved background opacity bleed-through on the Carbon dark theme, fixed button text contrast on the Close button, and polished interactive hover visual states.

## Getting Started
1. Download `wincmp-v2.1.6-win-x64.zip`.
2. Extract it to any folder on your system.
3. Double-click `WinCMP_v2.1.6.exe` to launch the control panel.
