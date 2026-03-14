# Release Guide

이 문서는 `pbdash` 유지보수자가 릴리스를 진행할 때 필요한 절차를 정리한다.

## 범위

- 태그 생성
- GitHub Release 발행 (GoReleaser 자동)
- Homebrew tap 갱신 (GoReleaser 자동)
- 기본 검증 순서

정책 판단 기준, 버전 해석, 금지사항은 `AGENTS.md`의 릴리스 섹션을 따른다. 이 문서는 실제 실행 절차만 다룬다.

## 입력 형식

- `make release-tag`에는 `v` 없이 `x.y.z` 형식으로 넘긴다.
- 실제 태그 이름은 `v0.7.0`처럼 `v` 접두어가 붙는다.

## 사전 준비 (최초 1회)

### 1. `jiseop121/homebrew-pbdash` 탭 레포 생성

GoReleaser가 Formula를 별도 탭 레포에 커밋한다. 없으면 릴리스 시 실패한다.

```
GitHub → New repository → jiseop121/homebrew-pbdash
설명: Homebrew tap for pbdash
Public, README 없이 빈 레포로 생성
```

### 2. `HOMEBREW_TAP_TOKEN` 시크릿 등록

GoReleaser는 `GITHUB_TOKEN`으로 이 레포의 Release는 업로드하지만,
다른 레포(`homebrew-pbdash`)에 커밋하려면 별도 PAT가 필요하다.

1. GitHub → Settings → Developer settings → Personal access tokens → Fine-grained tokens
2. 토큰 이름: `homebrew-tap-writer`
3. Repository access: `jiseop121/homebrew-pbdash` 선택
4. Permissions: `Contents → Read and write`
5. 생성 후 복사
6. `jiseop121/pbdash` 레포 → Settings → Secrets and variables → Actions → `HOMEBREW_TAP_TOKEN` 추가

## 머지 전 준비

릴리즈 대상 변경은 PR 머지 전에 아래를 먼저 끝낸다.

1. 최신 태그 또는 최신 GitHub Release 기준으로 다음 버전을 확정한다.
2. `go test ./...`와 `go build -ldflags "-X github.com/jiseop121/pbdash/internal/app.Version=x.y.z" ./cmd/pbdash && ./pbdash -c "version"` 기준으로 버전 주입이 동작하는지 확인한다.
3. 위 상태 그대로 PR을 머지한다.

> `internal/app/run.go`의 `var Version = "dev"`는 수동으로 바꾸지 않는다. GoReleaser가 빌드 시 `-X` 플래그로 실제 버전을 주입한다.

## 머지 금지 정책

아래 중 하나라도 빠지면 릴리즈 대상 PR은 머지하지 않는다.

1. 다음 버전이 최신 태그 기준으로 확정되지 않음
2. `go test ./...` 검증이 끝나지 않음
3. PR 본문에 릴리즈 영향 범위와 롤백 관점이 비어 있음

## 새 릴리스 절차

```bash
# 1. main 브랜치가 clean 상태인지 확인
git status --short

# 2. 태그 생성 및 푸시
make release-tag VERSION=x.y.z
```

태그가 푸시되면 `.github/workflows/release.yml`이 자동 실행된다.

워크플로우 실행 순서:

1. `go test ./...` — 실패 시 릴리스 전체 중단
2. GoReleaser 빌드 — `darwin-arm64`, `darwin-amd64` 바이너리 빌드
3. GitHub Release 생성 및 아티팩트 업로드
4. `jiseop121/homebrew-pbdash` 레포에 Formula 커밋

```
# 3. GitHub Actions 완료 확인
# https://github.com/jiseop121/pbdash/actions

# 4. GitHub Release에 아티팩트 2개 확인
# https://github.com/jiseop121/pbdash/releases/tag/vx.y.z

# 5. homebrew-pbdash 레포에 formula 커밋 확인
# https://github.com/jiseop121/homebrew-pbdash
```

필요하면 GitHub Actions에서 `workflow_dispatch`로 기존 태그를 다시 지정해 재배포할 수 있다.

## 로컬 GoReleaser 테스트

```bash
brew install goreleaser
make release-dry-run
# dist/ 디렉토리에 생성 결과 확인
```

## Homebrew 설치

```bash
brew tap jiseop121/pbdash
brew install pbdash
pbdash -c "version"
```

## 릴리즈 노트 가이드

기본값은 GitHub auto-generated release notes를 사용한다.

- GoReleaser가 Release를 생성하거나 갱신한다.
- 별도 수동 릴리즈 노트 작성이나 덮어쓰기는 사용자가 명시적으로 요청한 경우에만 한다.
- 릴리즈 대상 PR은 머지 시점에 태그 릴리즈 노트 초안을 함께 준비한다.
- 초안 형식은 `docs/development/release-note-template.md`를 기본으로 사용한다.

수동 작성 시 확인할 기준:

- merged PR과 커밋 목록을 먼저 확인한다.
- 사용자에게 보이는 변경을 우선 정리한다.
- breaking change가 있으면 가장 먼저 명시한다.

현재 공개 릴리스 기준 권장 구성:

- `What's Changed` 제목 아래에 섹션을 둔다.
- `Added`: 새 기능, 새 옵션, 새 사용자 흐름
- `Changed`: 동작 변경, 리팩터링, 버그 수정, 설치/배포 변경
- `Breaking`: 호환성 깨짐, 명령/환경변수/경로 변경이 있을 때만 추가한다.

## 확인 포인트

- GitHub Release에 darwin 아티팩트 2개가 모두 올라갔는지 본다.
- `jiseop121/homebrew-pbdash` 레포에 Formula가 갱신됐는지 본다.
- brew 설치 후 `pbdash -c "version"` 결과가 기대 버전인지 본다.
