#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<USAGE
Usage:
  $(basename "$0") \
    --version <semver> \
    --source-repo-dir <path> \
    --tap-repo-dir <path> \
    --tap-github-repo <owner/repo> \
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

github_http_url() {
  local remote_url="$1"
  if [[ "$remote_url" =~ ^git@github\.com:(.+)\.git$ ]]; then
    echo "https://github.com/${BASH_REMATCH[1]}"
    return
  fi
  if [[ "$remote_url" =~ ^git@github-[^:]+:(.+)\.git$ ]]; then
    echo "https://github.com/${BASH_REMATCH[1]}"
    return
  fi
  if [[ "$remote_url" =~ ^https://github\.com/(.+)\.git$ ]]; then
    echo "https://github.com/${BASH_REMATCH[1]}"
    return
  fi
  if [[ "$remote_url" =~ ^https://github\.com/.+ ]]; then
    echo "$remote_url"
    return
  fi
  echo ""
}

VERSION=""
SOURCE_REPO_DIR=""
TAP_REPO_DIR=""
TAP_GITHUB_REPO=""
FORMULA_NAME="pocketbase-multiview"
BINARY_NAME="pbmulti"
DRY_RUN=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      VERSION="${2:-}"
      shift 2
      ;;
    --source-repo-dir)
      SOURCE_REPO_DIR="${2:-}"
      shift 2
      ;;
    --tap-repo-dir)
      TAP_REPO_DIR="${2:-}"
      shift 2
      ;;
    --tap-github-repo)
      TAP_GITHUB_REPO="${2:-}"
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

if [[ -z "$VERSION" || -z "$SOURCE_REPO_DIR" || -z "$TAP_REPO_DIR" || -z "$TAP_GITHUB_REPO" ]]; then
  usage
  exit 1
fi

for cmd in go gh shasum git tar; do
  require_cmd "$cmd"
done
if [[ "$DRY_RUN" -eq 0 ]]; then
  require_cmd brew
fi

if [[ ! -d "$SOURCE_REPO_DIR" || ! -d "$TAP_REPO_DIR" ]]; then
  echo "source/tap repo path not found" >&2
  exit 1
fi

TAG="v${VERSION}"
FORMULA_CLASS="$(to_formula_class "$FORMULA_NAME")"

WORKDIR="$(mktemp -d)"
cleanup() {
  rm -rf "$WORKDIR"
}
trap cleanup EXIT

mkdir -p "$WORKDIR/arm64" "$WORKDIR/amd64"
ARTIFACT_ARM64="${BINARY_NAME}-v${VERSION}-darwin-arm64.tar.gz"
ARTIFACT_AMD64="${BINARY_NAME}-v${VERSION}-darwin-amd64.tar.gz"

echo "==> build binaries"
(
  cd "$SOURCE_REPO_DIR"
  GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o "$WORKDIR/arm64/${BINARY_NAME}" ./cmd/pbmulti
  GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o "$WORKDIR/amd64/${BINARY_NAME}" ./cmd/pbmulti
)

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

echo "==> artifacts"
echo "arm64 sha256: $SHA_ARM64"
echo "amd64 sha256: $SHA_AMD64"

if [[ "$DRY_RUN" -eq 0 ]]; then
  echo "==> publish release assets to $TAP_GITHUB_REPO:$TAG"
  if gh release view "$TAG" --repo "$TAP_GITHUB_REPO" >/dev/null 2>&1; then
    gh release upload "$TAG" \
      "$WORKDIR/${ARTIFACT_ARM64}" \
      "$WORKDIR/${ARTIFACT_AMD64}" \
      --repo "$TAP_GITHUB_REPO" \
      --clobber
  else
    gh release create "$TAG" \
      "$WORKDIR/${ARTIFACT_ARM64}" \
      "$WORKDIR/${ARTIFACT_AMD64}" \
      --repo "$TAP_GITHUB_REPO" \
      --title "$TAG" \
      --notes "${BINARY_NAME} Homebrew assets for ${TAG}."
  fi
else
  echo "==> dry-run: skip release upload"
fi

REMOTE_URL="$(git -C "$SOURCE_REPO_DIR" remote get-url origin 2>/dev/null || true)"
HOMEPAGE="$(github_http_url "$REMOTE_URL")"
if [[ -z "$HOMEPAGE" ]]; then
  HOMEPAGE="https://github.com/${TAP_GITHUB_REPO}"
fi

RELEASE_BASE_URL="https://github.com/${TAP_GITHUB_REPO}/releases/download/${TAG}"
FORMULA_PATH="$TAP_REPO_DIR/Formula/${FORMULA_NAME}.rb"

FORMULA_CONTENT=$(cat <<RUBY
class ${FORMULA_CLASS} < Formula
  desc "CLI tool for exploring multiple PocketBase instances"
  homepage "${HOMEPAGE}"
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
    assert_match version.to_s, shell_output("#{bin}/${BINARY_NAME} -c \"version\"")
  end
end
RUBY
)

if [[ "$DRY_RUN" -eq 0 ]]; then
  mkdir -p "$(dirname "$FORMULA_PATH")"
  printf '%s\n' "$FORMULA_CONTENT" > "$FORMULA_PATH"

  echo "==> commit formula"
  git -C "$TAP_REPO_DIR" add "$FORMULA_PATH"
  if git -C "$TAP_REPO_DIR" diff --cached --quiet; then
    echo "no formula changes"
  else
    git -C "$TAP_REPO_DIR" commit -m "chore(formula): release ${FORMULA_NAME} ${TAG}"
    git -C "$TAP_REPO_DIR" push origin HEAD
  fi
else
  echo "==> dry-run: skip formula write/commit"
fi

TAP_OWNER="${TAP_GITHUB_REPO%%/*}"
TAP_REPO_NAME="${TAP_GITHUB_REPO##*/}"
TAP_NAME_PART="$TAP_REPO_NAME"
if [[ "$TAP_REPO_NAME" == homebrew-* ]]; then
  TAP_NAME_PART="${TAP_REPO_NAME#homebrew-}"
fi
TAP_NAME="${TAP_OWNER}/${TAP_NAME_PART}"

if [[ "$DRY_RUN" -eq 0 ]]; then
  echo "==> brew smoke"
  brew untap "$TAP_NAME" >/dev/null 2>&1 || true
  brew tap "$TAP_NAME"
  brew uninstall --force "$FORMULA_NAME" >/dev/null 2>&1 || true
  brew install "$FORMULA_NAME"

  INSTALLED_VERSION="$(${BINARY_NAME} -c "version" | tr -d '\n')"
  if [[ "$INSTALLED_VERSION" != "$VERSION" ]]; then
    echo "version mismatch: expected=${VERSION} got=${INSTALLED_VERSION}" >&2
    exit 1
  fi
  echo "smoke passed: ${BINARY_NAME} version ${INSTALLED_VERSION}"
else
  echo "==> dry-run: skip brew install smoke"
fi

cat <<DONE
Done.
- tag: ${TAG}
- formula: ${FORMULA_PATH}
- tap: ${TAP_NAME}
DONE
