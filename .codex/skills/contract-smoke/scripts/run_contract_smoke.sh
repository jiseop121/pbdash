#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="$(pwd)"

usage() {
  cat <<USAGE
Usage: $(basename "$0") [--repo-dir <path>]
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo-dir)
      REPO_DIR="${2:-}"
      shift 2
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

if [[ ! -d "$REPO_DIR" ]]; then
  echo "repo dir not found: $REPO_DIR" >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "missing command: go" >&2
  exit 1
fi

WORKDIR="$(mktemp -d)"
cleanup() {
  rm -rf "$WORKDIR"
}
trap cleanup EXIT

BIN="$WORKDIR/pbmulti"
(
  cd "$REPO_DIR"
  go build -o "$BIN" ./cmd/pbmulti
)

EXPECTED_VERSION="$(awk -F'"' '/const Version = / {print $2; exit}' "$REPO_DIR/internal/app/run.go")"
if [[ -z "$EXPECTED_VERSION" ]]; then
  echo "cannot read expected version from internal/app/run.go" >&2
  exit 1
fi

FAIL_COUNT=0
PASS_COUNT=0
REPORT=""

run_case() {
  local name="$1"
  local cmd="$2"
  local expect_code="$3"
  local expect_stdout_contains="$4"
  local expect_stderr_contains="$5"

  local out_file="$WORKDIR/out.txt"
  local err_file="$WORKDIR/err.txt"

  set +e
  bash -lc "$cmd" >"$out_file" 2>"$err_file"
  local code=$?
  set -e

  local out_text
  local err_text
  out_text="$(cat "$out_file")"
  err_text="$(cat "$err_file")"

  local ok=1
  if [[ "$code" -ne "$expect_code" ]]; then
    ok=0
  fi
  if [[ -n "$expect_stdout_contains" && "$out_text" != *"$expect_stdout_contains"* ]]; then
    ok=0
  fi
  if [[ -n "$expect_stderr_contains" && "$err_text" != *"$expect_stderr_contains"* ]]; then
    ok=0
  fi

  if [[ "$ok" -eq 1 ]]; then
    PASS_COUNT=$((PASS_COUNT + 1))
    REPORT+="- ${name}: PASS"$'\n'
  else
    FAIL_COUNT=$((FAIL_COUNT + 1))
    REPORT+="- ${name}: FAIL (exit=${code}, expected=${expect_code})"$'\n'
    REPORT+="  stdout: ${out_text}"$'\n'
    REPORT+="  stderr: ${err_text}"$'\n'
  fi
}

run_case "one-shot version" "${BIN} -c \"version\"" 0 "$EXPECTED_VERSION" ""
run_case "ui reserved" "${BIN} -ui" 2 "" "UI mode is not available in Track 1"
run_case "unknown option" "${BIN} --unknown" 2 "" "Unknown option"
run_case "csv requires out" "${BIN} -c \"api collections --format csv\"" 2 "" "Missing required option"

OUT_FILE="$WORKDIR/out_sep.txt"
ERR_FILE="$WORKDIR/err_sep.txt"
set +e
"$BIN" -c "version" >"$OUT_FILE" 2>"$ERR_FILE"
code_ok=$?
set -e
if [[ "$code_ok" -eq 0 && -s "$OUT_FILE" && ! -s "$ERR_FILE" ]]; then
  PASS_COUNT=$((PASS_COUNT + 1))
  REPORT+="- stdout/stderr on success: PASS"$'\n'
else
  FAIL_COUNT=$((FAIL_COUNT + 1))
  REPORT+="- stdout/stderr on success: FAIL"$'\n'
fi

set +e
"$BIN" --unknown >"$OUT_FILE" 2>"$ERR_FILE"
code_fail=$?
set -e
if [[ "$code_fail" -ne 0 && ! -s "$OUT_FILE" && -s "$ERR_FILE" ]]; then
  PASS_COUNT=$((PASS_COUNT + 1))
  REPORT+="- stdout/stderr on failure: PASS"$'\n'
else
  FAIL_COUNT=$((FAIL_COUNT + 1))
  REPORT+="- stdout/stderr on failure: FAIL"$'\n'
fi

printf 'Contract Smoke Report\n'
printf -- '- repo: %s\n' "$REPO_DIR"
printf -- '- passed: %s\n' "$PASS_COUNT"
printf -- '- failed: %s\n\n' "$FAIL_COUNT"
printf 'Checks\n%s' "$REPORT"

if [[ "$FAIL_COUNT" -gt 0 ]]; then
  exit 1
fi
