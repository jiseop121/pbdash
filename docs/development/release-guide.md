# Release Guide

이 문서는 `pbdash` 유지보수자가 태그 릴리스와 Homebrew 배포를 진행할 때 필요한 공개 절차를 정리한다.

## 범위

- 태그 생성과 GitHub Release 발행
- Homebrew 배포 아티팩트 업로드
- `Formula/pbdash.rb` 갱신
- 기본 검증 순서

정책 판단 기준, 버전 해석, 금지사항은 `AGENTS.md`의 릴리스 섹션을 따른다. 이 문서는 실제 실행 절차만 다룬다.

## 입력 형식

- `make release-tag`에는 `v` 없이 `x.y.z` 형식으로 넘긴다.
- 실제 태그 이름은 `v0.4.1`처럼 `v` 접두어가 붙는다.

## 태그 릴리스

```bash
make release-tag VERSION=0.4.1
```

이 명령은 다음 순서로 동작한다.

- `go test ./...`를 실행한다.
- `v0.4.1` 태그를 생성한다.
- 원격 저장소로 태그를 푸시한다.

태그가 푸시되면 [`.github/workflows/release.yml`](/Users/hjs/Personal/multi-pocketbase-ui/.github/workflows/release.yml)이 실행되어 GitHub Release를 생성하거나 갱신한다.

워크플로우 역할:

- `v0.4.1` 형식의 태그만 처리한다.
- 대상 태그를 checkout 한다.
- GitHub auto-generated release notes로 릴리스 본문을 만든다.

필요하면 GitHub Actions에서 `workflow_dispatch`로 기존 태그를 다시 지정해 Release만 재생성할 수 있다.

## Homebrew 배포

GitHub 태그와 Release가 이미 존재하는 상태에서 실행한다.

```bash
make release-brew VERSION=0.4.1
```

이 명령은 다음을 한 번에 처리한다.

- `darwin-arm64`, `darwin-amd64` 바이너리 tar.gz를 빌드한다.
- 현재 레포 Release(`v0.4.1`)에 아티팩트를 업로드한다.
- `Formula/pbdash.rb`의 URL과 SHA256을 갱신한다.
- Formula 변경을 커밋하고 푸시한다.
- Homebrew 설치 스모크 테스트를 수행한다.

아티팩트 이름은 항상 아래 형식을 유지한다.

- `pbdash-v<x.y.z>-darwin-arm64.tar.gz`
- `pbdash-v<x.y.z>-darwin-amd64.tar.gz`

## 실행 순서

1. `git status --short`로 워킹 트리가 clean 상태인지 확인한다.
2. `go test ./...`를 실행한다.
3. `make release-tag VERSION=x.y.z`를 실행한다.
4. GitHub Release가 생성되었는지 확인한다.
5. `make release-brew VERSION=x.y.z`를 실행한다.
6. Release asset 2개와 `Formula/pbdash.rb` 갱신 여부를 확인한다.
7. brew 설치 후 `pbdash -c "version"` 출력이 기대 버전인지 확인한다.

## 확인 포인트

- Release에 darwin 아티팩트 2개가 모두 올라갔는지 본다.
- `Formula/pbdash.rb`가 새 아티팩트 URL과 SHA256으로 갱신됐는지 본다.
- brew 설치 후 `pbdash -c "version"` 결과가 기대 버전인지 본다.
