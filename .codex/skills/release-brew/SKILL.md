---
name: release-brew
description: Build and publish macOS Homebrew release artifacts for pbdash, update Formula/pbdash.rb in the same repository, and verify installation. Use when users ask to release a new version, refresh the Homebrew formula SHA/URL, fix brew distribution, or run end-to-end brew delivery checks.
---

# Release Brew

## Workflow

Use this skill to publish a `pbdash` Homebrew release from this repository and verify installability.

1. Confirm the target version and repository state.
2. Create and push the release tag first with `make release-tag VERSION=<version>` if it does not already exist.
3. Run `make release-brew VERSION=<version>`.
4. Verify the release uploaded `pbdash-v<version>-darwin-arm64.tar.gz` and `pbdash-v<version>-darwin-amd64.tar.gz`.
5. Verify [Formula/pbdash.rb](/Users/hjs/Personal/multi-pocketbase-ui/Formula/pbdash.rb) was updated and the brew smoke check passed.

## Guardrails

- Keep release tag format as `v<version>`.
- This project uses a single repository for source, release assets, and Homebrew formula: `jiseop121/pbdash`.
- Keep the formula name and binary name as `pbdash`.
- Build from `./cmd/pbdash`.
- Keep artifact names as `pbdash-v<version>-darwin-<arch>.tar.gz`.
- Do not run the brew release step before the tag exists on GitHub.
- Fail on version mismatch in smoke checks.
