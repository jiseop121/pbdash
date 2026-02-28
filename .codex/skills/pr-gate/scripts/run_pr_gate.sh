#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="$(pwd)"
OUT_FILE=""
SKIP_RACE=0

usage() {
  cat <<USAGE
Usage: $(basename "$0") [--repo-dir <path>] [--out <file>] [--skip-race]
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo-dir)
      REPO_DIR="${2:-}"
      shift 2
      ;;
    --out)
      OUT_FILE="${2:-}"
      shift 2
      ;;
    --skip-race)
      SKIP_RACE=1
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

if [[ ! -d "$REPO_DIR" ]]; then
  echo "repo dir not found: $REPO_DIR" >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "missing command: go" >&2
  exit 1
fi

declare -a CHECK_NAMES
declare -a CHECK_CMDS
CHECK_NAMES+=("Unit tests")
CHECK_CMDS+=("go test ./...")
if [[ "$SKIP_RACE" -eq 0 ]]; then
  CHECK_NAMES+=("Race tests")
  CHECK_CMDS+=("go test -race ./...")
fi
CHECK_NAMES+=("Go vet")
CHECK_CMDS+=("go vet ./...")

PASS_COUNT=0
FAIL_COUNT=0
REPORT=""

for idx in "${!CHECK_NAMES[@]}"; do
  name="${CHECK_NAMES[$idx]}"
  cmd="${CHECK_CMDS[$idx]}"
  out_file="$(mktemp)"
  err_file="$(mktemp)"

  set +e
  (
    cd "$REPO_DIR"
    bash -lc "$cmd"
  ) >"$out_file" 2>"$err_file"
  code=$?
  set -e

  if [[ "$code" -eq 0 ]]; then
    status="PASS"
    PASS_COUNT=$((PASS_COUNT + 1))
  else
    status="FAIL"
    FAIL_COUNT=$((FAIL_COUNT + 1))
  fi

  REPORT+="- ${name}: ${status} (exit=${code})"$'\n'
  if [[ "$status" == "FAIL" ]]; then
    REPORT+="  command: ${cmd}"$'\n'
    REPORT+="  output:"$'\n'
    while IFS= read -r line; do
      REPORT+="    ${line}"$'\n'
    done < <(tail -n 40 "$out_file" "$err_file" 2>/dev/null)
  fi

  rm -f "$out_file" "$err_file"
done

TIMESTAMP="$(date '+%Y-%m-%d %H:%M:%S %z')"
SUMMARY=$(cat <<TXT
PR Gate Report
- repo: ${REPO_DIR}
- time: ${TIMESTAMP}
- passed: ${PASS_COUNT}
- failed: ${FAIL_COUNT}

Checks
${REPORT}
Decision
TXT
)

if [[ "$FAIL_COUNT" -eq 0 ]]; then
  SUMMARY+=$'\n- Merge-ready (for automated gates).'
else
  SUMMARY+=$'\n- Not merge-ready. Fix failing checks before merge.'
fi

if [[ -n "$OUT_FILE" ]]; then
  printf '%s\n' "$SUMMARY" > "$OUT_FILE"
fi

printf '%s\n' "$SUMMARY"

if [[ "$FAIL_COUNT" -gt 0 ]]; then
  exit 1
fi
