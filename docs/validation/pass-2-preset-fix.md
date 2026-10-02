# Independent default-preset rename fix

Pre-existing behavior: the preset editor renamed the collection item but did not
update config.toml's default_preset reference. Restart could select a fallback.

`fix/default-preset-rename`, from staging b06c056, validates the full collection,
updates the config reference and saves the presets, then replaces live values.
If the preset write fails, it compensates the config change; rollback errors
are reported. The draft remains editable on every failed save. This is the
existing Pass 1 pattern for separate atomic files, not a crash-proof multi-file
transaction; interruption between writes remains release-hardening debt.

All complete local checks passed: formatting/module consistency, Python (3),
actionlint, Go tests, race, vet, staticcheck, govulncheck (no vulnerabilities),
and build. Go 1.26.8 tests, amd64/arm64 release builds/checksums and isolated TUI
smoke also passed. Regression coverage in editor_default_test.go exercises
restart selection, duplicates, config write failure and preset write failure.

Release version: 1.1.0-beta.3; this independent promotion advances the planned
configuration-editor release to beta.4.
