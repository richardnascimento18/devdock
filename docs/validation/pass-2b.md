# Pass 2B: configuration editor

Branched from staging 315c972 after the verified beta.2 recursive model and
beta.3 independent default-preset rename fix. Release version: 1.1.0-beta.4.

The existing editor now has a Settings tab. It edits default_preset, validates
that a nonempty value names an existing preset, and reuses root addition/removal
and associated state cleanup. Credentials are excluded. Config.Commit clones,
validates the entire config, skips unchanged saves, persists atomically, then
returns the replacement. Failed saves retain drafts and live/disk config.

All local checks passed on Go 1.27.1: gofmt, tidy consistency/module verification,
Python tests (3), actionlint, go test, race, vet, staticcheck, govulncheck (no
vulnerabilities), and build. Minimum-supported Go 1.26.8 tests passed. Both
Linux release builds passed checksums, and the isolated amd64 TUI smoke exercised
the Settings tab and verified the saved default preset. Hosted verification now
runs this same smoke on both supported Go versions.

Final local measurement: 94 top-level tests / 200 including subtests; aggregate
statement coverage 51.4%. editor_config_test.go covers valid/blank defaults,
unknown presets, invalid/duplicate roots, failed writes, cancellation, no-op
saves, mode/navigation transitions, secret exclusion, root add/remove/cancel/
failure, deliberate state pruning, and subsequent edits retaining changed roots.
internal/config/config_test.go covers detached proposals and unchanged saves
with nil, empty, and populated roots. Existing preset/template regression tests
continue enforcing collection-wide validation, atomic writes and drafts.

The first extended smoke attempt batched the vi insert-mode key with its text;
the harness now sends the mode transition separately and parses TOML rather
than relying on a quoting style. All final checks passed. Raw local logs and
release verification evidence are kept outside the repository for review.
