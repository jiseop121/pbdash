# Changelog

All notable changes to this project will be documented in this file.

## [0.4.2] - 2026-03-10

### Added
- Added in-TUI local configuration management for saved `db` aliases and `superuser` aliases from the records view.

### Changed
- Refactored records TUI and local store command flows around a primary-flow-first structure.
- Fixed full-screen records TUI shortcut handling so `q`, `j/k`, `Enter`, `r`, and related keys are captured reliably from the focused view.
- Made db alias removal also clear locally saved superusers for that db and keep context/cache references in sync on rename/delete.

## [0.4.0] - 2026-03-09

### Changed
- Rebranded project name to `PocketBase Dash` and renamed CLI binary/command to `pbdash`.
- Renamed Homebrew formula to `pbdash` and updated release artifacts/repo paths to `jiseop121/pbdash`.
- Updated help text, prompts, build/install commands, and docs to use `pbdash`.

### Breaking
- Removed `pbviewer` command compatibility. Use `pbdash`.
- Replaced `PBVIEWER_HOME` with `PBDASH_HOME`.
- Replaced `PBVIEWER_SUPERUSER_KEY_B64` with `PBDASH_SUPERUSER_KEY_B64`.
- Changed default data directory from `~/.pbviewer` to `~/.pbdash`.

## [0.3.0] - 2026-03-02

### Added
- Added interactive REPL line editing with tab completion and command history.
- Added `context` commands (`show`, `use`, `save`, `clear`, `unsave`) to reuse db/superuser targets.
- Added full-screen `api records` TUI view with keyboard navigation and detail panel.

### Changed
- Added `api records --view auto|tui|table` and made `auto` prefer TUI in interactive REPL TTY mode.
- Added filter/sort/column selection interactions inside TUI with server-side re-query.
- Added per-target auth token cache and one-time re-auth retry on 401 responses.
- Increased source-build baseline to Go 1.25+.

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
