# Contract Cases

## Mandatory cases

- one-shot `version` succeeds with exit code `0`
- `-ui` returns exit code `2` and a Track 1 error message
- unknown option returns exit code `2` and invalid-arg error message
- `--format csv` without `--out` returns exit code `2`
- success writes to stdout and failure writes to stderr

## Add cases when

- exit code mapping changes
- command parser behavior changes
- output format contract changes
