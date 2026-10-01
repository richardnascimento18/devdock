# Pass 1 engineering report

Date: 2026-10-01. Baseline: `76879fb`, local `main`. Scope: structural stabilization of the existing DevDock behavior. No remote push, merge, branch creation, tag, or release publication was performed. No recursive-domain replacement, bulk selection, responsive-pane redesign, or visual theme work was introduced.

The requested GPT-6 Sol/High model setting could not be changed from inside this active session; that limitation was disclosed before work. No delegated agents were used.

## A. Repository audit

All tracked production/configuration/documentation files were inspected before substantial edits, together with recent materially relevant history. The initial findings are retained in [pass-1-audit.md](pass-1-audit.md).

The baseline compiled but contained no Go tests. Major issues were mixed filesystem/integration responsibilities in Bubble Tea flows, a 1,895-line editor, repeated atomic-write implementations, broad mutable detector caches, unused indexes, sentinel-based workflow routing, unowned asynchronous messages, discarded subprocess/tmux errors, unsafe collision/deletion semantics, and a default template that could not pass its own loader validation. No TODO/FIXME comments advertised those behavioral gaps.

The most recent baseline change had introduced the whole editor and tmux management together, explaining incomplete draft synchronization and duplicated persistence. README Go/version/download claims disagreed with actual dependencies and release infrastructure. There were no CI or release workflows.

## B. Refactors performed

| Boundary | Change and reason | Behavioral impact |
| --- | --- | --- |
| Workspace/core | Centralized names, strict descendant checks, creation collision rules, deletion, and moves. Removed filesystem copy logic from UI. | Existing hierarchy retained; invalid/destructive paths and directory merges rejected. |
| Scanning/detection | Collectors return partial results plus errors; per-scan Detector owns caches. Deleted unused persisted index and unconsumed UI maps. | Available projects survive partial failures; no hidden root failure or unbounded detector cache. |
| Editor | Split orchestration, preset/window editing, pane editing, and template editing into cohesive files. Clone → validate whole collection → save → update active revision. | Same editor interface; failed saves cannot change active collections. |
| Persistence | Config/preset/template/state/marker writes use one synced, closed temporary-file atomic writer. Config loads no longer save a migration as a hidden side effect. | Invalid data is reported; failed writes preserve old files/active preferences. |
| GitHub | Separated context-aware HTTP/auth, Git subprocesses, and Tea commands; introduced small HTTP/auth/Git execution test boundaries. | Device requests/polling do not block Update; cancellation, errors, and stale messages are explicit. |
| tmux | Runner-backed client owns preflight, commands, pane identities, selection, and attachment. | Failures propagate with context; configured pane topology is preserved. |
| PTY/process | Session owns reader, wait, event ordering, context cancellation, and process-group shutdown. | Exactly one completion after output; failed commands remain failures. |
| Application | Typed root/domain/group intent; separate creation/clone/move/delete flow files; operation IDs for auth, PTY, scans, repository loads, spinner, and tmux refresh. | Wrong-flow transitions and stale responses are rejected. |
| Rendering/inventory | Workspace snapshot includes projects/domains/groups. Rescan runs asynchronously; root filtering/list rebuilding use cached snapshots. | Root switching does not rescan; tree rendering does not repeatedly read directories. |
| Build/toolchain | Explicit VERSION, linker metadata, public Client ID injection, Linux cross-build/checksum scripts. | Local and CI release builds are reproducible with matching inputs; ordinary source builds remain available. |

Small creation/config/stat operations remain synchronous behind package helpers; long scans, Git/network work, PTY operations, moves, recursive deletion, and tmux listing/killing use command boundaries. Interfaces were added for external execution/testing rather than every domain object. No wholesale UI rewrite or framework major-version migration was performed.

## C. Every known bug

All fifteen listed bugs were fixed; none is being described as blocked.

