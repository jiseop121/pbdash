---
name: pr-gate
description: Run merge-readiness quality gates for this Go project and produce a concise pass/fail report. Use when users request pre-merge checks, CI parity validation, bug-risk scanning before push, or a quick go/no-go decision for a PR.
---

# PR Gate

## Overview

Use this skill to run deterministic local gates and summarize blockers.
Prefer the bundled script to keep checks and report format stable across reviews.

## Workflow

1. Run `scripts/run_pr_gate.sh` in the target repository.
2. Read the report and focus on failed checks first.
3. Use [references/gate-policy.md](references/gate-policy.md) to classify merge readiness.

## Run

```bash
./scripts/run_pr_gate.sh --repo-dir <path>
```

Optional output file:

```bash
./scripts/run_pr_gate.sh --repo-dir <path> --out /tmp/pr-gate.md
```

## Guardrails

- Treat any failed gate as `Request changes`.
- Include failing command output in review comments.
- Do not skip race test unless explicitly requested.
