# Development Guide

이 문서는 `PocketBase Dash`를 개발/검증할 때 필요한 실행 절차를 모아둔 가이드다.

## 문서 지도

- 개발 절차 문서 폴더: `docs/development`
- 릴리스 절차: `docs/development/release-guide.md`
- 공개 명세 문서 폴더: `docs/specs`
- 내부 문서 폴더: `docs/internal`
- 외부 의존성 인덱스: `docs/development/dependencies/README.md`
- PocketBase SDK 상세: `docs/development/dependencies/pocketbase-client.md`
- 문서 권한/경계: `docs/docs-index.md`
- Track 1 (CLI): `docs/cli-core-dev-spec.md`
- Track 1 구현 상세: `docs/specs/cli-core-implementation-spec.md`
- Track 2 (`-ui`): `docs/specs/ui-mode-dev-spec.md`
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

아래 E2E는 임시 디렉터리/임시 포트에서 PocketBase를 실제로 실행한 뒤 `pbdash` 핵심 흐름을 스모크 검증한다.

실행 커버리지:
- `version`, `help`
- `db add/list/remove`
- `superuser add/list/remove`
- `api collections/collection/records/record`
- `--format csv|markdown --out`
- script 모드(`pbdash <script-file>`)

사전 조건:
- 기본값으로 `make e2e`는 PocketBase CLI가 없으면 `.tmp/tools/pocketbase/<version>/pocketbase`에 자동 다운로드한다.
- 자동 다운로드한 zip은 PocketBase release의 `checksums.txt`와 대조해 SHA-256 검증 후에만 사용한다.
- 이미 설치된 다른 바이너리를 쓰려면 `POCKETBASE_BIN` 환경변수로 지정한다. 절대/상대 경로뿐 아니라 PATH에 있는 명령 이름도 사용할 수 있다.

실행:

```bash
make e2e
```

```bash
POCKETBASE_BIN=/absolute/path/to/pocketbase make e2e
```

필요하면 바이너리만 먼저 받아둘 수 있다.

```bash
make pocketbase-bin
```

## 수동 PocketBase 실행 (디버깅)

아래 타깃들도 기본값으로 PocketBase CLI를 자동 다운로드한 뒤 실행한다.

```bash
make pocketbase-superuser PB_SUPERUSER_EMAIL=root@example.com PB_SUPERUSER_PASSWORD=pass123456
make pocketbase-serve PB_HTTP=127.0.0.1:8090
```

짧은 별칭도 지원한다.

```bash
make pb-su PB_SUPERUSER_EMAIL=root@example.com PB_SUPERUSER_PASSWORD=pass123456
make pb-serve PB_HTTP=127.0.0.1:8090
```
