# Release Note Template

이 템플릿은 릴리즈 대상 PR을 머지할 때 함께 준비하는 태그 릴리즈 노트 초안 기본형이다.
기본값은 GitHub auto-generated notes를 사용하되, 수동 보완이 필요하면 아래 형식으로 초안을 채운다.

## Title

`v0.x.y`

## What's Changed

### Added

- 

### Changed

- 

### Fixed

- 

### Breaking

- None.

## Release Checks

- Version files were updated before merge.
- `go test ./...` passed.
- `pbdash -c "version"` matches the release version.
- GitHub tag/release was created.
- Homebrew artifacts and `Formula/pbdash.rb` were updated.