| Prompt | Root cause and correction | Regression evidence |
| --- | --- | --- |
| 4.1 PTY exit status | EOF discarded Wait status and input could start concurrent reads. One reader and waiter drain output and emit one final error-bearing completion; UI aborts later/post steps and never queues a successful launch on failure. | `internal/pty/pty_test.go`: success/exit 7/output order/one completion; `pty_screen_test.go`; `TestTemplateFailureNeverQueuesSuccessfulLaunch`. |
| 4.2 Multi-root group deletion | Root picker interpreted magic strings through unrelated creation routing. Typed rootPickerIntent/groupWorkflow routes deletion explicitly. | `TestMultiRootGroupDeleteRequiresFullPath`; sentinel-looking project names remain real names. |
| 4.3 Dangerous group deletion | Picker Enter immediately recursively removed a group. Full target-path confirmation precedes async deletion; project/domain confirmations use the same standard. Rooted removal confines traversal. | Multi-root group test and project/domain deletion confirmation table; core deletion/root/symlink safeguards. |
| 4.4 Move/copy correctness | Copy skipped links, rename/copy could merge targets, cleanup errors were ignored. No-replace rename; EXDEV-only isolated staging; links copied as links; special files rejected; copy/cleanup errors propagated as partial failures. State follows complete moves. | `TestMove`, `TestMoveFailurePreservesData`, FIFO/source-preservation test, metadata test, UI favorites/recents move test. |
| 4.5 Editor persistence | Editor changed live collections before Save. Proposed deep copy is saved first and active revision is committed afterward. | `TestEditorSaveTransactions`: filesystem failure leaves draft/active/persisted values consistent. |
| 4.6 Duplicate names | Individual validation missed collection rename/new-name collisions. Entire proposed preset/template collections are validated before save. | Editor transaction tables, preset/template validation and default round-trip tests cover duplicate, rename collision, and unchanged old state. |
| 4.7 Scanner errors | Multi-root/domain errors were continued/ignored. Joined contextual errors accompany successful projects; UI reports incomplete scans. | `TestScanPartialFailure`, unreadable-domain partial fixture, unavailable-root state test. |
| 4.8 OAuth cancellation | No polling context/identity, and StartDeviceFlow ran in Update. Commands use cancellable contexts and attempt IDs; credentials only commit after save. | Fake HTTP cancellation/errors; `auth_test.go` async start, Esc, late success/start, previous-attempt response, failed credential save. |
| 4.9 Spinner ticks | Every tick scheduled another regardless of state. Ticks require an active loading state and matching spinner ID. | `TestSpinnerLifecycleAndEmptyRoots`; stale ticks produce no new command. |
| 4.10 tmux errors | Session/window/pane/send/select/attach calls dropped errors and guessed pane targets. Runner client checks each operation and uses returned IDs. | `TestWorkspaceCommandsAndEveryFailure`, existing session attach, window/literal commands; nested target assertions. |
| 4.11 Command parsing | strings.Fields split quoted arguments/expanded paths. Custom argument parser handles quotes/backslashes before variable substitution; exec receives argv directly. | Parsing tables include spaced/quoted/escaped args, malformed quoting, variables, and literal shell syntax. |
| 4.12 Unsafe template removal | `rm .` and symlink/path traversal were accepted. Strict project-relative descendant validation rejects equality, absolute paths, `..`, and symlink components; rooted handles confine builtins/output. | Builtin safety, internal-parent traversal, command output, marker symlink tests. |
| 4.13 Creation collisions | MkdirAll treated existing projects as creation success; picker checked presentation rows. Exclusive project creation/central destination checks and raw-project inventory checks. | Core create collision tests, template Run collision, clone preflight collision, flat/collapsed picker tests. |
| 4.14 Favorites/state | Move/delete did not reconcile paths; recent filtering/counts depended on presentation; startup left stale entries. State helpers update successful moves/deletions and prune known missing paths only in available roots; root filtering applies to favorites/recents. | `state_test.go`, UI root filtering, unavailable root, move, delete and failed save tests. Permanent project IDs were not added. |
| 4.15 Detector mistakes | Exact `.csproj`/`.fsproj` names, wrong Solid/Fastify/F# mappings, generic Python marker false positives, impossible condition and random ties. Correct globs/associations, parsed dependency declarations, stable ordering, scan-scoped caches. | All-primary-marker table, framework/Python fixtures, group markers, cache/order tests. |

## D. Additional confirmed bugs

