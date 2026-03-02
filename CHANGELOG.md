# Changelog

All notable changes to this project will be documented in this file.

## [0.2.2] - 2026-03-02

### Changed
- Renamed CLI binary from `pbmulti` to `pbviewer` across command surface, docs, and release scripts.
- Updated runtime/script behavior to continue after per-command failures and return the last non-zero error code.

### Breaking
- Replaced `PBMULTI_HOME` with `PBVIEWER_HOME`.
- Replaced `PBMULTI_SUPERUSER_KEY_B64` with `PBVIEWER_SUPERUSER_KEY_B64`.

## [0.2.1] - 2026-03-02

### Added
- Added automated GitHub Release notes workflow on `v*` tag push.
- Added `make release-tag VERSION=x.y.z` command for release tagging.

## [0.2.0] - 2026-03-02

### Added
- Implemented Track 1 CLI runtime and command surface (`db`, `superuser`, `api`).
- Added deterministic E2E smoke test flow against a real PocketBase instance.
- Added MIT License.

### Changed
- Hardened endpoint handling and superuser credential storage.
- Improved command help output and developer guidance.

### Removed
- Untracked internal-only development assets (`docs/`, `preview/`, `styles/`) from Git.

## [0.1.0] - 2026-03-01

### Added
- Initial `pbviewer` CLI baseline and command parser behavior.
