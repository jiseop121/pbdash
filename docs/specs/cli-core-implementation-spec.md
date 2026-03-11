# Multi PocketBase UI - Track 1 CLI 구현 상세 명세 (Structure First)

이 문서는 `docs/cli-core-dev-spec.md`를 구현 가능한 수준으로 상세화한 Track 1 실행 문서다.
문서 우선순위 충돌은 `docs/docs-index.md`를 따르며, 제품/정책 원본(SOT)은 `docs/cli-core-dev-spec.md`다.
현재 이 문서의 위치는 `docs/specs/cli-core-implementation-spec.md`다.

## 1) Intent

Track 1 범위에서 `pbdash`의 성공 경로(TUI / REPL / one-shot / script)를 한 번에 읽히게 구현하고, 실패를 일관된 오류 형식과 종료 코드(`0/1/2/3`)로 고정한다.

## 2) Primary Flow

### 2.1 런타임 1차 흐름 (Main)

1. `argv`를 파싱해 `RunConfig`를 만든다.
2. 모드/플래그 충돌을 검증한다.
3. 실행 모드(`tui|repl|one-shot|script|ui-reserved`)를 결정한다.
4. 모드 실행기 1개만 호출한다.
5. 결과를 `stdout/stderr` 및 종료 코드로 매핑해 종료한다.

### 2.2 모드별 성공/실패 흐름

#### A. TUI (`pbdash`)
1. 전면 TUI를 시작한다.
2. DB 목록부터 탐색 흐름을 연다.
3. 필요 시 superuser, collection, records, record detail 화면으로 이동한다.
4. records 화면에서는 선택한 row에 `Enter`를 눌러 별도 record detail 화면으로 진입한다.
5. record detail 화면에서는 `Esc`/`Backspace`로 records 화면으로 돌아가고 `y`로 현재 레코드 JSON을 clipboard에 복사할 수 있다.
6. 종료 전까지 탐색 상태를 유지한다.

#### B. REPL (`pbdash -repl`)
1. REPL 루프를 시작한다.
2. 입력 한 줄을 `Command`로 파싱한다.
3. dispatcher로 명령 실행 후 결과를 출력한다.
4. `exit`/EOF 전까지 반복한다.

#### C. One-shot (`pbdash -c "<command>"`)
1. `-c` 텍스트를 단일 명령으로 파싱한다.
2. dispatcher로 1회 실행한다.
3. 결과 출력 후 종료한다.

#### D. Script (`pbdash <script-file>`)
1. UTF-8 파일을 읽고 줄 단위로 순회한다.
2. 빈 줄/`#` 주석 줄은 건너뛴다.
3. 명령 실행 오류가 발생해도 다음 줄 실행을 계속한다(continue-on-error).
4. 각 실패 줄마다 `Error: Script failed at line <N>: <message>`를 즉시 출력한다.
5. `exit`/`quit`를 만나면 그 시점에서 script 실행을 중단한다.
6. 종료 코드는 세션에서 마지막으로 발생한 오류 코드(`1/2/3`)를 따른다. 오류가 없으면 `0`이다.

#### E. UI 예약 플래그 (`pbdash -ui`) - Track 1
1. `-ui`를 예약 플래그로 인식한다.
2. 실제 UI 실행 없이 즉시 실패 처리한다.
3. `stderr`에 `Error: Web UI is under development.`를 출력한다.
4. 종료 코드 `2`로 종료한다.

## 3) Boundaries

### 3.1 I/O Boundaries

- OS 인자 입력: `argv`, `stdin`
- 출력 채널: `stdout`(성공), `stderr`(오류)
- 파일 I/O: script 파일 읽기, `csv|markdown` 출력 파일 쓰기
- 네트워크 I/O: PocketBase GET 요청

### 3.2 Domain Boundaries

- 실행 모드 결정/충돌 규칙
- 명령 유효성 검증 규칙(필수 인자, 조합 제약)
- 오류 형식 규칙(`Error: ...`, `Hint: ...`)
- 종료 코드 매핑 규칙(`0/1/2/3`)
- Track 1 `-ui` 미지원 정책

### 3.3 Transform Boundaries

- `argv -> RunConfig`
- `line(string) -> Command`
- `CommandResult -> RenderedOutput`
- `error -> AppError -> ExitCode`

## 4) Single Composition Point

오케스트레이션은 `internal/app/run.go` 1곳에서만 수행한다.

- `cmd/pbdash/main.go`는 `app.Run(...)`만 호출한다.
- 모드 분기/실행기 선택/종료 코드 결정은 `app.Run`이 단독 소유한다.
- 실행기(Atoms) 간 직접 호출은 금지하고, 조합은 `app.Run`에서만 수행한다.

