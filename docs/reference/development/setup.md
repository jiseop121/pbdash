# Development Setup

`pbdash` 개발/검증 시 필요한 실행 절차.

## 로컬 테스트

```bash
make test
# 또는
go test ./...
```

## 로컬 E2E (실제 PocketBase)

임시 디렉터리/포트에서 PocketBase를 실제 실행 후 pbdash 핵심 흐름을 스모크 검증한다.

실행 커버리지: `version`, `help`, `db add/list/remove`, `superuser add/list/remove`, `api collections/collection/records/record`, `--format csv|markdown --out`, script 모드

사전 조건:
- `make e2e` 기본값으로 PocketBase CLI 없으면 `.tmp/tools/pocketbase/<version>/pocketbase`에 자동 다운로드
- 자동 다운로드 zip은 PocketBase release `checksums.txt`와 SHA-256 대조 후 사용
- 다른 바이너리를 쓰려면 `POCKETBASE_BIN` 환경변수로 지정

```bash
make e2e
POCKETBASE_BIN=/absolute/path/to/pocketbase make e2e
make pocketbase-bin   # 바이너리만 먼저 받을 때
```

## 수동 PocketBase 실행 (디버깅)

```bash
make pocketbase-superuser PB_SUPERUSER_EMAIL=root@example.com PB_SUPERUSER_PASSWORD=pass123456
make pocketbase-serve PB_HTTP=127.0.0.1:8090
# 짧은 별칭
make pb-su PB_SUPERUSER_EMAIL=root@example.com PB_SUPERUSER_PASSWORD=pass123456
make pb-serve PB_HTTP=127.0.0.1:8090
```
