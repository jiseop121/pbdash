# Multi PocketBase UI - Track 2: `-ui` 기능 개발 명세

이 문서는 Track 2 개발의 단일 기준 문서다.
Track 1(CLI 핵심) 완료 이후에만 적용한다.

## 1) 시작 조건

Track 2 시작 전 선행 조건:
1. `docs/cli-core-dev-spec.md`의 완료 정의 충족
2. CLI 공통 계약(출력/오류/종료 코드) 고정
3. `-ui` 미구현 응답이 제거 가능한 상태

## 2) 목표

- `pbmulti -ui`를 실제 UI 실행 모드로 제공한다.
- 로컬 HTTP 서버 + 정적 리소스 + 헬스체크를 안정적으로 제공한다.
- Track 1 CLI 계약을 깨지 않고 UI 모드를 추가한다.

## 3) 범위

### 3.1 포함
- `-ui` 플래그 실행
- `-host`, `-port`, `-no-browser` 플래그 적용
- `/` `/preview/*` `/styles/*` 라우팅
- `/healthz` 200 응답
- 브라우저 오픈 시도(실패 시 서버 유지)

### 3.2 제외
- 원격 배포형 UI 서버
- 서버 공유 세션
- `/api/*` 백엔드 구현(V0는 404)

## 4) UI 모드 계약

실행:
- `pbmulti -ui` -> UI 서버 시작
- 기본 URL: `http://127.0.0.1:<port>`

라우팅:
- `/` -> `preview/index.html`
- `/preview/*` -> embedded assets
- `/styles/*` -> embedded assets
- `/healthz` -> `200 ok`
- `/api/*` -> `404`

예외 처리:
- 포트 사용 중: 종료 코드 `1`
- 브라우저 오픈 실패: URL 출력 후 서버는 계속 실행
- `-ui` + `-c` 또는 `-ui` + script: 종료 코드 `2`

## 5) 보안/운영 기본값

- host 기본값 `127.0.0.1`
- CORS 비활성(동일 출처)
- directory listing 금지

## 6) Track 2 테스트 게이트

릴리스 전 필수:
1. `go test ./...`
2. `pbmulti -ui -no-browser` 성공
3. `/healthz` 200 확인
4. `/api/*` 404 확인
5. 포트 충돌 시 종료 코드 `1` 확인
6. 모드 충돌(`-ui` + `-c`) 종료 코드 `2` 확인

## 7) 완료 정의 (Track 2)

아래 조건을 모두 만족하면 Track 2 완료:
1. `pbmulti -ui` 경로가 문서 계약대로 동작
2. Track 1 CLI 계약이 유지됨
3. `-ui` 관련 릴리스 게이트가 CI/수동 검증에 반영됨
