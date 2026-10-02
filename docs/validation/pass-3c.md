# Pass 3C — bulk workflows

Branch `feat/bulk-workflows`, from reconciled staging
`74b2978f89a04b3108f6b90f4522af448ba795e4`. Production beta.7
`67de3ebd1eab8fe9ec51ae84091c0043f4c85500` is its ancestor with equal trees.
Preflight confirmed clean working tree, only permanent remote branches,
production default, active protections and automatic branch cleanup.

## Scope and behavior

Space selects canonical loaded project paths. A foreground dot distinguishes
selection from the focused row and favorite. Selection survives navigation and
filtering, with total/hidden counts; root, scope or collection changes clear it.
Escape cancels search before clearing selection. Snapshot reconciliation removes
missing projects. Stale recents cannot become mutable selected targets.

Bulk move uses existing safe PlanMove/ExecuteMove APIs. It preflights every target
and collisions within the batch before confirmation, displays complete paths in
scrollable transparent modals, and re-plans every target before the first move.
Conflicts disable execution. Each backend execution revalidates again. Results
separate moved, cleanup warnings and failures; completed paths reconcile favorites
and recents, pending state saves remain retryable, and an explicit rescan updates
the snapshot. Executed selection clears to prevent stale partial targets. Failed
preflight moves nothing and preserves selection for correction.

Bulk favorites favor all, or unfavor all when every selected project is already
favorite. Hidden selections must be made visible before bulk actions. Palette
captures selection identity; destructive/delete/remove actions require clearing
selection. Bulk deletion is intentionally deferred. Domain and filesystem safety,
configuration transactions, OAuth and tmux services are unchanged.

## Validation

- Full existing check script: formatting/tidy/module verification, 8 Python tests,
  Go tests/race/vet/staticcheck/govulncheck/build, actionlint and version policy.
- Exact Go 1.26.0 tests; linux amd64/arm64 ELF builds and SHA256SUMS.
- 124 top-level Go tests; 276 tests including subtests; 24 paired render fixtures.
- Tests cover selection/filter/scope/clearing, low-risk bulk favorites and delete
  guards, entire-batch preflight, changed later destination before execution,
  intra-batch collision, partial errors, stale async messages, full move flow,
  favorite/recent reconciliation, modal cancellation and resize.
- Source-style/color-profile/render and emitted-PTY transparency checks passed.
- Binary PTY smoke includes multi-select and bulk destination cancellation plus
  dashboard/search/palette, editors/Settings persistence, OAuth, resize and quit.
- Whitespace review passed, including newly added fixtures after staging.

Logs: [pass-3c](pass-3c/). No dependency added. Motion still uses the existing
operation-aware spinner until 3E. Compositor review remains pending the user's
final terminal inspection; no headless test claims wallpaper observation.

## Promotion

Released [v1.1.0-beta.8](https://github.com/richardnascimento18/devdock/releases/tag/v1.1.0-beta.8)
through feature [#25](https://github.com/richardnascimento18/devdock/pull/25),
production [#26](https://github.com/richardnascimento18/devdock/pull/26), and
reconciliation [#27](https://github.com/richardnascimento18/devdock/pull/27).
Production `393ee9dc4d9d798b758dd818b17c209ceda15071`; staging
`4ea22eadf5fb0cbbddce25b6e371c8f2eeaf0a0c`. All hosted CI passed, including
final staging push `37070659803`. Actual published assets passed tag/commit,
prerelease, checksum, architecture, amd64 metadata and expanded smoke checks.
Production is an ancestor of staging and trees match; remote cleanup verified.
Complete live evidence accompanies the final review package. Pass 4 stays deferred.
