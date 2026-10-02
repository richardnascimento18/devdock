# Pass 2A local validation

Implementation: `b75fa03`; release preparation: `5776eb5`.
Environment: Linux amd64, Go 1.27.1, minimum Go 1.26.8.

| Check | Result |
| --- | --- |
| gofmt, diff whitespace | Passed |
| go mod tidy consistency, module verification | Passed |
| go test ./... | Passed (86 top-level tests, 172 including subtests) |
| go test -race ./... | Passed |
| go vet ./... | Passed |
| staticcheck ./... | Passed (pinned v0.8.1) |
| govulncheck ./... | Passed; no vulnerabilities found (v1.8.0) |
| go build ./... | Passed |
| actionlint | Passed (v1.7.12; ShellCheck available) |
| Python unittest discovery | Passed, 3 tests |
| Go 1.26.8 tests | Passed |
| Linux amd64/arm64 release builds | Passed |
| Local SHA256SUMS | Both assets passed |
| Built amd64 --version | beta.2, commit 5776eb5 |
| Isolated TUI smoke | Passed with embedded test public OAuth ID |

Aggregate statement coverage: 49.4% (whole application, including TUI). Deep
scanner/model behavior has targeted regression tests; coverage is diagnostic,
not a claim of exhaustive interaction coverage. Full transient logs remain in
`/tmp/devdock-pass2a-*` and are not committed.

Three-iteration scanner benchmarks on AMD Ryzen 7 5700G, Linux amd64 (7.18 ms wide, 2.75 ms deep,
23.66 ms mixed in the final short sample): 1,000 wide projects, a 200-level chain, and
1,000 projects under 20 groups. No timing assertion is imposed on unit tests.
The scan regression counts one inspection per candidate at depths 25, 50 and
200. The navigation regression relocates the physical root and verifies that
collapse, rendering, selection and destination generation still use snapshots.

Sandbox-only attempts initially failed on read-only Go/staticcheck caches.
Writable temporary caches and network-enabled validation resolved those
environment restrictions. Staticcheck also caught one obsolete style and an
unused test variable; both were removed before the successful complete check.

The focused `fix/deep-location-disambiguation` correction was validated with
the same full local suite, Go 1.26.8 tests, both release builds/checksums and
isolated TUI smoke. It adds regressions for identical truncated breadcrumbs,
same-basename roots, complete long confirmation text, and rejection of group
creation inside a discovered project's source tree. Display hashes are shared
with tmux's existing path discriminator; session names are unchanged.
