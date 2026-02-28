# Release Inputs

## Required

- `version`: semantic version without `v` prefix (example: `0.1.1`)
- `source-repo-dir`: local checkout path of `multi-pocketbase-ui`
- `tap-repo-dir`: local checkout path of tap repo
- `tap-github-repo`: GitHub repo for tap release assets (`owner/homebrew-...`)

## Defaults

- formula name: `pocketbase-multiview`
- binary name: `pbmulti`

## Expected outputs

- GitHub release `v<version>` on tap repo with `darwin-arm64` and `darwin-amd64` tarballs
- Updated `Formula/pocketbase-multiview.rb`
- Successful install via `brew install pocketbase-multiview`
