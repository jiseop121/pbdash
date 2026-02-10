# Pocketbase Multiview - Structure First 설계서 (V0)

이 문서는 `$structure-first` 원칙으로 `pocketbase-multiview` / `pbmulti` 프로젝트를 구현하기 위한 설계 기준이다.
목표는 "읽히는 성공 경로(Primary Flow)"를 먼저 고정하고, 경계(I/O)와 Atom을 최소 단위로 정리하는 것이다.
문서 간 우선순위 충돌은 `docs/docs-index.md`를 따른다.

---

## 1) Intent

`pbmulti`의 핵심 실행 경로(`-ui`, `-c`, script, repl`)를 한 번에 읽히도록 구성하고, 기능 확장은 Atom 계약과 테스트 계약으로 안전하게 누적한다.

---

## 2) Primary Flow

### 2.1 런타임 1차 흐름 (Main)

1. 입력 인자/플래그를 파싱한다.
2. 실행 모드(`ui`, `one-shot`, `script`, `repl`)를 결정한다.
3. 선택된 모드 실행기를 1개 호출한다.
4. 실행 결과를 표준 출력/표준 오류와 종료 코드로 반환한다.
5. 실패 시 공통 에러 규약(`ERR_CODE: message`)으로 종료한다.

### 2.2 UI 모드 흐름 (`pbmulti -ui`)

1. 서버 설정(host, port, no-browser)을 검증한다.
2. 내장 정적 리소스(`preview`, `styles`)로 HTTP 서버를 시작한다.
3. `-no-browser=false`면 브라우저 열기를 시도한다.
4. `/healthz`로 상태를 제공한다.
5. 종료 신호를 받아 안전하게 셧다운한다.

### 2.3 CLI 모드 흐름 (`-c`, script, repl)

1. `-c`면 단일 명령 실행 후 즉시 종료한다.
2. script 파일 인자가 있으면 라인 단위로 실행하고 첫 실패에서 종료한다.
3. 둘 다 없으면 REPL 루프를 시작한다.
4. 모든 모드는 동일한 명령 dispatcher를 사용한다.

---

## 3) Boundaries

### 3.1 I/O Boundary

- OS 인자/환경 읽기
- stdin/stdout/stderr
- 파일 읽기(script)
- TCP 포트 바인딩(UI 서버)
- 브라우저 실행(OS command)
- embed 리소스 조회

### 3.2 Domain Boundary

- 모드 결정 규칙
- 모드 충돌 검증 규칙
- 종료 코드 매핑 규칙
- 명령어 계약(입력/출력/실패)

### 3.3 Transform Boundary

- raw argv -> `RunConfig`
- script line -> `Command`
- 실행 에러 -> `ExitResult`

---

## 4) Single Composition Point

Primary orchestration은 `internal/app/run.go` 하나에서만 수행한다.

- `cmd/pbmulti/main.go`는 `RunConfig` 생성 후 `app.Run(config)`만 호출한다.
- 실제 분기(`ui`, `one-shot`, `script`, `repl`)는 `app.Run`에서만 결정한다.
- Atom 간 직접 호출은 금지하고, orchestration을 통해서만 조합한다.

---

## 5) Atoms (role-fixed)

아래 Atom은 V0에서 고정 역할을 가진다.

1. `parseArgs(argv) -> (RunConfig, error)`
- 입력: `[]string`
- 출력: 모드/옵션이 확정된 `RunConfig`

2. `validateMode(config) -> error`
- 입력: `RunConfig`
- 출력: 충돌 여부 에러

3. `runUI(ctx, config) -> error`
- 입력: 서버 설정
- 출력: UI 서버 실행 결과

4. `runOneShot(ctx, commandText) -> error`
- 입력: 단일 명령 문자열
- 출력: 명령 실행 결과

5. `runScript(ctx, path) -> error`
- 입력: script 파일 경로
- 출력: 첫 실패 또는 성공

6. `runREPL(ctx) -> error`
- 입력: 없음(표준 입출력 사용)
- 출력: REPL 종료 결과

7. `dispatchCommand(ctx, command) -> CommandResult`
- 입력: 파싱된 command
- 출력: 공통 결과 구조체(`ok`, `message`, `code`)

8. `mapErrorToExitCode(err) -> int`
- 입력: domain error
- 출력: `{0,1,2,3}` 종료 코드

9. `startHTTPServer(config, assets) -> (ServerHandle, error)`
- 입력: host/port, embed fs
- 출력: 서버 핸들

10. `openBrowser(url) -> error`
- 입력: URL
- 출력: 브라우저 오픈 결과

---

## 6) Package / File 설계

```text
cmd/pbmulti/main.go

internal/app/run.go
internal/app/config.go
internal/app/errors.go
internal/app/exit_code.go

internal/cli/repl.go
internal/cli/script.go
internal/cli/dispatch.go
internal/cli/command_parser.go

internal/ui/server.go
internal/ui/routes.go
internal/ui/browser.go

internal/fs/embed.go
internal/buildinfo/version.go
```

읽기 순서 원칙:
- public entry -> orchestrator -> atoms -> utils

---

## 7) 명확한 결정 규칙 (모호성 제거)

1. 기본 모드는 REPL이다.
2. REPL 시작 시 첫 줄에 `Tip: pbmulti -ui`를 출력한다.
3. `-ui`는 `-c` 또는 script 파일과 동시 사용 불가다.
4. `-c`와 script 파일 동시 사용도 불가다.
5. script 문법은 1줄 1명령, `#` 주석, fail-fast다.
6. 표준 출력은 결과, 표준 오류는 오류만 출력한다.
7. 종료 코드는 `0/1/2/3` 고정 규약을 따른다.
8. V0에서 `/api/*`는 404 고정이다.
9. 배포는 prebuilt binary만 사용한다.
10. Linux 지원은 바이너리 제공까지, Ubuntu apt 운영은 제외한다.

---

## 8) 테스트 설계 (Contract-driven)

테스트는 구현 상세가 아니라 Atom 계약을 검증한다.

### 8.1 `parseArgs` 계약 테스트

- 정상: `[]` -> REPL
- 정상: `-ui` -> UI
- 정상: `-c "version"` -> one-shot
- 정상: `script.pbmulti` -> script
- 실패: `-ui -c` -> invalid args
- 실패: `-c` + script -> invalid args

### 8.2 `runScript` 계약 테스트

- 빈 줄/주석 줄 무시
- 명령 3개 모두 성공 시 전체 성공
- 2번째 줄 실패 시 즉시 중단 + `ERR_SCRIPT_LINE_2`

### 8.3 `mapErrorToExitCode` 계약 테스트

- `ErrInvalidArgs` -> `2`
- `ErrRuntime` -> `1`
- `ErrExternal` -> `3`
- `nil` -> `0`

### 8.4 `runUI` 계약 테스트

- 포트 점유 시 `ErrRuntime`
- `-no-browser`면 브라우저 오픈 미시도
- `/healthz`가 200 반환
- `/api/x`가 404 반환

### 8.5 통합 Smoke

- `pbmulti -c "version"` exit code 0
- `pbmulti -ui -no-browser` 실행 후 헬스체크
- `pbmulti -ui -c "version"` exit code 2

---

## 9) 구현 단계 (Structure First 순서)

### Step A: Orchestrator 먼저 고정

- `RunConfig`
- `Run(config)` 모드 분기
- 종료 코드 매핑

### Step B: 실행기 Atom 추가

- `runOneShot`
- `runScript`
- `runREPL`
- `runUI`

### Step C: 경계 Atom 연결

- embed fs
- http server
- browser open

### Step D: 계약 테스트 추가

- parse/mode/script/exit/ui contracts

### Step E: 배포 파이프라인 연결

- darwin/linux 빌드
- release artifacts
- homebrew formula 업데이트

---

## 10) 변경 시 Gate

1. 성공 경로를 위에서 아래로 한 번에 읽을 수 있는가.
2. 새 함수가 Atom 역할을 명확히 가지는가.
3. side effect가 boundary로 밀려났는가.
4. 테스트가 구현이 아닌 계약을 검증하는가.

---

## 11) Completion Evidence

Primary Flow: parse args -> decide mode -> execute single runner -> map error -> return exit code.
Boundaries: argv/stdin/stdout/stderr, script file I/O, HTTP port bind, browser open, embed FS.
Tests: mode parse conflicts, script fail-fast line error, UI health/404 contracts, exit code mapping.
