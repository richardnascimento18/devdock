# Pass 3D — editor redesign

Branch `feat/editor-redesign`, from reconciled beta.8 staging
`4ea22eadf5fb0cbbddce25b6e371c8f2eeaf0a0c`. Production
`393ee9dc4d9d798b758dd818b17c209ceda15071` is its ancestor with identical trees;
only permanent remotes remain. Previous milestone's published assets and every
protected workflow check were verified before this branch began.

## Architecture and behavior

Editor rendering is separate from navigation/guards and existing draft/save
handlers. Wide collections have a compact list and preview; narrow collections
stack them with bounded detail scrolling. Settings shows current roots, default
preset and root actions, excludes credentials, and reuses configuration/root
transactions. Presets mark the default and expose window/pane fields. Template
forms show metadata, clear step types, command/path/output fields and full selected
step previews. Active fields remain visible, inputs resize, and complete long
text is recoverable through input navigation or detail paging.

Draft snapshots produce an explicit Unsaved label. Leaving a changed document
asks to discard; Escape keeps editing. Typing Escape stops typing before back.
Nested field cancellation is labeled; split-pane Escape respects nested typing
and pane states. Ctrl+S validates and commits through the approved persistence
handlers; failures retain drafts and active collections. Template step acceptance
now checks quoting, relative path and output safety near the form before save;
collection-wide save validation remains in place.

Collection removal is confirmed and uses existing atomic validated saves.
Configured/implicit default presets are guarded until another default is chosen.
Removing an earlier preset preserves active selection by name rather than index.
The approved default-preset rename transaction and compensation remain unchanged.
Optional external-editor handoff is not included; no new dependency was added.

## Validation

- Full existing check script, exact Go 1.26.0 tests, both release architectures,
  SHA256SUMS and expanded release-binary PTY smoke.
- 131 top-level Go tests, 296 tests including subtests, 8 Python tests and
  32 paired ANSI/plain fixtures, including narrow Settings, window/pane/step/meta
  forms, unsaved confirmation, validation and definition removal.
- Tests cover save/failure transactions, default rename/removal, collection
  removal/cancel/write failure, unsaved continue/save/discard, output and quoting,
  Unicode labels, 200-level roots, nested Escape, focus and pathological resize.
- Transparency source/render/profile/PTY checks remain green. Staticcheck found
  obsolete fixed-box wrappers after the redesign; they were removed, not waived.
- PTY smoke exercises unsaved preset confirmation and cancellation, template
  entry, Settings persistence and all existing dashboard/workflow entry checks.
- Whitespace review includes newly staged fixtures. Logs: [pass-3d](pass-3d/).

Compositor review remains pending the user's final transparent-terminal inspection.
Native arm64 and live OAuth remain Pass 4. Shared motion/final polish follow in 3E.

## Promotion

Released [v1.1.0-beta.9](https://github.com/richardnascimento18/devdock/releases/tag/v1.1.0-beta.9)
through feature [#28](https://github.com/richardnascimento18/devdock/pull/28),
production [#29](https://github.com/richardnascimento18/devdock/pull/29), and
reconciliation [#30](https://github.com/richardnascimento18/devdock/pull/30).
Production `1975e7a1e641e244c54763cde9e0bfba218ca1a6`; all PR, initial staging
push and release checks passed. Published assets were downloaded fresh and passed
tag/commit, prerelease, architecture, SHA256SUMS, amd64 version and PTY checks.

GitHub did not create a staging push run for reconciliation merge
`919e8f7b56ea709ad4d61e228c188b17e3bb5036`. The cause is unknown; an absent run
is not reported as passed. To require real staging push validation before 3E,
source-neutral branch `chore/pass3d-ci-recovery` added an empty commit through
protected [#31](https://github.com/richardnascimento18/devdock/pull/31).
Its PR CI `37073754222` and final staging push `37073871343` both passed.
Final staging `6ab158d084e52e0f37b7d2e57c3e76990219c7aa` contains production
with identical trees; only permanent remote branches remain. No source, version,
workflow, protection or release asset changed during recovery. Live evidence and
the missing-run investigation accompany the final independent-review package.
