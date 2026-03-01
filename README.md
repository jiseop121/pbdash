# multi-pocketbase-ui

`pbmulti`는 여러 PocketBase 인스턴스를 별칭 기반으로 관리하고, 컬렉션/레코드를 읽기 전용으로 조회하는 로컬 CLI 도구입니다.

## 설치

### Homebrew (공식)

```bash
brew tap jiseop121/pocketbase-multiview https://github.com/jiseop121/multi-pocketbase-ui
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

## Release 자동화

태그를 푸시하면 GitHub Release 본문은 자동 생성된다.

```bash
make release-tag VERSION=0.2.1
```

동작:
- `go test ./...` 실행
- `v0.2.1` 태그 생성 및 원격 푸시
- GitHub Actions(`.github/workflows/release.yml`)가 Release를 생성/갱신하고 변경사항 노트를 자동 작성

Homebrew 배포 파일(아티팩트 + Formula) 갱신:

```bash
make release-brew VERSION=0.2.1
```

동작:
- `darwin-arm64`, `darwin-amd64` 바이너리 tar.gz 빌드
- 현재 레포 Release(`v0.2.1`)에 아티팩트 업로드
- `Formula/pocketbase-multiview.rb` SHA/URL 갱신 후 커밋/푸시
- Homebrew 설치 스모크 테스트

## 참고

- 버그/제안은 GitHub Issues로 등록
- 보안 이슈는 공개 이슈 대신 비공개 채널로 제보 권장
