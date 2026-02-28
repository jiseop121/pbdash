# Multi PocketBase UI - CLI/Brew 통합 배포 로드맵 (Track 분리)

이 문서는 Track 1과 Track 2를 합친 통합 배포 기준을 정의한다.
트랙 내부 구현 상세는 각 트랙 문서를 단일 기준으로 사용한다.

## 1) 트랙 구성

1. Track 1: CLI 핵심 기능
- 기준 문서: `docs/cli-core-dev-spec.md`
- 범위: REPL / one-shot / script, 종료 코드, `-ui` 미구현 처리

2. Track 2: `-ui` 기능
- 기준 문서: `docs/ui-mode-dev-spec.md`
- 범위: UI 서버/라우팅/헬스체크/브라우저 오픈

현재 우선순위:
- 먼저 Track 1만 진행
- Track 1 완료 후 Track 2 진행

## 2) 공통 불변 계약

아래 계약은 두 트랙에서 공통으로 유지한다.

- 바이너리 이름: `pbmulti`
- 오류 형식: `ERR_CODE: message`
- 출력 분리: 성공 `stdout`, 오류 `stderr`
- 종료 코드:
- `0` 성공
- `1` 런타임 실패
- `2` 인자/모드/미지원 기능 오류
- `3` 외부 의존 실패(예약)

## 3) 릴리스 전략

### 3.1 Track 1 단계

- 내부/사전 릴리스는 가능
- `-ui`는 `ERR_UI_NOT_READY` 응답이 정상 동작
- Homebrew 공개 배포는 선택 사항(팀 정책에 따름)

### 3.2 Track 2 완료 후

- `pbmulti -ui`를 포함한 통합 릴리스 수행
- Homebrew 공개 배포 기준은 Track 1 + Track 2 게이트를 모두 통과해야 한다

## 4) 통합 게이트

최종 공개 릴리스 전 필수:
1. Track 1 게이트 전부 통과
2. Track 2 게이트 전부 통과
3. `brew install pocketbase-multiview` 설치 확인
4. `pbmulti version` 및 `pbmulti -ui` 동작 확인

## 5) 변경 관리

- Track 1 변경은 `docs/cli-core-dev-spec.md` 먼저 수정
- Track 2 변경은 `docs/ui-mode-dev-spec.md` 먼저 수정
- 통합 배포 조건 변경 시 이 문서를 동기화
