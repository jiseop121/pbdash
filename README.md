# multi-pocketbase-ui

PocketBase Admin UI의 멀티 인스턴스 탐색/비교를 위한 로컬 도구 프로젝트.

## V0 요약

- 런타임: `pbmulti` CLI + `pbmulti -ui` 로컬 UI 서버
- 인증: `token`, `adminUser`는 메모리 세션만 사용(디스크 저장 금지)
- 저장: 인스턴스/뷰/워크스페이스 프리셋만 로컬 저장

## 문서 가이드

- 문서 권한/경계: `docs/docs-index.md`
- Track 1 (우선): `docs/cli-core-dev-spec.md`
- Track 2 (`-ui`): `docs/ui-mode-dev-spec.md`
- 제품 기능(PRD): `docs/ui-spec.md`
- 타입/상태 계약: `docs/spec-contracts.md`
- UI 시각 규칙: `docs/ui-design-spec.md`
- 통합 배포 로드맵: `docs/deployment-cli-brew-spec.md`
- 코드 구조 원칙: `docs/structure-first-project-design.md`

## 참고 경로

- CSS 토큰: `styles/tokens.css`
- 프리뷰: `preview/index.html`

## 로컬 E2E (실제 PocketBase)

아래 E2E는 임시 디렉터리/임시 포트에서 PocketBase를 실제로 띄운 뒤 `pbmulti` 전체 핵심 흐름을 스모크 검증한다.

- 실행 커버리지:
  - `version`, `help`
  - `db add/list/remove`
  - `superuser add/list/remove`
  - `api collections/collection/records/record`
  - `--format csv|markdown --out`
  - script 모드(`pbmulti <script-file>`)

### 사전 조건

- `pocketbase` 실행 파일이 PATH에 있어야 한다.
  - 기본값은 `pocketbase`
  - 다른 경로/이름이면 `POCKETBASE_BIN` 환경변수로 지정

### 실행

```bash
make e2e
```

```bash
POCKETBASE_BIN=/absolute/path/to/pocketbase make e2e
```

### 수동 PocketBase 실행(디버깅용)

```bash
make pocketbase-superuser PB_SUPERUSER_EMAIL=root@example.com PB_SUPERUSER_PASSWORD=pass123456
make pocketbase-serve PB_HTTP=127.0.0.1:8090
```
