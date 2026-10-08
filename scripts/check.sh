#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ -n "${GOBIN:-}" ]]; then export PATH="$GOBIN:$PATH"; fi
unformatted=$(gofmt -l .)
if [[ -n "$unformatted" ]]; then printf 'Run gofmt on:\n%s\n' "$unformatted"; exit 1; fi
# Compare copies: this works before a local commit as well as in CI.
module_snapshot=$(mktemp -d)
trap 'rm -rf "$module_snapshot"' EXIT
cp go.mod go.sum "$module_snapshot/"
go mod tidy
diff -u "$module_snapshot/go.mod" go.mod
diff -u "$module_snapshot/go.sum" go.sum
go mod verify
python3 -m unittest discover -s scripts -p 'test_*.py'
python3 scripts/privacy.py --tracked
actionlint
python3 scripts/version.py "$(cat VERSION)"
go test ./...
go test -race ./...
go vet ./...
staticcheck ./...
govulncheck ./...
go build ./...