- Generated FastAPI defaults contained shell redirection that the loader rejected. Explicit stdout `output` replaces redirection, without enabling shell evaluation.
- PTY post-step transition stalled, and control-byte rendering could loop forever. Both now advance with regression tests.
- Pane Enter was intercepted by its parent editor; Ctrl+S omitted current window/step/pane drafts. Draft save and nested-layout tests protect the correction.
- Editing an unchanged-count nested layout flattened its topology; template Layout/Output metadata could disappear. Existing topology and metadata are retained.
- Clone → new domain resumed project creation. Typed domain intent preserves clone routing.
- Reusing group fields inherited a previous domain/phase. New flows reset owned group state.
- Empty root picker confirmation could panic. Bounds checks return a real error.
- Resize was consumed globally and never reached active PTY/editor screens. Child screens now receive it.
- Invalid config could launch setup and overwrite configuration; legacy Load performed an ignored write. Load now reports malformed data and migration is read-only.
- State JSON errors were suppressed or exposed partially decoded data. Load returns fresh defaults and an error; invalid saves preserve old disk state. Failed saves retain an error after TUI shutdown.
- Git remote substring parsing accepted lookalike hosts; worktrees were missed and stale GitHub badges persisted. Exact host parsing/worktree detection/reset fixes are tested.
- Git initialization/README/push failures could lead to a success message. Every command is checked and the UI aborts launch on failure.
- tmux nested layouts targeted wrong panes and were overwritten by tiled; existing TMUX attachment required switch-client. Correct IDs, layout preservation, and contextual switch/attach behavior replace this.
- Domain creation did not refresh the new domain in cached inventory. Snapshot-refresh regression test added.
- `.ddignore` matching interpreted glob characters in the root path. Root-relative matching and a root-containing-`[` regression fix it.
- Preset pane sizes and sanitized tmux window-name collisions lacked validation. Collection fixtures now reject them.
- Vulnerability inspection found an affected indirect/direct x/sys version (the reported Windows-only path was not reachable on Linux). Minimal update to v0.44.0 removes the advisory; final govulncheck reports no vulnerabilities.

## E. Tests and manual verification

Added **14 Go test files, 64 named Go tests**, with additional table-driven cases/subtests. Files are `auth_test.go`, `editor_test.go`, `flows_test.go`, `pty_screen_test.go`, and one suite in each of core, config, detect, fileutil, github, preset, pty, state, template, and tmux. Two Python unittest methods exercise SemVer validation/ordering with multiple fixtures.

All Go tests and race tests passed. Go 1.26.8 minimum-supported-toolchain tests also passed. Temporary directories cover creation/deletion/collisions/moves/symlinks/FIFO rejection and simulated EXDEV/cleanup errors. External clients/runners isolate GitHub, Git, and tmux. Tests do not authorize accounts, contact real APIs, invoke user editors, or use real user workspaces. PTY tests intentionally execute short local shell processes and verify reaping.

Measured statement coverage:

| Package | Coverage |
| --- | --- |
| Root application | 33.2% |
| config | 63.2% |
| core | 69.2% |
| detect | 85.4% |
| fileutil | 76.9% |
| github | 56.3% |
| preset | 75.0% |
| pty | 81.6% |
| state | 78.4% |
| template | 79.9% |
| tmux | 61.6% |
| Total | 43.9% |

[coverage.log](validation/coverage.log) records the final per-package run. Coverage is concentrated on safety, transactions, lifecycle, and failure paths, rather than artificial rendering tests. Interactive editor/navigation paths remain less covered.

A real amd64 binary was run in an isolated temporary HOME/workspace and sized pseudoterminal: startup, project rendering, flat view, root switching, favorite toggle, editor opening, missing OAuth ID error, Esc cancellation, and clean shutdown passed. Linux amd64/arm64 files were built as static ELF binaries and checksums verified. Arm64 was cross-built, not executed.

Not manually exercised: real GitHub authorization/private repo creation/push, real tmux attachment/window rendering, external editors/scaffolding tools, native arm64, GitHub-hosted workflows/publication. Those require external accounts/services/tools/hardware or owner-authorized remote actions. Fake integration tests verify construction and failure behavior; they do not substitute for eventual end-to-end release hardening.

## F. Actual static/build outcomes

The final full script exited **0**, with no linter suppressions and no skipped validation stage. See [full-check.log](validation/full-check.log), [minimum toolchain log](validation/go-1.26-tests.log), [coverage log](validation/coverage.log), and [build/smoke evidence](validation/build-smoke.log). Logs are text-only and contain no OAuth credentials.

| Check | Actual outcome |
| --- | --- |
| `gofmt -l .` | No unformatted files. |
| `go mod tidy` consistency / `go mod verify` | No changes; all modules verified. |
| `go test ./...` | Passed all packages. |
| `go test -race ./...` | Passed all packages on Linux with CGO. |
| `go vet ./...` | Passed. |
| `staticcheck ./...` (v0.8.1) | Passed; no broad nolint directives. |
| `govulncheck ./...` (v1.8.0) | Passed: `No vulnerabilities found.` |
| `go build ./...` | Passed. |
| actionlint (v1.7.12) | Passed all workflows. |
| Python version-script tests | Passed. |
| Linux amd64/arm64 release builds | Passed; SHA-256 verification passed. |

