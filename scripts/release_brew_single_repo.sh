#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<USAGE
Usage:
  $(basename "$0") \
    --version <semver> \
    --github-repo <owner/repo> \
    [--formula-name <name>] \
    [--binary-name <name>] \
    [--dry-run]
USAGE
}

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "missing command: $cmd" >&2
    exit 1
  fi
}

to_formula_class() {
  local input="$1"
  awk -F- '{for(i=1;i<=NF;i++){printf toupper(substr($i,1,1)) substr($i,2)}printf "\n"}' <<<"$input"
}

VERSION=""
GITHUB_REPO=""
FORMULA_NAME="pocketbase-multiview"
BINARY_NAME="pbmulti"
DRY_RUN=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      VERSION="${2:-}"
      shift 2
      ;;
    --github-repo)
      GITHUB_REPO="${2:-}"
      shift 2
      ;;
    --formula-name)
      FORMULA_NAME="${2:-}"
      shift 2
      ;;
    --binary-name)
      BINARY_NAME="${2:-}"
      shift 2
      ;;
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown arg: $1" >&2
      usage
      exit 1
      ;;
  esac
done

if [[ -z "$VERSION" || -z "$GITHUB_REPO" ]]; then
  usage
  exit 1
fi

if ! [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "version must be semantic version without v prefix (example: 0.2.1)" >&2
  exit 1
fi

for cmd in go gh shasum git tar; do
  require_cmd "$cmd"
done
if [[ "$DRY_RUN" -eq 0 ]]; then
  require_cmd brew
fi

TAG="v${VERSION}"
FORMULA_CLASS="$(to_formula_class "$FORMULA_NAME")"

if [[ "$DRY_RUN" -eq 0 ]]; then
  if ! gh release view "$TAG" --repo "$GITHUB_REPO" >/dev/null 2>&1; then
    echo "release tag not found: ${TAG} (create/push the tag first)" >&2
    exit 1
  fi
fi

WORKDIR="$(mktemp -d)"
cleanup() {
  rm -rf "$WORKDIR"
}
trap cleanup EXIT

mkdir -p "$WORKDIR/arm64" "$WORKDIR/amd64"
ARTIFACT_ARM64="${BINARY_NAME}-v${VERSION}-darwin-arm64.tar.gz"
ARTIFACT_AMD64="${BINARY_NAME}-v${VERSION}-darwin-amd64.tar.gz"

echo "==> build binaries"
BUILD_FLAGS=(-trimpath -ldflags "-s -w -buildid=")
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build "${BUILD_FLAGS[@]}" -o "$WORKDIR/arm64/${BINARY_NAME}" ./cmd/pbmulti
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build "${BUILD_FLAGS[@]}" -o "$WORKDIR/amd64/${BINARY_NAME}" ./cmd/pbmulti
touch -t 197001010000 "$WORKDIR/arm64/${BINARY_NAME}" "$WORKDIR/amd64/${BINARY_NAME}"

(
  cd "$WORKDIR/arm64"
  tar -czf "$WORKDIR/${ARTIFACT_ARM64}" "$BINARY_NAME"
)
(
  cd "$WORKDIR/amd64"
  tar -czf "$WORKDIR/${ARTIFACT_AMD64}" "$BINARY_NAME"
)

SHA_ARM64="$(shasum -a 256 "$WORKDIR/${ARTIFACT_ARM64}" | awk '{print $1}')"
SHA_AMD64="$(shasum -a 256 "$WORKDIR/${ARTIFACT_AMD64}" | awk '{print $1}')"
RELEASE_BASE_URL="https://github.com/${GITHUB_REPO}/releases/download/${TAG}"
FORMULA_PATH="Formula/${FORMULA_NAME}.rb"

FORMULA_CONTENT=$(cat <<RUBY
class ${FORMULA_CLASS} < Formula
  desc "CLI tool for exploring multiple PocketBase instances"
  homepage "https://github.com/${GITHUB_REPO}"
  version "${VERSION}"

  on_macos do
    if Hardware::CPU.arm?
      url "${RELEASE_BASE_URL}/${ARTIFACT_ARM64}"
      sha256 "${SHA_ARM64}"
    else
      url "${RELEASE_BASE_URL}/${ARTIFACT_AMD64}"
      sha256 "${SHA_AMD64}"
    end
  end

  def install
    bin.install "${BINARY_NAME}"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/${BINARY_NAME} -c \\"version\\"")
  end
end
RUBY
)

if [[ "$DRY_RUN" -eq 0 ]]; then
  echo "==> upload release assets to ${GITHUB_REPO}:${TAG}"
  gh release upload "$TAG" \
    "$WORKDIR/${ARTIFACT_ARM64}" \
    "$WORKDIR/${ARTIFACT_AMD64}" \
    --repo "$GITHUB_REPO" \
    --clobber

  echo "==> update formula ${FORMULA_PATH}"
  mkdir -p "$(dirname "$FORMULA_PATH")"
  printf '%s\n' "$FORMULA_CONTENT" > "$FORMULA_PATH"

  git add "$FORMULA_PATH"
  if git diff --cached --quiet; then
    echo "no formula changes"
  else
    git commit -m "chore(formula): release ${FORMULA_NAME} ${TAG}"
    git push origin HEAD
  fi

  echo "==> brew smoke"
  TAP_ALIAS="${GITHUB_REPO%%/*}/pocketbase-multiview"
  TAP_REMOTE="$(git remote get-url origin 2>/dev/null || true)"
  if [[ -z "$TAP_REMOTE" ]]; then
    TAP_REMOTE="https://github.com/${GITHUB_REPO}.git"
  fi
  brew untap "$TAP_ALIAS" >/dev/null 2>&1 || true
  brew tap "$TAP_ALIAS" "$TAP_REMOTE"
  brew uninstall --force "$FORMULA_NAME" >/dev/null 2>&1 || true
  brew install "$TAP_ALIAS/$FORMULA_NAME"

  INSTALLED_VERSION="$(${BINARY_NAME} -c "version" | tr -d '\n')"
  if [[ "$INSTALLED_VERSION" != "$VERSION" ]]; then
    echo "version mismatch: expected=${VERSION} got=${INSTALLED_VERSION}" >&2
    exit 1
  fi
  echo "smoke passed: ${BINARY_NAME} version ${INSTALLED_VERSION}"
else
  echo "==> dry-run: skip upload/formula-write/smoke"
  echo "$FORMULA_CONTENT"
fi

cat <<DONE
Done.
- tag: ${TAG}
- formula: ${FORMULA_PATH}
- assets:
  - ${ARTIFACT_ARM64}
  - ${ARTIFACT_AMD64}
DONE
