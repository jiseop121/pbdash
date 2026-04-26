# CLAUDE.md

## 릴리스 절차

릴리스 관련 작업을 시작하기 전에 반드시 `docs/reference/development/release.md`를 먼저 읽는다.

### 태그 생성은 반드시 make release-tag 사용

```bash
make release-tag VERSION=x.y.z
```

`git tag` / `git push origin vX.Y.Z` 를 직접 실행하지 않는다.
스크립트가 annotated tag 생성, 테스트 실행, 중복 태그 검사를 포함한다.

### CHANGELOG 작성 전 프로세스 확인 순서

1. `docs/reference/development/release.md` 확인
2. CHANGELOG 항목 작성 및 커밋
3. `make release-check VERSION=x.y.z` 로 사전 검증
4. `make release-tag VERSION=x.y.z` 로 태그 생성 및 푸시