## 5) Atoms (역할 고정 + I/O 계약)

| Atom | 입력 | 출력 | 책임 |
|---|---|---|---|
| `parseRunConfig` | `[]string` | `RunConfig, error` | 플래그/위치 인자 파싱 |
| `validateRunConfig` | `RunConfig` | `error` | 모드 충돌/필수 조합 검증 |
| `resolveMode` | `RunConfig` | `ExecMode` | `tui/repl/one-shot/script/ui-reserved` 결정 |
| `runTUI` | `context` | `error` | 기본 전면 TUI 실행 |
| `runOneShot` | `context, commandText` | `error` | 단일 명령 실행 |
| `runScript` | `context, path` | `error` | 파일 라인 실행 + continue-on-error + 마지막 오류 코드 반영 |
| `runREPL` | `context` | `error` | 인터랙티브 명령 루프 |
| `dispatchCommand` | `context, Command` | `CommandResult` | 서브커맨드 실행 진입점 |
| `renderSuccess` | `CommandResult` | `stdout text` | 성공 출력 포맷 생성 |
| `renderError` | `AppError` | `stderr text` | 오류/힌트 출력 포맷 생성 |
| `mapErrorToExitCode` | `error` | `int` | `0/1/2/3` 매핑 |

원칙:
- 각 Atom은 한 문장으로 책임 설명이 가능해야 한다.
- Domain 결정 로직(예: 포맷+`--out` 규칙)은 소유 Atom 1곳에만 둔다.

## 6) 핵심 데이터 계약

### 6.1 RunConfig

```go
type RunConfig struct {
    UIEnabled   bool
    REPLEnabled bool
    CommandText string // -c
    ScriptPath  string // positional arg
    Stdout      io.Writer
    Stderr      io.Writer
    Stdin       io.Reader
}
```

규칙:
- `CommandText`와 `ScriptPath` 동시 존재 금지
- `UIEnabled`와 (`CommandText` 또는 `ScriptPath` 또는 `REPLEnabled`) 동시 존재 금지
- `REPLEnabled`와 (`CommandText` 또는 `ScriptPath`) 동시 존재 금지

### 6.2 ExecMode

```go
type ExecMode string

const (
    ModeTUI        ExecMode = "tui"
    ModeREPL       ExecMode = "repl"
    ModeOneShot    ExecMode = "one-shot"
    ModeScript     ExecMode = "script"
    ModeUIReserved ExecMode = "ui-reserved"
)
```

### 6.3 AppError

```go
type AppErrorKind string

const (
    ErrInvalidArgs AppErrorKind = "invalid_args" // exit 2
    ErrRuntime     AppErrorKind = "runtime"      // exit 1
    ErrExternal    AppErrorKind = "external"     // exit 3
)

type AppError struct {
    Kind    AppErrorKind
    Message string
    Hint    string // optional
}
```

## 7) 명령 계약 구현 상세

### 7.1 공통 규칙

- 모든 실행 경로(TUI / REPL / one-shot / script)는 같은 dispatcher를 공유한다.
- PocketBase API는 GET만 허용한다.
- 쓰기 동작 요청은 `ErrInvalidArgs`로 거절한다.

### 7.2 `db` / `superuser`

- `db add/list/remove`, `superuser add/list/remove`를 지원한다.
- 별칭 고유성 검증은 add 시점에서 수행한다.
- 누락 인자/중복 별칭/형식 오류는 종료 코드 `2`.

### 7.3 `api` 조회

- 지원: `collections`, `collection`, `records`, `record`
- `records` 쿼리 옵션: `--page --per-page --sort --filter`
- 쿼리 값이 유효하지 않으면 종료 코드 `2`

### 7.4 출력 포맷

- 기본 `--format table`
- `table`: `stdout`에 ASCII 테이블 + 마지막 줄 `N rows`
- `csv|markdown`: `--out` 필수, 파일 저장 후 `stdout` 요약 1줄 출력
- 다건 조회 메타: `page`, `perPage`, `totalItems`, `totalPages`

## 8) 오류/종료 코드 상세

### 8.1 출력 포맷

- 오류 1행: `Error: <plain English message>`
- 오류 2행(선택): `Hint: <next action>`
- 내부 코드(`ERR_...`)는 사용자 출력에 노출하지 않는다.

### 8.2 종료 코드 매핑

- `0`: 성공
- `1`: 런타임 실패(파일 읽기/쓰기 실패 등)
- `2`: 인자/모드/미지원 기능(`-ui`) 오류
- `3`: 외부 의존 실패(인증 실패/네트워크 실패 등)

