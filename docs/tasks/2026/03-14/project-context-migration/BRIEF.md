# project-context 마이그레이션

## Goal
기존 `docs/development/`, `docs/specs/` 문서들을 `docs/reference/` 레이아웃으로 이동하고, `docs/memory.md`를 신규 생성해 project-context 구조를 완성한다.

## Scope
- 읽기: `AGENTS.md`, `docs/development/**`, `docs/specs/**`, `.codex/skills/release-brew/SKILL.md`
- 쓰기: `docs/memory.md`, `docs/reference/**`, `AGENTS.md`(경로 업데이트), `.codex/skills/release-brew/SKILL.md`(아카이브 마킹)

## Audit Map

| source | target | action |
|--------|--------|--------|
| `docs/development/development-guide.md` | `docs/reference/development/setup.md` | REFERENCE (doc map 섹션 stale ref 제거) |
| `docs/development/release-guide.md` | `docs/reference/development/release.md` | REFERENCE |
| `docs/development/release-note-template.md` | (제자리) | LEAVE |
| `docs/development/dependencies/README.md` | `docs/reference/dependencies/overview.md` | REFERENCE |
| `docs/development/dependencies/pocketbase-client.md` | `docs/reference/dependencies/pocketbase-client.md` | REFERENCE |
| `docs/specs/cli-core-implementation-spec.md` | `docs/reference/specs/cli-core.md` | REFERENCE |
| `docs/specs/ui-mode-dev-spec.md` | `docs/reference/specs/ui-mode.md` | REFERENCE |
| `.codex/skills/release-brew/SKILL.md` | (제자리, 상단에 DEPRECATED 마킹) | ARCHIVE |
| `AGENTS.md` 경로 포인터 | 새 경로로 업데이트 | UPDATE |
| `docs/memory.md` | 신규 생성 | MEMORY |

## Current Output Snapshot
마이그레이션 완료. check_runtime_shape.py OK.
