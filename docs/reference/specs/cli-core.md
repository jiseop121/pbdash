# CLI 핵심 구현 명세 (Track 1)

Track 1 `pbdash` CLI 구현의 단일 기준 문서.

## 1) Intent

Track 1 범위에서 `pbdash`의 성공 경로(TUI / REPL / one-shot / script)를 한 번에 읽히게 구현하고, 실패를 일관된 오류 형식과 종료 코드(`0/1/2/3`)로 고정한다.

## 2) Primary Flow

### 2.1 런타임 1차 흐름 (Main)

1. `argv`를 파싱해 `RunConfig`를 만든다
2. 모드/플래그 충돌을 검증한다
3. 실행 모드(`tui|repl|one-shot|script|ui-reserved`)를 결정한다
4. 모드 실행기 1개만 호출한다
5. 결과를 `stdout/stderr` 및 종료 코드로 매핑해 종료한다

### 2.2 모드별 성공/실패 흐름

#### A. TUI (`pbdash`)
1. 전면 TUI를 시작한다
2. DB 목록부터 탐색 흐름을 연다
3. 필요 시 superuser, collection, records, record detail 화면으로 이동한다
4. records 화면에서 `Enter`로 record detail 화면 진입
5. record detail 화면에서 `Esc`/`Backspace`로 records 화면으로 복귀, `y`로 현재 레코드 JSON clipboard 복사

#### B. REPL (`pbdash -repl`)
1. REPL 루프 시작
2. 입력 한 줄을 `Command`로 파싱
3. dispatcher로 명령 실행 후 결과 출력
4. `exit`/EOF 전까지 반복

#### C. One-shot (`pbdash -c "<command>"`)
1. `-c` 텍스트를 단일 명령으로 파싱
2. dispatcher로 1회 실행
3. 결과 출력 후 종료

#### D. Script (`pbdash <script-file>`)
1. UTF-8 파일을 읽고 줄 단위로 순회
2. 빈 줄/`#` 주석 줄 건너뜀
3. 명령 실행 오류가 발생해도 다음 줄 실행 계속(continue-on-error)
4. 각 실패 줄마다 `Error: Script failed at line <N>: <message>` 즉시 출력
5. `exit`/`quit`를 만나면 그 시점에서 script 실행 중단
6. 종료 코드는 세션에서 마지막으로 발생한 오류 코드(`1/2/3`); 오류 없으면 `0`

#### E. UI 예약 플래그 (`pbdash -ui`) - Track 1
1. `-ui`를 예약 플래그로 인식
2. 실제 UI 실행 없이 즉시 실패 처리
3. `stderr`에 `Error: Web UI is under development.` 출력
4. 종료 코드 `2`로 종료

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

- `cmd/pbdash/main.go`는 `app.Run(...)`만 호출한다
- 모드 분기/실행기 선택/종료 코드 결정은 `app.Run`이 단독 소유한다
- 실행기(Atoms) 간 직접 호출은 금지하고, 조합은 `app.Run`에서만 수행한다

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

## 6) 핵심 데이터 계약

### RunConfig

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

### ExecMode

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

### AppError

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

### 공통 규칙
- 모든 실행 경로(TUI / REPL / one-shot / script)는 같은 dispatcher를 공유한다
- PocketBase API는 GET만 허용한다
- 쓰기 동작 요청은 `ErrInvalidArgs`로 거절한다

### `db` / `superuser`
- `db add/list/remove`, `superuser add/list/remove` 지원
- 별칭 고유성 검증은 add 시점에서 수행
- 누락 인자/중복 별칭/형식 오류는 종료 코드 `2`

### `api` 조회
- 지원: `collections`, `collection`, `records`, `record`
- `records` 쿼리 옵션: `--page --per-page --sort --filter`
- 쿼리 값이 유효하지 않으면 종료 코드 `2`

### 출력 포맷
- 기본 `--format table`
- `table`: `stdout`에 ASCII 테이블 + 마지막 줄 `N rows`
- `csv|markdown`: `--out` 필수, 파일 저장 후 `stdout` 요약 1줄 출력
- 다건 조회 메타: `page`, `perPage`, `totalItems`, `totalPages`

## 8) 오류/종료 코드

- 오류 1행: `Error: <plain English message>`
- 오류 2행(선택): `Hint: <next action>`
- 종료 코드: `0`성공, `1`런타임, `2`인자/모드/미지원, `3`외부 의존 실패

## 9) 파일/패키지 배치

```
cmd/pbdash/main.go
internal/app/run.go, config.go, errors.go, exit_code.go
internal/cli/repl.go, script.go, dispatch.go, command_parser.go, formatters.go
internal/pocketbase/client.go, query.go
internal/storage/db_store.go, superuser_store.go
```

읽기 순서 원칙: public entry → orchestrator → mode executors → command atoms → utility
