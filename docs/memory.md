# Project Memory

## 프로젝트 정체성
- `pbdash`: 여러 PocketBase 인스턴스를 별칭으로 관리하고 컬렉션/레코드를 읽기 전용 조회하는 로컬 CLI 도구
- 기본 동작은 read-only; 명시적 요구 없이 쓰기 동작 추가/확장 금지

## 기술 스택
- Go 1.25+, 진입점 `cmd/pbdash/main.go`
- `internal/app`: 실행, 설정, 종료 코드
- `internal/cli`: 명령 파싱, REPL, TUI, 출력 포맷
- `internal/pocketbase`: PocketBase API 접근/조회
- `internal/storage`: 로컬 저장소(db/superuser/context)

## 현재 상태
- Track 1 완료: TUI / REPL / one-shot / script 모드 모두 동작
- Track 2 미착수: `-ui` 플래그는 예약만 됨(즉시 실패 처리 중)

## 활성 불변 규칙
- 버전: `v0.x.y`만 관리; `v1.0.0` 출시 없음
- PR-first 워크플로우: main에 직접 push 금지; 항상 작업 브랜치 → PR
- 릴리스: `make release-tag VERSION=x.y.z` → GoReleaser CI 자동 처리; `make release-brew` 사용 금지(deprecated)
- `internal/app/run.go`의 `var Version = "dev"`는 수동 수정 금지(GoReleaser `-X` 플래그로 주입)
- 패키지 경계 유지; 기능 변경과 무관한 정리 작업 혼재 금지

## 문서 탐색
- 개발 환경/테스트: `docs/reference/development/setup.md`
- 릴리스 절차: `docs/reference/development/release.md`
- 의존성 개요: `docs/reference/dependencies/overview.md`
- PocketBase SDK 상세: `docs/reference/dependencies/pocketbase-client.md`
- CLI Track 1 명세: `docs/reference/specs/cli-core.md`
- CLI Track 2 명세: `docs/reference/specs/ui-mode.md`
- 릴리스 노트 템플릿(인간용): `docs/development/release-note-template.md`
