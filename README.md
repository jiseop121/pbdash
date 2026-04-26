# PocketBase Dash

`pbdash`는 여러 PocketBase 인스턴스를 별칭 기반으로 관리하고, 컬렉션/레코드를 읽기 전용으로 조회하는 로컬 CLI 도구입니다.

## 설치

### Homebrew (공식)

```bash
brew tap jiseop121/pbdash https://github.com/jiseop121/homebrew-pbdash
brew install jiseop121/pbdash/pbdash
```

### 소스에서 설치 (보조)

사전 조건: Go 1.25+

```bash
go build -o pbdash ./cmd/pbdash
./pbdash -c "version"
```

또는 `go install`을 사용할 수 있습니다.

```bash
go install ./cmd/pbdash
pbdash -c "version"
```

## Quick Start

### 1) 버전 및 도움말 확인

```bash
pbdash -c "version"
pbdash -c "help"
```

### 2) PocketBase 인스턴스 등록

```bash
pbdash -c "db add --alias local --url http://127.0.0.1:8090"
pbdash -c "db list"
```

### 3) superuser 등록

```bash
pbdash -c "superuser add --db local --alias root --email root@example.com --password pass123456"
pbdash -c "superuser list --db local"
```

### 4) API 조회

```bash
pbdash -c "api collections --db local --superuser root"
pbdash -c "api records --db local --superuser root --collection posts --page 1 --per-page 20"
```

### 5) TUI/REPL 진입

기본 실행은 전면 TUI다.
탐색 흐름은 `DB 목록 -> (필요 시 superuser 선택) -> collections -> records table -> record detail` 순서다.
기본 탐색은 `j/k` 또는 화살표키로 이동한다.
`q`는 종료, `Esc`/`Backspace`는 이전 화면으로 돌아간다.
records 화면에서는 `Enter`로 별도 `record detail` 화면으로 들어간다.
`record detail` 화면에서는 `y`로 현재 레코드 JSON을 clipboard에 복사할 수 있다.
컬럼 선택 모달(`c`)에서는 `Space`로 컬럼을 토글하고 `Enter`로 적용, `Esc`로 취소한다.
필터 모달(`/`)에서는 `Enter`로 적용하고 `Esc`로 취소한다.

```bash
pbdash
```

기존 REPL이 필요하면 `-repl`로 연다.

```bash
pbdash -repl
pbdash> context use --db local --superuser root
pbdash> context save
pbdash> api records --collection posts
```

웹 UI 예약 옵션 `-ui`는 아직 개발중이다.

```bash
pbdash -ui
```

### 6) REPL 기본 컨텍스트 설정

```bash
pbdash -repl
pbdash> context use --db local --superuser root
pbdash> context save
pbdash> api records --collection posts
```

### 7) REPL/script 에러 처리 규칙

- `pbdash`(REPL)와 `pbdash <script-file>`는 명령 오류가 나도 다음 명령 실행을 계속합니다.
- 세션 종료 코드는 마지막 오류 코드(`1/2/3`)를 따르며, 오류가 없으면 `0`입니다.

## 출력 포맷

- 기본 포맷은 `table`입니다.
- `--format csv|markdown`을 사용하면 `--out <path>`가 필수입니다.
- `api records`는 TTY 환경에서 기본적으로 풀스크린 TUI(`--view auto`)로 표시됩니다.
- `--view table`로 텍스트 테이블 출력을 강제할 수 있습니다.

예시:

```bash
pbdash -c "api records --db local --superuser root --collection posts --format csv --out ./posts.csv"
```

## 개발/배포 문서

유지보수자용 개발 절차와 릴리스 순서는 아래 문서를 참고합니다.

- `docs/development/development-guide.md`
- `docs/development/release-guide.md`

## 참고

- 버그/제안은 GitHub Issues로 등록
- 보안 이슈는 공개 이슈 대신 비공개 채널로 제보 권장