Sandboxed govulncheck initially failed DNS; it was rerun with permitted network access, not omitted. A coverage-summary command initially used a read-only default Go cache; rerunning with the writable task cache succeeded. The first terminal smoke harness lacked a terminal size; sizing the test terminal resolved the harness failure and actual checks then passed.

Go support is deliberately **1.26+**, CI latest patches of 1.26/1.27, releases built on latest 1.27. Dependency metadata/rooted filesystem APIs have a technical floor of 1.25 (x/sys v0.44.0); maintained support and analysis-tool requirements justify 1.26 as the supported minimum. Existing Bubble Tea/Lip Gloss major versions were retained. Module tidy corrected direct dependencies and removed no required runtime capability.

## G. CI/CD

- `ci.yml`: pull requests targeting staging/production and staging pushes. Read-only permissions; cancellation of superseded runs; promotion policy plus reusable verification aggregate into **Required validation**.
- `verify.yml`: Linux latest Go 1.26/1.27 matrix, Go caching, pinned analysis tools, full checks, both Linux release builds, checksum checks, amd64 metadata execution.
- Production PR policy accepts only same-repository staging and a strictly newer explicit VERSION; the exact PR candidate is reverified, independently of staging checks.
- `release.yml`: production pushes rerun full verification, then build artifacts with the required public `vars.DEVDOCK_GITHUB_CLIENT_ID`. Read-only build job, separate environment-gated `contents: write` publish job. The latter checks artifacts, checks out/executes no repository code, refuses mismatched existing tags, and supports failed-upload retries.
- SemVer is explicit in VERSION; initial `0.1.0-dev.1` is a prerelease. Tag is `v<VERSION>` at the production push SHA; no speculative increment script. Prerelease identifiers produce GitHub prereleases.
- Artifacts: `devdock_<version>_linux_amd64`, `devdock_<version>_linux_arm64`, `SHA256SUMS`. Linker metadata records version/commit/commit date/public Client ID. CGO is disabled for distributed binaries.
- Official checkout/setup-go/upload/download actions are pinned to reviewed commit SHAs. No pull_request_target, privileged PR execution, automatic merge bypass, or credential exposure to forks.

Owner-side steps are explicit in [CONTRIBUTING.md](../CONTRIBUTING.md): establish staging/production baseline branches; required PR reviews, most-recent-push approval, stale-review dismissal, **Required validation**, up-to-date branches, conversation resolution, no force push/deletion/admin bypass; configure the public OAuth repository variable and production-only production-release environment. These settings were **not** remotely configured. Do not enable a merge queue without adding merge_group verification. Release serialization is not an unlimited pending-run queue; finish publication before the next promotion.

## H. Security notes

Only GitHub's public OAuth App Client ID is embedded; no OAuth client secret exists in source/configuration/API requests/build flags or binaries. Device flow needs the Client ID, not a secret. Runtime environment override supports self-built/custom apps. Credential config is mode 0600; raw API response bodies are not printed as errors. The real official ID is owner-supplied, not invented in this pass.

Filesystem operations reject unsafe names, root equality, escapes, collisions, and symlink components. Rooted recursive deletion and builtin template file operations prevent escapes during traversal. Cross-device copies preserve links without intentional recursive traversal and reject unsupported special files. Partial failure never reports full move success. Atomic writes replace a symlink itself rather than following it.

Arbitrary user-configured template/preset commands remain trusted executable code, not sandboxed commands. No automatic shell expansion was added. Race-free confinement for every adversarial concurrent mutation of a workspace during move/copy/creation is not claimed. Atomic file replacement is guaranteed at the rename commit point; directory-metadata durability after abrupt power loss is not promised.

## I. Documentation

README now accurately describes Linux/Go requirements, source/binary installation, current keys/features/hierarchy, templates/presets/state/config/markers, partial failures, confirmation, OAuth behavior, missing ID errors, official public-ID injection, custom app builds, checks, and unreleased infrastructure status.

CONTRIBUTING documents package ownership, testing, permitted development branch prefixes, Conventional Commits, PR/release promotion, merge ancestry, hotfix procedure, explicit SemVer, reproducibility inputs, exact required GitHub settings, Actions security, publication retries, and filesystem/persistence limitations. The initial audit and final validation logs/report remain committed review artifacts.

## J. Local commit history

Each production milestone was tested before committing; workflow milestones also passed actionlint/version checks. Published history was not rewritten.

