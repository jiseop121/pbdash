---
name: contract-smoke
description: Verify CLI contract behavior (exit codes, stdout/stderr separation, and key error messages) for pbmulti with deterministic smoke checks. Use when users ask for regression checks after CLI changes, before release, or after bug fixes in command parsing and mode handling.
---

# Contract Smoke

## Overview

Use this skill to run fast CLI contract regression checks without external services.
Prefer the bundled script to validate critical behavior consistently before push or release.

## Workflow

1. Run `scripts/run_contract_smoke.sh`.
2. Inspect failed cases and compare with [references/contract-cases.md](references/contract-cases.md).
3. Fix behavior and re-run until all checks pass.

## Run

```bash
./scripts/run_contract_smoke.sh --repo-dir <path>
```

## Guardrails

- Keep checks offline and deterministic.
- Validate both exit code and output channel expectations.
- Add a new case when fixing a user-facing contract bug.
