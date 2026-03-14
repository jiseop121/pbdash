**2026-03-14**
- `docs/development/`, `docs/specs/`는 runbook/spec → `docs/reference/`로 이동; AGENTS.md 경로 포인터 함께 갱신한다
- `AGENTS.md`는 LEAVE: 최상위 에이전트 지시문이자 repo 내러티브를 포함하므로 이동하지 않는다
- `release-note-template.md`는 LEAVE: 인간 작성용 초안 템플릿으로 AI 컨텍스트 아님
- `.codex/skills/release-brew/SKILL.md`는 ARCHIVE: `make release-brew` deprecated(GoReleaser CI 대체), 내용이 현행 프로세스와 불일치하므로 DEPRECATED 마킹 후 제자리 보존
