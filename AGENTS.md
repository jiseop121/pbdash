# AGENTS.md

## 프로젝트 개요
- 이 저장소는 `pbdash`를 개발합니다.
- `pbdash`는 여러 PocketBase 인스턴스를 별칭으로 관리하고, 컬렉션/레코드를 읽기 전용으로 조회하는 로컬 CLI 도구입니다.
- 기본 동작은 안전한 조회 중심이며, 명시적 요구가 없는 한 쓰기 동작을 추가하거나 확장하지 않습니다.

## 기술 스택
- Go 1.25+
- 진입점: `cmd/pbdash/main.go`
- 주요 패키지
  - `internal/app`: 앱 실행, 설정, 종료 코드
  - `internal/cli`: 명령 파싱, REPL, TUI, 출력 포맷
  - `internal/pocketbase`: PocketBase API 접근 및 조회 로직
  - `internal/storage`: 로컬 저장소(db/superuser/context)

## 문서 인덱스
- 문서 루트는 `docs/`입니다.
- `docs/development`는 개발 절차, 실행 방법, 외부 의존성 사용 맥락을 담습니다.
- `docs/specs`는 구현 명세, 기능 계약, 동작 기준 문서를 담습니다.
- `docs/internal`는 내부 협업용 문서를 담습니다.
- `docs/internal`는 Git 추적 대상이 아니므로, 내부 문서에만 의존하는 공개 규칙을 만들지 않습니다.

## 문서 탐색 규칙
- 개발 환경, 테스트, 실행 방법이 필요하면 먼저 `docs/development/development-guide.md`를 봅니다.
- 외부 라이브러리 사용 이유나 수정 영향 범위를 확인할 때는 `docs/development/dependencies/README.md`를 먼저 봅니다.
- PocketBase SDK 동작을 수정하거나 인증/조회 흐름을 건드릴 때는 `docs/development/dependencies/pocketbase-client.md`를 우선 확인합니다.
- CLI/TUI/UI 동작 기준이나 구현 범위를 확인할 때는 `docs/specs` 아래 문서를 먼저 봅니다.
- 내부 체크리스트나 기획 전달 메모가 필요할 때만 `docs/internal`을 참고합니다.
- 문서와 코드가 충돌하면, 실제 구현과 테스트를 우선 확인한 뒤 필요한 문서를 함께 갱신합니다.
- 새 문서를 추가할 때는 먼저 공개 문서인지 내부 문서인지 판단한 뒤 `development`, `specs`, `internal` 중 하나에만 둡니다.

## 작업별 추천 문서
- 릴리스, 버전 정책, brew 배포 작업: 이 문서의 릴리스 섹션과 `docs/development/release-guide.md`를 먼저 확인합니다.
- 로컬 개발, 테스트, E2E 검증: `docs/development/development-guide.md`
- 외부 의존성 수정: `docs/development/dependencies/README.md`
- PocketBase SDK 수정: `docs/development/dependencies/pocketbase-client.md`
- CLI 핵심 동작 및 Track 1 구현 기준: `docs/specs/cli-core-implementation-spec.md`
- `-ui` 관련 Track 2 기준: `docs/specs/ui-mode-dev-spec.md`
- 내부 기획 점검 맥락: `docs/internal/pm-review-checklist.md`

## 작업 원칙
- 큰 리팩터링보다 작은 단위의 명확한 변경을 우선합니다.
- 기존 CLI 동작은 명시적 요구가 없으면 유지합니다.
- 패키지 경계를 흐리지 않습니다.
- 기능 변경과 무관한 정리 작업은 섞지 않습니다.
- 새 의존성 추가는 꼭 필요할 때만 합니다.
- 코드 수정이 발생하면 관련 문서도 항상 함께 검토하고, 필요한 업데이트를 같은 작업 범위에 반영합니다.
- 사용자에게 보이는 동작이 바뀌면 `README.md` 반영 여부도 함께 확인합니다.

## 브랜치와 PR 원칙
- 모든 수정은 직접 기본 브랜치에 반영하지 않습니다.
- 항상 작업용 브랜치를 새로 만든 뒤 작업합니다.
- 변경사항은 의미 단위로 적절히 커밋합니다.
- 작업이 끝나면 Pull Request를 올려 리뷰 가능한 형태로 제출합니다.
- Pull Request를 올린 뒤에는 Copilot에게도 리뷰를 요청합니다.
- 기능 변경, 버그 수정, 리팩터링, 릴리스 관련 수정은 가능한 한 서로 분리된 PR로 다룹니다.
- 릴리스 대상 PR은 머지 전에 다음 버전 번호를 먼저 확정하고, 버전 관련 파일을 그 다음 버전으로 맞춘 상태에서 머지합니다.
- 버전 관련 파일에는 최소한 앱 버전 상수와 `Formula/pbdash.rb`의 버전 문자열이 포함됩니다. 같은 릴리스에 다른 버전 표기 파일이 추가되면 그 파일도 같은 PR에서 함께 맞춥니다.

## 자주 사용하는 명령
- `go test ./...`
- `make test`
- `make e2e`
- `go build -o pbdash ./cmd/pbdash`
- `./pbdash -c "version"`
- `./pbdash -c "help"`

## 로컬 PocketBase 검증
- 서버 실행: `make pocketbase-serve`
- superuser 준비: `make pocketbase-superuser`
- E2E 검증이 꼭 필요한 경우에만 `make e2e`를 사용합니다.

## 테스트 원칙
- 의미 있는 변경에는 기본적으로 `go test ./...`를 실행합니다.
- CLI 파싱, REPL, TUI, storage, PocketBase 조회 로직을 변경하면 관련 테스트를 함께 수정하거나 추가합니다.
- 가능하면 결정적인 단위 테스트를 우선합니다.
- 변경 범위와 무관한 테스트 실패는 숨기지 말고 명확히 기록합니다.

