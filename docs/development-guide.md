# Development Guide

이 문서는 `multi-pocketbase-ui`를 개발/검증할 때 필요한 실행 절차를 모아둔 가이드다.

## 문서 지도

- 문서 권한/경계: `docs/docs-index.md`
- Track 1 (CLI): `docs/cli-core-dev-spec.md`
- Track 1 구현 상세: `docs/cli-core-implementation-spec.md`
- Track 2 (`-ui`): `docs/ui-mode-dev-spec.md`
- 제품 기능(PRD): `docs/ui-spec.md`
- 타입/상태 계약: `docs/spec-contracts.md`
- UI 시각 규칙: `docs/ui-design-spec.md`
- 통합 배포 로드맵: `docs/deployment-cli-brew-spec.md`
- 코드 구조 원칙: `docs/structure-first-project-design.md`

## 로컬 테스트

```bash
make test
```

또는:

```bash
go test ./...
```

## 로컬 E2E (실제 PocketBase)

아래 E2E는 임시 디렉터리/임시 포트에서 PocketBase를 실제로 실행한 뒤 `pbmulti` 핵심 흐름을 스모크 검증한다.

실행 커버리지:
- `version`, `help`
- `db add/list/remove`
- `superuser add/list/remove`
- `api collections/collection/records/record`
- `--format csv|markdown --out`
- script 모드(`pbmulti <script-file>`)

사전 조건:
- `pocketbase` 실행 파일이 PATH에 있어야 한다.
- 다른 경로/이름이면 `POCKETBASE_BIN` 환경변수로 지정한다.

실행:

```bash
make e2e
```

```bash
POCKETBASE_BIN=/absolute/path/to/pocketbase make e2e
```

## 수동 PocketBase 실행 (디버깅)

```bash
make pocketbase-superuser PB_SUPERUSER_EMAIL=root@example.com PB_SUPERUSER_PASSWORD=pass123456
make pocketbase-serve PB_HTTP=127.0.0.1:8090
```