| Commit | Explanation |
| --- | --- |
| `2ce16cb` docs: record full repository stabilization audit | Initial full audit and boundaries. |
| `84471b9` fix(workspace): enforce collision safety and preserve partial move failures | Core/scanner/detector safety and initial fixtures. |
| `931bc61` fix(scaffold): preserve PTY exits and secure template execution | Ordered process completion, parsing, builtin safety. |
| `83c8c50` refactor(editor): separate responsibilities and commit validated state atomically | Editor/persistence transactions and validation. |
| `931f80b` refactor(integrations): isolate clients and propagate lifecycle errors | GitHub/Git/tmux clients and fakes. |
| `fd515df` fix(app): type workflow intent and isolate asynchronous workspace state | Flow identities, async inventory, reconciliation. |
| `90a7555` fix(editor): preserve active pane drafts and nested layouts | Additional editor regressions. |
| `556adaa` test: broaden integration and application regression coverage | Integration/state/detection tests. |
| `1f72073` fix(app): finalize startup reconciliation and persistence error reporting | Startup, shutdown/save errors, metadata entrypoint, path validation. |
| `b2d8876` fix(workspace): confine recursive deletion and test confirmation workflows | Rooted delete plus project/domain confirmation tests. |
| `0d2d374` build: align supported Go versions and generate reproducible Linux releases | Build/check/version tooling and minimal advisory update. |
| `fe0940c` ci: verify staging promotions and publish protected production releases | Read-only verification and protected publication. |
| `28ddd36` fix(scanner): match ignore patterns without interpreting workspace paths | Root glob-character regression. |
| `75e887e` docs: define configuration development and release workflow | README/contributor workflow and release setup. |
| Report commit (this file) | Engineering report and validation evidence. Exact hash in final git log. |

Use `git log --reverse --oneline 76879fb..HEAD` for the exact complete list, including documentation/report commits.

## K. Remaining technical debt

- **Corrective Pass 1B:** no listed bug is left knowingly unfixed. Independent review may identify further edge cases. Potential follow-ups include stronger fd-based move/copy confinement against adversarial concurrent ancestor replacement, explicit snapshot guarantees for concurrently edited source files, source/destination crash recovery, and more exhaustive async/interactive transition coverage. These are not represented as existing completed guarantees.
- **Pass 2 domain/features:** bounded Group/Subgroup model, path-based identity, five-favorite limit, sanitized project-name tmux-session collisions across roots, and metadata reconciliation after externally renamed paths. A permanent project-ID or recursive hierarchy redesign was intentionally deferred.
- **Pass 3 UI:** broad root model ownership, remaining large cohesive template editor/update/rendering routines, terminal output/Unicode/ANSI rendering fidelity, and exhaustive interactive accessibility/navigation coverage. Current visuals remain unchanged.
- **Pass 4 release hardening:** native arm64 smoke tests, real GitHub device-flow/private repository tests, real tmux nested layouts/attachment, interactive external scaffold/editor checks, first hosted CI/release rehearsal, artifact signing/provenance if desired, and stable-release criteria. Background descendant-process behavior under arbitrary long-lived scaffold commands merits further end-to-end stress testing.
- **Owner setup:** branch protections, permanent remote branches, official OAuth registration/public Client ID variable, and publication environment. Repository files cannot substitute for those actions, and this task explicitly prohibited unauthorized remote changes.

## L. Diff statistics

Final statistics relative to `76879fb`: **74 files changed, 7,017 insertions, 3,535 deletions**. Most apparent editor deletions are cohesive file extraction, not functionality removal. Tests, workflows, scripts, audit/report/logs, and documentation account for substantial additions. One unused index source file was removed.

Reproduce with `git diff --stat 76879fb HEAD`. Local binaries/checksums live in ignored `dist/` and are not committed. Build caches and detailed function-coverage data live outside the repository in temporary task directories.

## M. Readiness assessment

I consider the **Pass 1 repository implementation and local verification complete**: every listed bug has a correction and regression coverage, boundaries are materially clearer, error handling/destructive operations are safer, checks pass, and staging/production verification plus release generation are prepared. This assessment is subject to the independent review requested by the owner.

This is not a claim that a stable release has been published or that GitHub rules/OAuth registration have been configured. Owner setup and hosted/native/service end-to-end hardening remain explicit. Local verification builds omit a real official Client ID because none was supplied; production automation requires the owner-configured public variable. The first stable release remains gated on later passes and release-candidate hardening. No Pass 2 work was started.
