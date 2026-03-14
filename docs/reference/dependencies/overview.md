# Dependencies Guide

`pbdash`가 직접 사용하는 외부 의존성의 역할, 사용 맥락, 수정 시 확인 포인트.

## 읽는 순서
- PocketBase 연동 수정 또는 인증/조회 흐름을 봐야 하면 `docs/reference/dependencies/pocketbase-client.md`를 먼저 본다
- 그 외 터미널 UI, REPL, 테스트 보조 라이브러리는 이 문서에서 개요를 확인한다

## 직접 사용하는 주요 의존성

### `github.com/rivo/tview`
- 목적: 터미널 기반 UI(TUI) 구성
- 사용 위치: `internal/cli/records_tui.go`
- 현재 사용 방식
  - `Application`, `Pages`, `Flex`, `Table`, `TextView` 중심으로 화면을 구성한다
  - DB 목록, superuser 선택, collection 목록, records 테이블, record detail 화면을 한 개의 navigator 흐름으로 조합한다
  - 상세 패널, 상태바, 도움말 라인을 모두 `tview` primitive로 렌더링한다
  - 폼/모달 입력은 공통 포커스 이동 규칙과 모달별 단축키(`Enter` 제출, `Esc` 취소, 컬럼 선택은 `Space` 토글)를 함께 사용한다
- 주의사항
  - 레이아웃/포커스 처리 변경은 키 입력 동작과 함께 깨지기 쉽다
  - `setupViews()`와 `handleKey()`는 항상 같이 검토한다
  - `installFormArrowNavigation`, `installFormArrowNavigationWithClose`, `installSubmitCancelNavigation` 같은 모달 입력 규칙 헬퍼도 함께 확인한다
  - 업그레이드 후에는 records TUI 관련 테스트를 먼저 확인한다

### `github.com/gdamore/tcell/v2`
- 목적: 터미널 키 입력 및 스타일 처리
- 사용 위치: `internal/cli/records_tui.go`, `internal/cli/records_tui_test.go`
- 현재 사용 방식
  - 방향키, Enter, Esc, Backspace 등 이벤트 처리를 `tcell.EventKey` 기준으로 분기한다
  - 선택 행 스타일과 같은 기본 TUI 색상도 `tcell.Style`로 설정한다
  - record detail 화면의 clipboard 복사는 `tcell.Screen.SetClipboard()`로 처리한다 (OSC52)
  - records TUI 렌더링 회귀는 `SimulationScreen` 기반 테스트에서 실제 키 주입과 화면 텍스트 일부를 검증한다
- 주의사항
  - 키 코드 또는 입력 처리 규칙이 바뀌면 TUI 탐색성이 바로 깨질 수 있다
  - `j/k` 같은 단축키 지원 여부와 방향키 동작은 테스트와 함께 유지한다

### `github.com/chzyer/readline`
- 목적: 인터랙티브 REPL 입력, 히스토리, 자동완성
- 사용 위치: `internal/cli/repl.go`
- 현재 사용 방식
  - TTY 환경일 때만 `readline` 기반 REPL을 사용한다
  - 히스토리 파일, 인터럽트 처리, 동적 자동완성 연결을 담당한다
  - 비-TTY 환경에서는 사용하지 않고 scanner fallback으로 내려간다
- 주의사항
  - REPL UX 변경 시 `runReadlineREPL()`과 `runScannerREPL()`의 동작 차이가 불필요하게 벌어지지 않게 유지한다
  - 자동완성은 `dynamicCompleter`를 통해 붙어 있으므로, command parser 변경 시 같이 검토한다

### `golang.org/x/term`
- 목적: 터미널 여부(TTY) 판정
- 사용 위치: `internal/cli/repl.go`
- 현재 사용 방식
  - `IsTTY()`에서 stdin/stdout이 둘 다 terminal인지 확인한다
  - 이 결과로 `readline` REPL 사용 가능 여부를 결정한다
- 주의사항
  - TTY 판정 로직이 바뀌면 CI, 파이프 입력, script 모드에서 REPL 동작이 달라질 수 있다

### `github.com/stretchr/testify`
- 목적: 테스트 assertion 보조
- 사용 위치: `*_test.go`
- 현재 사용 방식: `assert`, `require`만 사용; production code에는 사용하지 않는다
- 주의사항: 테스트 표현을 간결하게 만드는 용도; 도메인 로직을 감추는 과한 helper 추상화는 피한다

## 코드에서 먼저 볼 위치
- PocketBase 인증/조회 흐름: `internal/pocketbase/client.go`
- PocketBase 응답 후처리 및 endpoint 조립: `internal/pocketbase/query.go`
- REPL 입력 흐름: `internal/cli/repl.go`
- TUI 탐색 흐름: `internal/cli/records_tui.go`
- TUI 회귀 테스트: `internal/cli/records_tui_test.go`

## 의존성 변경 시 기본 체크리스트
1. `go test ./...`를 실행한다
2. REPL 관련 의존성을 건드렸다면 `pbdash -repl` 기본 흐름을 확인한다
3. TUI 관련 의존성을 건드렸다면 기본 TUI 진입과 키 입력 동작을 확인한다
4. PocketBase SDK 관련 변경이라면 `docs/reference/dependencies/pocketbase-client.md` 기준으로 인증, collections 조회, records 조회를 확인한다
5. 출력 형식이나 에러 메시지 계약이 바뀌지 않았는지 확인한다
