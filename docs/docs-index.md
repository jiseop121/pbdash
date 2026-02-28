# Multi PocketBase UI - 문서 인덱스 (권한/경계)

이 문서는 `docs/` 문서의 역할 경계와 우선순위를 정의한다.
문서 간 충돌이 나면 이 문서를 먼저 따른다.

## 1) 현재 실행 우선순위

현재 활성 트랙:
1. Track 1 (`docs/cli-core-dev-spec.md`) 우선 진행
2. Track 2 (`docs/ui-mode-dev-spec.md`)는 Track 1 완료 후 시작

## 2) 문서 체계

| 문서 | 목적 | 포함 범위 | 제외 범위 |
|---|---|---|---|
| `docs/ui-spec.md` | 제품 기능 요구사항(PRD) | 사용자 동작, 정책, 수용 기준 | 타입 코드, 파일 구조, 구현 순서 |
| `docs/ui-design-spec.md` | UI 시각/상호작용 기준 | 토큰, 레이아웃, 상태별 시각 규칙, 접근성 | 기능 정책, API/상태 전이 로직 |
| `docs/spec-contracts.md` | 타입/상태 계약 | 타입 인터페이스, 상태 전이, 저장 경계 | 화면 UX 문구/컴포넌트 스타일 |
| `docs/cli-core-dev-spec.md` | Track 1 개발 기준 | REPL / one-shot / script, 종료코드, `-ui` 미구현 처리 | UI 서버 계약 |
| `docs/cli-core-implementation-spec.md` | Track 1 구현 상세 | Atom 계약, 실행 순서, 테스트 계약, 파일 배치 | 제품 정책 재정의 |
| `docs/ui-mode-dev-spec.md` | Track 2 개발 기준 | `-ui` 실행, UI 서버/라우팅/헬스체크 | CLI 핵심 동작 재정의 |
| `docs/deployment-cli-brew-spec.md` | 통합 배포 로드맵 | 트랙별 릴리스/통합 게이트 | 트랙 내부 상세 규칙 |
| `docs/structure-first-project-design.md` | 코드 구조 원칙 | 오케스트레이션, boundary, 테스트 구조 | 제품 요구사항 결정 |

## 3) 우선순위 규칙

1. 문서 역할 충돌: `docs/docs-index.md`
2. Track 1 진행 중 실행/CLI 충돌: `docs/cli-core-dev-spec.md`
3. Track 2 진행 중 `-ui` 충돌: `docs/ui-mode-dev-spec.md`
4. 제품 기능 충돌: `docs/ui-spec.md`
5. 시각 규칙 충돌: `docs/ui-design-spec.md`
6. 타입/상태 충돌: `docs/spec-contracts.md`
7. 코드 구조 충돌: `docs/structure-first-project-design.md`

## 4) 중복 금지 규칙

1. 동일 규칙은 한 문서에만 원본(SOT)으로 둔다.
2. 다른 문서에는 요약 대신 링크만 둔다.
3. 구현 예시 코드(스크립트/워크플로/샘플 CSS)는 원칙 문서가 아니라 실행 문서에 둔다.
4. 기획 문서(`ui-spec`)에는 파일명, 함수명, 구현 순서를 넣지 않는다.

## 5) 변경 규칙

1. 기능 정책 변경: `ui-spec` 수정 후 필요 시 `spec-contracts`, `ui-design-spec` 동기화
2. 타입/상태 변경: `spec-contracts` 수정 후 `ui-spec` 용어 동기화
3. Track 1 CLI 변경: `cli-core-dev-spec` 수정 후 `cli-core-implementation-spec`, `README` 명령 예시 동기화
4. Track 2 `-ui` 변경: `ui-mode-dev-spec` 수정 후 `deployment-cli-brew-spec` 동기화
5. 구조 변경: `structure-first-project-design` 수정

## 6) 권장 읽기 순서

기획/PM:
1. `README.md`
2. `docs/docs-index.md`
3. `docs/cli-core-dev-spec.md`
4. `docs/ui-mode-dev-spec.md`

개발:
1. `docs/cli-core-dev-spec.md`
2. `docs/cli-core-implementation-spec.md`
3. `docs/structure-first-project-design.md`
4. `docs/ui-mode-dev-spec.md` (Track 2 시작 시점부터)
5. `docs/spec-contracts.md`
6. `docs/ui-spec.md`