## 릴리스 원칙
- 이 프로젝트는 당분간 `v1.0.0`을 출시하지 않습니다.
- 모든 버전 업데이트는 `v0.x.y` 범위에서만 관리합니다.
- 메이저 버전 업데이트는 고려하지 않고, `minor`, `patch` 업데이트만 다룹니다.
- 버전 태그는 항상 `v0.x.y` 형식을 사용합니다.
- 릴리스 작업 전에는 워킹 트리가 깨끗해야 합니다.
- 릴리스 전 기본 검증은 `go test ./...` 입니다.
- 명시적 요청이 없으면 릴리스 절차를 임의로 바꾸지 않습니다.
- PR 머지 후에 릴리스하는 경우에도, 머지 전에 이미 버전 관련 파일은 다음 릴리스 버전으로 올라가 있어야 합니다.

## 버전 해석 기준
- 마이너 버전은 하위 호환성이 있는 API 변경사항을 의미합니다.
- 기존 기능을 유지한 채 새로운 기능을 추가하는 경우 마이너 버전을 올립니다.
- 기존 API를 즉시 제거하지 않고 deprecated 처리하는 경우도 마이너 버전을 올립니다.
- 즉, 기존 사용자가 큰 수정 없이 업그레이드할 수 있어야 마이너 버전 대상입니다.

- 패치 버전은 표면상(public surface) API 변경사항이 없는 수정에 사용합니다.
- 단순 버그 수정, 안정성 개선, 내부 리팩터링, 테스트 보강, 문서 보완 등이 이에 해당합니다.
- 업그레이드를 권장하지만 외부 사용 방식은 바뀌지 않아야 합니다.

## 다음 버전 제안 규칙
- 항상 최신 태그 또는 최신 GitHub Release를 기준으로 다음 버전을 제안합니다.
- 하위 호환 신규 기능 추가 또는 deprecated 처리: minor 업데이트를 제안합니다.
- public API 변화 없는 버그 수정, 리팩터링, 테스트, 문서 변경: patch 업데이트를 제안합니다.
- breaking change가 필요하더라도 지금은 `v1`을 올리지 않으므로, 원칙적으로 그러한 변경은 피합니다.
- 불가피하게 breaking change가 필요한 경우에는 바로 진행하지 말고 먼저 별도 논의 또는 이슈로 합의합니다.

## 릴리즈 노트 지침
- GitHub Release 노트는 태그 푸시 후 GitHub Actions가 자동 생성합니다.
- 관련 워크플로우는 `.github/workflows/release.yml` 입니다.
- 수동으로 릴리즈 노트를 따로 작성하거나 덮어쓰는 작업은 사용자가 명시적으로 요청한 경우에만 합니다.
- 새 버전 릴리스 시 먼저 태그를 만들고 푸시해야 합니다.
- 표준 절차는 `make release-tag VERSION=x.y.z` 입니다.
- 이미 존재하는 태그를 다시 만들지 않습니다.
- 릴리즈 노트 내용이 필요하면 우선 GitHub auto-generated notes를 기준으로 확인합니다.
- 릴리스 대상 PR은 머지와 함께 태그 릴리즈 노트 초안을 준비합니다. 기본 템플릿은 `docs/development/release-note-template.md`를 사용합니다.

## Homebrew 릴리스 지침
- Homebrew 릴리스는 GitHub 태그/릴리스가 먼저 존재해야만 진행합니다.
- 표준 절차는 `make release-brew VERSION=x.y.z` 입니다.
- brew 릴리스는 다음을 한 번에 처리합니다.
  - macOS `darwin-arm64`, `darwin-amd64` 바이너리 tar.gz 빌드
  - GitHub Release asset 업로드
  - `Formula/pbdash.rb`의 URL과 SHA256 갱신
  - Formula 변경 커밋 및 푸시
  - brew 설치 스모크 테스트
- 아티팩트 이름은 항상 아래 형식을 유지합니다.
  - `pbdash-v<x.y.z>-darwin-arm64.tar.gz`
  - `pbdash-v<x.y.z>-darwin-amd64.tar.gz`
- Formula 이름과 바이너리 이름은 모두 `pbdash`를 유지합니다.
- 빌드 대상 엔트리포인트는 항상 `./cmd/pbdash` 입니다.
- brew 스모크 테스트에서 설치된 `pbdash -c "version"` 결과가 릴리스 버전과 다르면 실패로 간주합니다.

## 릴리스 순서
1. 워킹 트리가 clean 상태인지 확인합니다.
2. `go test ./...` 통과를 확인합니다.
3. `make release-tag VERSION=x.y.z`를 실행합니다.
4. GitHub Release가 생성되었는지 확인합니다.
5. `make release-brew VERSION=x.y.z`를 실행합니다.
6. Release asset 2개와 `Formula/pbdash.rb` 갱신 여부를 확인합니다.
7. brew 설치 후 `pbdash -c "version"`이 기대 버전인지 확인합니다.

## 금지사항
- 태그가 없는데 brew 릴리스를 먼저 실행하지 않습니다.
- 기존 태그를 재사용하거나 덮어쓰지 않습니다.
- Formula URL, SHA, 바이너리 이름을 수동으로 임의 변경하지 않습니다.
- 릴리스와 무관한 수정사항을 릴리스 커밋에 섞지 않습니다.

## 판단이 애매할 때
1. 먼저 변경 대상 패키지의 테스트 파일을 읽습니다.
2. 새로운 추상화 추가보다 기존 패턴을 우선 따릅니다.
3. 사용자 동작이 바뀌면 문서 반영 여부를 함께 확인합니다.
