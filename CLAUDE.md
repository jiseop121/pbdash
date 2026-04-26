# CLAUDE.md

## 릴리스 절차

릴리스 관련 작업을 시작하기 전에 반드시 아래 순서로 확인한다.

1. `AGENTS.md` 의 릴리스 섹션 확인
2. `docs/reference/development/release.md` 확인

### 릴리스 실행 순서

```bash
# 1. CHANGELOG 항목 작성 및 커밋
# 2. 사전 검증
make release-check VERSION=x.y.z
# 3. 태그 생성 및 푸시 (CI 자동 실행)
make release-tag VERSION=x.y.z
```

### 금지 사항

- `git tag vX.Y.Z` 직접 실행 금지 — lightweight tag가 생성되고, 테스트·중복 검사가 생략됨
- `git push origin vX.Y.Z` 직접 실행 금지 — 항상 `make release-tag` 를 통해서만 푸시
- CHANGELOG 커밋 없이 태그 생성 금지
- 릴리스 절차 확인 전 태그 생성 금지

### Homebrew tap

Homebrew Formula는 메인 repo가 아닌 별도 tap repo에서 관리한다.

- tap repo: `https://github.com/jiseop121/homebrew-pbdash`
- GoReleaser가 태그 푸시 시 자동으로 `Formula/pbdash.rb` 를 해당 repo에 커밋함

설치 명령:

```bash
brew tap jiseop121/pbdash https://github.com/jiseop121/homebrew-pbdash
brew install jiseop121/pbdash/pbdash
```

`brew tap jiseop121/pbdash https://github.com/jiseop121/pbdash` (메인 repo URL) 로 tap하면
formula가 없어 설치 불가 — 반드시 `homebrew-pbdash` repo URL을 사용한다.
