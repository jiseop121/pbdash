# Gate Policy

## Required checks

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`

## Decision

- All pass: merge-ready from test/lint perspective
- Any fail: not merge-ready

## Reporting

- Show each check with PASS/FAIL
- Include stderr/stdout snippet for failures
- End with explicit recommendation
