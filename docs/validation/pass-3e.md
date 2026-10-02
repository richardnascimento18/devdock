# Pass 3E — motion and final polish

Branch `feat/ui-motion-polish` starts from reconciled beta.9 staging
`6ab158d084e52e0f37b7d2e57c3e76990219c7aa`, after protected source-neutral
CI recovery #31. Production `1975e7a1e641e244c54763cde9e0bfba218ca1a6`
is its ancestor, trees match, and only permanent remotes remained at the boundary.

## Architecture and behavior

One generation-checked 120 ms presentation clock schedules only visible work.
It replaces all operation spinner loops; stale/idle ticks cannot restart it.
Scan, clone/create, move/delete, OAuth, repo/tmux loading and template execution
share foreground-only, grapheme-safe progress. Startup scanning runs asynchronously
through the existing snapshot adapter. Rendering remains free of filesystem and
network calls. Reduced motion uses `DEVDOCK_REDUCED_MOTION=1`/true/on, `NO_COLOR`,
or `TERM=dumb`; service execution remains independent and no migration is needed.

Sibling-aware tree guides cap visual indentation while preserving logical depth
and full location details. Inspector and list show cached tmux availability when
known, with no invented attachment or Git branch facts. `!` opens complete pageable
status/error details. Narrow dialog hints retain save/confirm/cancel. Long editor
definition confirmations now page consistently. Template terminal chrome uses the
available width/height and preserves its interrupt hint at 24×8.

## Validation

- Full existing validation: formatting, tidy consistency, module verification,
  tests, race, vet, staticcheck, govulncheck, build, actionlint and 8 Python tests.
- Exact Go 1.26.0 tests, Linux amd64/arm64 release builds, SHA256SUMS and ELF
  architecture checks; native amd64 smoke with normal and reduced motion.
- 137 top-level Go tests, 357 including subtests, 40 ANSI/plain fixture pairs.
- All fixture states undergo repeated resize at 120×40, 100×30, 80×24, 60×20,
  40×15, 24×8, 1×1, 0×0 and 300×80, with cell/height/UTF-8/background checks.
- Clock lifecycle, no duplicate timers, late/stale completion, reduced-motion
  environments, async startup, scan identity, cached inspector, compact PTY,
  long error/definition paging and editor save/cancel hints have regression tests.
- Source, truecolor/256/basic/no-color × light/dark rendered profiles, and actual
  emitted PTY bytes reject decorative backgrounds and inverse video. Both Go and
  Python detectors distinguish legal colon/semicolon foreground RGB payloads.
- Smoke additionally exercises status details and explicit refresh; hosted
  verification includes both normal and reduced-motion smoke without weakening
  any existing check. No dependencies were added; existing uniseg is now direct
  for grapheme-safe progress.
- Hosted run `37075950794` caught a domain-refresh test that passed a Tea command
  bundle directly to Update instead of executing its scan command when animation
  was enabled. The test now exercises both motion modes and still asserts the
  real refreshed domain snapshot. Full local validation also runs with NO_COLOR
  unset to match hosted behavior; no check or assertion was waived.
- Staticcheck identified obsolete `centerInTerminal`; it was removed, not waived.
- Coverage and performance reports are included in [pass-3e](pass-3e/). Diagnostics
  cover 1,000/5,000 projects, fuzzy input, deep trees, large selection and resize.

Local binaries built before the milestone commit identify the reconciled base
commit; their logs validate the working source. The release verification separately
checks actual published artifacts against the production merge commit.

Compositor/wallpaper review remains pending the user's final transparent-terminal
inspection. Native arm64 runtime and live backend integration stay in Pass 4.
Optional mouse, bulk deletion and external editor handoff are not included.

## Promotion and review

Target `v1.1.0-beta.10`. The final review package records actual feature/production/
reconciliation PRs, hosted runs, production/staging SHAs, published-asset verification,
branch cleanup and complete Pass 3 assessment. Pass 3 is not declared shipped until
the protected promotion, release checks and ancestry reconciliation finish.
Do not begin Pass 4.