## 9) 파일/패키지 배치 (Track 1 기준)

```text
cmd/pbdash/main.go

internal/app/run.go
internal/app/config.go
internal/app/errors.go
internal/app/exit_code.go

internal/cli/repl.go
internal/cli/script.go
internal/cli/dispatch.go
internal/cli/command_parser.go
internal/cli/formatters.go

internal/pocketbase/client.go
internal/pocketbase/query.go

internal/storage/db_store.go
internal/storage/superuser_store.go
```

읽기 순서 원칙:
- public entry -> orchestrator -> mode executors -> command atoms -> utility

## 10) Tests (Contract-driven)

### 10.1 `parseRunConfig` / `validateRunConfig`

table case 필수:
- `[]` -> `ModeTUI`
- `-repl` -> `ModeREPL`
- `-c "version"` -> `ModeOneShot`
- `script.txt` -> `ModeScript`
- `-ui` -> `ModeUIReserved`
- `-c + script` -> `ErrInvalidArgs`
- `-ui + -c` -> `ErrInvalidArgs`
- `-ui + script` -> `ErrInvalidArgs`
- `-repl + -c` -> `ErrInvalidArgs`
- `-repl + script` -> `ErrInvalidArgs`

### 10.2 `runScript`

table case 필수:
- 빈 줄/주석만 포함 -> 성공
- 3줄 모두 성공 -> 성공
- 2번째 줄 실패 + 3번째 줄 성공 -> 계속 실행 + line 2 오류 메시지
- 다중 실패 -> 모든 실패 줄 오류 출력 + 마지막 실패의 종료 코드 반환
- `exit`/`quit` 포함 -> 해당 줄에서 실행 중단 + 중단 전 마지막 오류 코드 반환
- UTF-8 아닌 파일 -> `ErrRuntime`

### 10.3 `dispatchCommand`

계약 테스트:
- `db/superuser/api` 각 명령의 필수 인자 누락 시 종료 코드 `2`
- API 읽기 전용 위반 시 종료 코드 `2`
- 없는 `db`/`superuser` 별칭 참조 시 종료 코드 `2`

### 10.4 포맷터

- `table` 출력에 ASCII header/row/`N rows` 포함
- `csv|markdown`에서 `--out` 누락 시 종료 코드 `2`
- 결과 0건이면 빈 테이블 + `0 rows`

### 10.5 종료 코드/채널 분리

- 성공 결과는 `stdout`만 사용
- 실패 결과는 `stderr`만 사용
- 인증/네트워크 실패는 종료 코드 `3`

## 11) 구현 단계 (Changes)

1. `internal/app/run.go`에 Primary Flow 오케스트레이션 고정
2. `RunConfig`/`ExecMode`/`AppError` 계약 타입 고정
3. REPL / one-shot / script 실행기 Atom 구현
4. Track 1 `-ui` 예약 실패 경로 구현
5. dispatcher 기반 `db/superuser/api` 조회 명령 연결
6. 출력 포맷터(`table/csv/markdown`)와 `--out` 검증 구현
7. 오류 포맷/종료 코드/출력 채널 분리 일괄 적용
8. Atom 계약 테스트 + CLI smoke 테스트 추가

## 12) Refactor Check (Track 1)

- Parameter growth reason: `RunConfig`는 실행 책임(입출력/모드 결정) 범위에서만 확장한다.
- Decision owner: 모드 충돌/종료 코드/포맷 조합 규칙은 각각 소유 Atom 1곳에만 둔다.
- Legacy path status: Track 1에서 `-ui` 실구현 경로는 제거 상태로 유지하고, Track 2에서 `docs/ui-mode-dev-spec.md` 기준으로 신규 도입한다.

## 13) Version Management

- 모든 Track 1 구현 변경은 Git으로 누락 없이 버전 관리한다.
- 기능 변경 시 관련 테스트와 문서를 같은 변경 단위로 함께 관리한다.
- 릴리스/배포 기준 버전은 `version` 명령 출력과 동기화한다.

## 14) Completion Evidence

Primary Flow:
1) `argv`를 `RunConfig`로 변환한다.
2) 충돌 검증 후 실행 모드를 결정한다.
3) 모드 실행기 1개를 호출한다.
4) 결과를 `stdout/stderr`로 출력한다.
5) `mapErrorToExitCode`로 종료한다.

Boundaries: `argv/stdin/stdout/stderr`, script file I/O, output file I/O, PocketBase GET network call.

Tests: 모드 결정/충돌, script continue-on-error(line 번호), 읽기 전용 규칙, 포맷-옵션 조합, 종료 코드/출력 채널 분리를 계약 테스트로 검증한다.
