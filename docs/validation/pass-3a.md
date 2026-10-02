# Pass 3A — UI foundation

Branch: `refactor/ui-foundation`, based on verified staging
`e7033c27e896420c6f4bff65596b0faa44dce5b5`. Starting production:
`d2d8ced198c290b768bc18dead004c2ddca6fa60`; beta.5 tag resolves to that SHA.
Production is default, ancestor of staging, and has the same tree. Only permanent
remote branches existed. Both rulesets were active, required PR/check workflows,
had no bypass actors, and automatic merged branch deletion was enabled.

## Scope and architecture

See [TUI architecture](../architecture/tui.md). Adds one presentation package for
semantic foreground roles, responsive allocation, focus, header/footer, modal
and cell-aware text primitives. The root view dispatches screens; dashboard frame
and project delegate are separate. Existing domain, snapshot, operation, OAuth,
tmux and persistence code remain authoritative.

Replaces explicit backgrounds in all existing screens, language badges and
selections; overrides the Bubbles list title and inverse-video input cursor.
Primary text inherits terminal foreground. Other roles use adaptive foregrounds.
Selection uses `>` and emphasis. The giant wordmark becomes a compact title.
Tab/Shift+Tab focus workspace/projects; `{`/`}` switch roots directly. Help scrolls.
Sizing derives 98/66-column breakpoints from minimum pane widths and gutters,
with a 24×8 minimum; dashboard composition follows in 3B.

## Validation

- Full `scripts/check.sh`: formatting, tidy consistency, module verification,
  Python tests (8), actionlint, version policy, Go tests/race/vet/staticcheck,
  govulncheck (no vulnerabilities), build: passed.
- Exact Go 1.26.0 `go test ./...`: passed.
- Go suite: 110 top-level tests, 256 tests/subtests passed across 12 packages.
- 18 paired ANSI/plain-text golden fixtures; real styled output checked across
  light/dark defaults and truecolor, 256, basic ANSI and no-color profiles.
- Style AST regression rejects backgrounds and inverse video. The SGR detector
  rejects explicit background codes and allows resets/default-background and
  foreground payloads whose RGB components happen to equal background codes.
- Resize/focus/help/Escape interaction tests: passed. Allocation tests caught and
  corrected an inspector over-allocation at 100 columns.
- Linux amd64/arm64 static ELF builds and SHA256SUMS: passed.
- Isolated PTY smoke: startup, project, help, six resize sizes, root focus,
  favorites, editor, Settings persistence, OAuth entry/cancel and quit: passed.
  Actual captured terminal bytes contain no explicit background/inverse fill.
- Diagnostics, 1,000/5,000 rows: ~257/262 μs per frame, ~76 KB, 1,035 allocations
  on this workstation. Observational only; no wall-clock assertions.
- `git diff --check`: passed.

Logs are under [pass-3a](pass-3a/); inspect representative renders in
[testdata/ui](../../testdata/ui/README.md). No new dependency version was added;
the existing `termenv` dependency becomes direct for color-profile regression tests.

## Promotion

Release target: [v1.1.0-beta.6](https://github.com/richardnascimento18/devdock/releases/tag/v1.1.0-beta.6).
Feature PR: [#19](https://github.com/richardnascimento18/devdock/pull/19).
Promotion [#20](https://github.com/richardnascimento18/devdock/pull/20) produced
`ba525a76cb8f6418b9f0f3fa95f814e797f32e5a`. Actual published beta.6 assets passed
checksum/architecture checks; the downloaded amd64 passed version/commit and
expanded-at-the-time PTY smoke. Release run `37022997486` succeeded.
Reconciliation [#21](https://github.com/richardnascimento18/devdock/pull/21) produced
staging `1e00eb6696b1db93ccff25cdb20a158bc5a7074d`. Its push run `37024501159`
passed. Trees are equal and production is an ancestor; only permanent remote
branches remain. Complete live JSON/log evidence is included in the review package.
Protected feature → staging → production PRs, hosted runs, release verification
and production → staging reconciliation are recorded in the review evidence.
These steps must finish before 3B starts.

## Known limitations

Foundation retains existing workflow forms/list semantics. Dashboard/inspector/
palette, bulk operations, editor reflow and final motion polish belong to the
following milestones. Native arm64 execution and live OAuth remain Pass 4.
Compositor/wallpaper continuity cannot be observed by this headless environment;
the user requested inspection after the entire pass. This is explicitly pending
manual review, with no claim that ANSI/PTY tests substitute for that observation.
