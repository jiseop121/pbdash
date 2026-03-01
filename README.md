# multi-pocketbase-ui

`pbmulti`는 여러 PocketBase 인스턴스를 별칭 기반으로 관리하고, 컬렉션/레코드를 읽기 전용으로 조회하는 로컬 CLI 도구입니다.

## 설치

### Homebrew (공식)

```bash
brew tap jiseop121/pocketbase-multiview
brew install pocketbase-multiview
```

선택적으로 formula 전체 경로로도 설치할 수 있습니다.

```bash
brew install jiseop121/pocketbase-multiview/pocketbase-multiview
```

### 소스에서 설치 (보조)

사전 조건: Go 1.23+

```bash
go build -o pbmulti ./cmd/pbmulti
./pbmulti -c "version"
```

또는 `go install`을 사용할 수 있습니다.

```bash
go install ./cmd/pbmulti
pbmulti -c "version"
```

## Quick Start

### 1) 버전 및 도움말 확인

```bash
pbmulti -c "version"
pbmulti -c "help"
```

### 2) PocketBase 인스턴스 등록

```bash
pbmulti -c "db add --alias local --url http://127.0.0.1:8090"
pbmulti -c "db list"
```

### 3) superuser 등록

```bash
pbmulti -c "superuser add --db local --alias root --email root@example.com --password pass123456"
pbmulti -c "superuser list --db local"
```

### 4) API 조회

```bash
pbmulti -c "api collections --db local --superuser root"
pbmulti -c "api records --db local --superuser root --collection posts --page 1 --per-page 20"
```

## 출력 포맷

- 기본 포맷은 `table`입니다.
- `--format csv|markdown`을 사용하면 `--out <path>`가 필수입니다.

예시:

```bash
pbmulti -c "api records --db local --superuser root --collection posts --format csv --out ./posts.csv"
```

## 추가 문서

- 개발/테스트 가이드: `docs/development-guide.md`
- 문서 권한/우선순위: `docs/docs-index.md`
- CLI 계약(Track 1): `docs/cli-core-dev-spec.md`
- UI 모드 계약(Track 2): `docs/ui-mode-dev-spec.md`
