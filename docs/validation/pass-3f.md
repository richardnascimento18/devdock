# Pass 3F — UX acceptance and regression corrections

Status: candidate implementation; **owner acceptance pending**. Pass 3 is not
independently accepted. No feature merge, staging/production promotion, release,
or Pass 4 work is authorized before the owner acceptance gate.

## Preflight and scope

Started from reconciled staging `37ac721918ccc04e66e3eda98c81380aa614b364`;
production `cbfc93cb85765afc043a3d90bd6a76723398a814` was its ancestor, with
identical file trees. Production remained default; only production/staging
remote branches remained; working tree was clean. Latest prerelease was beta.10;
beta.11 was unused. Rulesets 24337571 and 24336971 were active and retained
required CI, PR, deletion and non-fast-forward restrictions without bypass actors.
Classic protection endpoints return 404 because protection uses rulesets.
Evidence: [preflight](pass-3f/preflight.json), branch/release snapshots and rulesets
in [validation evidence](pass-3f/).

Branch `fix/pass-3f-ux-acceptance`; target `1.1.0-beta.11`. Responsive architecture,
transparent native background, palette, help, search, bulk selection, inspector,
shared animation clock, reduced motion and transactional persistence are retained.

## A. Move failure

Beta.10 accepted only EXDEV for its fallback and aborted unsupported NOREPLACE
flags as `rename: invalid argument`. Linux documents filesystem-dependent
RENAME_NOREPLACE and EINVAL for unsupported flags in the
[rename manual](https://man7.org/linux/man-pages/man2/rename.2.html).
An isolated disposable probe inside this mounted workspace reported `fuseblk`,
reproduced errno 22 / EINVAL and retained the source; ordinary rename succeeded.
This observation supports the owner finding; it is not owner project acceptance.
See [probe](pass-3f/mounted-rename-probe.json).

`core.Mover` supplies per-instance primitives, shared by standalone and bulk
commands. Native atomic no-replace remains first choice. EINVAL, ENOSYS and
EOPNOTSUPP (ENOTSUP alias on Linux) trigger validated source/containment checks
and a fresh Lstat destination check immediately before portable rename. Errors
still propagate; a second EXDEV follows isolated copy and guarded publication.
Copy preserves symlinks; publication failures retain source; staging/cleanup
failures stay visible, including PartialMoveError. State reconciliation follows
filesystem success, including existing partial-cleanup handling.

Tests force unsupported flags independently of CI filesystem support, exercise
native success, EXDEV, existing/late destinations (including dangling symlinks),
copy publication collision, failed fallback source retention, directories/files/
symlinks, standalone command favorite/recent remapping, and real-filesystem bulk
execution. Existing partial-cleanup and unsupported-copy tests remain.

The portable check/rename fallback cannot guarantee atomic no-replace against
an external writer between the check and rename. This limitation is documented
in README and explicitly deferred to Pass 4 concurrency hardening; no journal or
full transaction system was introduced. Owner standalone/bulk/collision/state/
favorite/recent results on the actual mounted workspace: **pending**.

## B. Legacy .devdock

History at `d0cf60e` (`ReadDevDockMarker` / `ClassifyDir`) treated valid markers
with any type other than group as project boundaries. That included missing or
blank types; historical parsing was also overly permissive for malformed files.
The intended compatibility subset is restored: valid missing/whitespace-blank
`type` normalizes to project in memory; explicit project/group and subgroup group
alias retain semantics; malformed TOML and unknown explicit nonblank types error.
No file is rewritten.

Eight stored TOML fixtures test the full scanner outcome: no type, empty type,
project, group, subgroup, malformed, unknown, and other legacy fields without
type. They verify project boundaries, empty group retention and unchanged marker
bytes. Actual owner workspace scan result: pending.

## C. ANSI corruption

The preset picker embedded dim ANSI-rendered default/split fragments inside an
underlined parent style. It now truncates plain names first and styles arrow,
name, default marker and summary independently. Long names reserve space for the
default marker. Dashboard metadata is also joined from separately styled semantic
fragments. Render tests reject incomplete/non-SGR escapes and visible `[38;2;` /
`[38;5;` fragments across truecolor/256/basic/no-color profiles. Declared literal
user text is distinguished from renderer metadata. Python tests and PTY smoke
validate emitted controls and transparency as well.

## D. Project scanability

Rows again identify projects without requiring the Inspector: bold names,
foreground stack glyphs, GitHub/local-Git/local status, favorite and cached tmux
markers. Location is compact, muted and component-aware, with nearby ancestry
retained at smaller widths. Tags precede location in the secondary line; roomy
rows align location after metadata. Technical short IDs appear only when same-name
compact locations collide. Loaded scan entries cache local `.git` presence;
rendering performs no filesystem work. Canonical Git remote matching remains the
existing implementation and is recorded for Pass 4.

Wide/medium/narrow fixtures include duplicate names, local Git, GitHub and tmux.
The Inspector uses a clear name/stack lead, section labels and semantic accents,
with complete paths and loaded status for investigation.

## E. Workspace navigation

The former two-Tab Projects → Inspector → Workspace route is optional now.
Alt+1/2/3 focus Workspace/Projects/Inspector directly; Tab/Shift+Tab retain the
cycle. Backspace moves to parent scope, Ctrl+U to current root, Ctrl+A to all.
These ordinary editing keys do not change scope during text entry. Explicit Alt
pane navigation applies a search query and exits input without inserting digits.
The palette appends fuzzy `Scope <root › ancestry>` entries (including collapsed
nodes) and explicit parent/root/all actions; selecting a scope returns to Projects.
Scope changes preserve query and clear stale multi-selection through the existing
Update boundary. Tree guides, kind accents, active scope glyph, focus and subordinate
counts establish hierarchy. Duplicate root basenames receive secondary IDs.

## F. Editor UX

110+ columns: persistent collection / windows or ordered steps / properties.
74–109: collection + detail, with component previews on the item view. Narrow:
existing page flow with definition/component breadcrumb and visible save/cancel.
Window, nested pane, metadata and step fields update the property area while
collection context remains visible. Settings uses sections/values; root workflows
keep section context at 74+ columns.

Alt+1 browses the collection without replacing the draft; Enter/Esc invokes the
unsaved guard. Alt+3 resumes properties. Alt+2 accepts validated window/step fields
and returns to the item view. Existing sub-editor controls remain available.
Resize preserves active definition identity and unfinished fields. Tests cover
selection, inline property edits, collection return, discard/continue, save/cancel,
wide→medium→narrow resize and near-field validation.

No persistence rewrite: collection-wide validation, duplicate prevention, default
rename compensation, deep draft copies, cancellation and atomic saves remain.
Credential values remain excluded from editor views.

## G. Tmux UX

Backend `devdock-<directory>-<path hash>` identity remains unchanged. Loaded project
sessions display project names and location; external/legacy sessions retain their
own names. Inspector technical details may show the backend ID. Kill confirmation
uses the display name or full readable project path for duplicate names; it keeps
the exact backend ID separately. Tests verify the human target and exact kill ID.

## H. Status and warnings

Scan warnings retain full contextual messages separately. Dashboard shows severity,
count and `! details`; Status Details appends all warnings even if another status
message becomes current. Successful rescans clear stale scan warnings. Tests verify
concise status and complete detail retention; long-detail paging remains covered.

## I. Branding

The original beta six-line mark is restored for wide (110+) and tall (36+) empty/
startup states only, reserving body space. Normal use retains the compact DevDock
header; narrow screens keep a minimal wordmark. Foregrounds only.

## J. Motion

New activity remains static for 320 ms before the shared clock begins. Subsequent
160 ms frames use a slower half-speed foreground sweep and restrained luminance;
only transient activity labels animate. Normal navigation and confirmations remain
static. Generation checks, idle/hidden-work handling and reduced motion remain.

## K. Transparency

No pane/row/modal fills, reverse-video styles or decorative backgrounds were added.
Existing source and light/dark × color-profile rendered invariants remain, with
all fixtures under repeated resize and PTY emitted-byte checks. Owner Caelestia/
wallpaper confirmation is pending; automated checks cannot establish it.

## L. Manual acceptance

The [mandatory checklist](pass-3f/manual-acceptance.md) prominently requires real
mounted standalone/bulk moves, collision behavior, project state and favorite/
recent remapping, plus full UI/resize/transparency review. Candidate SHA, PR and
binary path are supplied at the checkpoint and in review-package `candidate.json`.
Owner feedback, requested corrections and explicit merge approval: **not received**.
The feature PR stays open for corrections on the same branch.

## M. Automated validation

Detailed logs live in [pass-3f](pass-3f/); machine-readable counts are in
[test-summary.json](pass-3f/test-summary.json). Current totals: 157 top-level Go
tests, 433 including subtests, zero skips/failures, 59 ANSI/plain fixture pairs,
11 Python tests; total statement coverage 66.9%.

Exact minimum Go 1.26.0 compatibility tests passed. Its vulnerability scan reports
18 called standard-library advisories, fixed in subsequent Go patch releases;
that historical minimum is not used for candidate release builds. Full validation
uses patched Go 1.27.1 and includes formatting, tidy/module consistency, tests,
race, vet, staticcheck, vulnerability scan, build, actionlint and Python tests.
The original modified Arch Go toolchain failed to compile tests; standard Go
1.26.0/1.27.1 are used instead. Sandbox cache/network failures were rerun with
approval; no test or analyzer finding was waived.

Linux amd64/arm64 builds, ELF architecture checks, checksums and amd64 --version
passed. Normal/reduced-motion PTY smokes passed, covering Settings persistence,
legacy marker startup, direct pane/scope navigation, palette, search, multi-selection,
editor drafts, resize, OAuth/cancel and actual emitted transparency/ANSI. Final
candidate rebuild identifies its exact committed SHA; fresh published-artifact
verification remains deferred until owner-approved promotion. Full validation on
patched Go 1.27.1 passed with no vulnerabilities. Hosted CI evidence is added to
the review package at the acceptance checkpoint.

## N. GitHub lifecycle

Feature → staging PR: created only after local validation. Green hosted feature CI
is required at the acceptance checkpoint. **No merge before owner approval.**
Staging push CI, staging→production PR/source validation/full matrix, merge, beta.11
release, fresh download of actual published assets, checksums/architecture/version/
PTY/transparency/ANSI checks and protected production→staging reconciliation remain
pending until approval. Local candidate binaries are not published-release evidence.
Final production ancestor/identical-tree checks and permanent-branch cleanup follow
release. This report will be updated with actual results after acceptance.

## O. Deferred Pass 4

[Complete backlog](../pass-4-backlog.md): crash/power-loss journals, durable writes,
full concurrency transactions (including portable publication race), multi-file
recovery, real OAuth authorization matrix, native ARM64 runtime, full tmux/backend
and scaffold integration, signal/interruption torture, immutable release hardening,
signing/provenance, reproducible builds, clean-machine installation rehearsal,
dependency/security/license audit and final stable-release rehearsal. Also canonical
Git remote identity: Git-reported remotes; SSH/HTTPS and .git normalization;
origin/upstream; forks; renamed directories; worktrees; multiple remotes; duplicate
clones; malformed remotes; GitHub Enterprise/non-GitHub hosts. No Pass 4 campaign
was begun. Move portability was the expressly authorized current-regression fix.
