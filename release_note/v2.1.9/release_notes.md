# WinCMP v2.1.9

Release date: 2026-10-23

This release rebuilds WinCMP with a patched Go toolchain, because earlier builds shipped a standard library with an unfixed certificate trust vulnerability.

## What's Changed

### Security
- **Rebuilt with Go 1.27.3**: The Windows executable is now compiled by Go 1.27.3, whose `crypto/x509` rejects certificate chains containing a small set of distrusted intermediate CAs (CVE-2026-111943). Releases v2.1.5 through v2.1.8 were built with Go 1.26.2, which has no such mitigation, so every HTTPS download they performed trusted those chains.
- **Toolchain floor raised**: `go.mod` now requires Go 1.27.3, so a local build either switches to 1.27.3 through Go's automatic toolchain handling or stops with an error, instead of silently producing a binary with the unfixed standard library.

### Dependencies
- **Go 1.27.3 as the minimum**: The README badges and the prerequisites section now state Go 1.27.3 as the hard minimum. No Go module dependencies changed, and `go.sum` is untouched.

## Getting Started
1. Download `wincmp-v2.1.9-win-x64.zip`.
2. Extract the archive into any folder on your system.
3. Double-click `WinCMP_v2.1.9.exe` to launch the control panel.
4. Upgrading from v2.1.8: no configuration migration is needed. The updater verifies the downloaded executable against `checksums.txt` with SHA-256 before replacing the current one.
