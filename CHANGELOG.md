# Changelog

All notable changes to this project will be documented in this file.

## [0.7.1] - 2026-03-18

### Fixed
- Disable HTTP keep-alives on the PocketBase HTTP client to fix EOF errors
  when connecting through TCP proxies (e.g. kftray Kubernetes port-forwarding).
  Previously, Go's default keep-alive transport caused the proxy to close
  connections immediately, leading to `require_record_token` or
  `resource_not_found (404)` errors during authentication.

## [0.7.0] - 2026-03-15

### Added
- New 2-phase modal UI for DB alias and Superuser management:
  open a list view first, then edit inline without leaving the screen.
- Keyboard shortcuts `n` (new), `e` (edit), `D` (delete) available
  directly from DB alias and Superuser list modals.
- `Tab` toggles focus between the DB dropdown and the superuser table
  in the Superuser modal.

### Changed
- `Esc`/`Backspace` on the DB list screen no longer exits the app.
- Removed `j`/`k` navigation aliases; use arrow keys.
- `b`/`u`/`n`/`e`/`D` keys are now scoped per screen to avoid conflicts.
- Confirm modal no longer shows a redundant border/title;
  `[Enter]`/`[Esc]` hint is displayed inline in the message.

## [0.6.0] - 2026-03-12

### Added
- Press `Enter` on a record to open a dedicated record detail screen. In the detail view, press `y` to copy the current record JSON to the clipboard.
- Collections list now shows collection names instead of internal IDs.

### Changed
- Keyboard navigation across forms and modals now supports arrow keys in addition to `Tab` and `Shift+Tab`.

## [0.5.2] - 2026-03-12

### Changed
- Stability and release infrastructure improvements. No user-visible changes.

## [0.5.1] - 2026-03-12

### Changed
- Restored TUI key focus so `q`, `j/k`, `Enter`, and related keys are reliably captured from the focused view.
- Simplified PocketBase setup and connection handling.

## [0.5.0] - 2026-03-12

### Added
- Added in-TUI local configuration management for saved `db` aliases and `superuser` aliases from the records view.

### Changed
- Refactored records TUI and local store command flows around a primary-flow-first structure.
- Fixed full-screen records TUI shortcut handling so `q`, `j/k`, `Enter`, `r`, and related keys are captured reliably from the focused view.
- Made db alias removal also clear locally saved superusers for that db and keep context/cache references in sync on rename/delete.

## [0.4.1] - 2026-03-10

### Added
- Added `k9s`-style full-screen TUI navigation across saved db aliases, conditional superuser selection, collections, and records.
- Added `pbdash -repl` for explicit access to the legacy REPL flow now that the default entrypoint opens the TUI.

### Changed
- Made bare `pbdash` launch the full-screen navigator TUI instead of the REPL.
- Changed `api records --view auto|tui` to reuse the navigator-based TUI when a TTY is available.
- Reserved `-ui` for the future web UI and changed its current behavior to return an "under development" message.

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
