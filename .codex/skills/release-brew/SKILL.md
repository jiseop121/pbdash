---
name: release-brew
description: Build and publish macOS Homebrew release artifacts for pbmulti, update the tap formula, and verify installation. Use when users ask to release a new version, refresh formula SHA/URL, fix Homebrew distribution, or run end-to-end brew delivery checks.
---

# Release Brew

## Overview

Use this skill to execute a deterministic Homebrew release flow for `pbmulti`.
Prefer the bundled script so versioning, artifact naming, SHA updates, and brew install checks stay consistent.

## Workflow

1. Confirm release inputs.
2. Run `scripts/release_brew.sh` with explicit paths and version.
3. Verify script output for release URL, formula update, and brew smoke pass.
4. If needed, inspect [references/release-inputs.md](references/release-inputs.md) for naming and assumptions.

## Run

```bash
./scripts/release_brew.sh \
  --version <semver> \
  --source-repo-dir <path-to-multi-pocketbase-ui> \
  --tap-repo-dir <path-to-homebrew-repo> \
  --tap-github-repo <owner/homebrew-repo>
```

Use `--dry-run` first when validating changes without publishing.

## Guardrails

- Keep release tag format as `v<version>`.
- Keep artifact names as `<binary>-v<version>-darwin-<arch>.tar.gz`.
- Publish binaries to the tap repository release to avoid private source build issues.
- Fail on version mismatch in smoke checks.
