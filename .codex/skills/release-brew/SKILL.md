---
name: release-brew
description: DEPRECATED — GoReleaser CI가 자동 처리하므로 더 이상 사용하지 않는다. `make release-brew`는 deprecated.
---

# Release Brew (DEPRECATED)

> **이 스킬은 더 이상 사용하지 않는다.**
> Homebrew Formula 갱신은 태그 푸시 후 GoReleaser CI(`.github/workflows/release.yml`)가 자동 처리한다.
> 릴리스 절차는 `docs/reference/development/release.md`를 참고한다.

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
