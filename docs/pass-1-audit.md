# Pass 1 initial audit

Baseline: `76879fb` (main). All tracked files and package implementations were inspected before substantial edits. The baseline builds and `go test ./...` succeeds, but every package reports no test files. No CI, release automation, test suite, or contributor workflow exists. No TODO/FIXME markers exist; incompleteness is behavioral rather than labelled.

## Responsibilities and findings

- Root package: Bubble Tea model, screen routing, list/tree rendering, editor, scaffold orchestration, filesystem moves and groups. `editor.go` is 1,895 lines combining four editors and another atomic writer. Update contains filesystem copying and synchronous OAuth requests. Global detector adapters are mutable. Name/GitHub indexes are rebuilt but never consumed.
- `core`: project/domain operations and scanning. Creation uses `MkdirAll`, deletion accepts root equality, lexical containment ignores intermediate symlinks, cross-root scanning suppresses failures. Index persistence writes an unused cache and ignores errors. Group/Subgroup depth is bounded and must remain unchanged in this pass.
- `detect`: marker/dependency/extension registry plus global scan caches. .csproj/.fsproj are exact names, F# is mislabelled, Solid Start and Fastify are misassociated. An impossible string condition exists. Unknown/invalid markers and unreadable directories are silently classified. Equal extension counts have nondeterministic ordering.
- `pty`: `cmd.Wait` result is discarded on EOF. Input schedules overlapping reads. Close does not ensure child termination. Template post-step transition emits output but schedules no next step. Control bytes can trap the output parser in an infinite loop.
- `template`: parsing uses `strings.Fields`; variables with spaces split incorrectly. Destructive builtins accept `.` and intermediate symlinks. Defaults contain a shell step forbidden by their own loader, so generated defaults cannot round-trip. Creation permits existing destinations. Validation appends into input slice capacity.
- `preset`: useful existing individual/collection validation, but editor validates only an individual object. Pane size and normalized tmux window-name collisions need validation.
- `github`: HTTP timeout exists but no cancellation lifecycle. Start-device request blocks Update. Polling sleeps without context and hides transport failures. Errors include raw response bodies. Git subprocesses live alongside HTTP/auth; README write failure is ignored. Remote parser accepts substring hosts. Only a public OAuth Client ID is required; no client secret is present.
- `tmux`: creation/split/send/select/attach errors discarded. Pane target construction is incorrect and layout is overwritten with tiled. No fake execution boundary. Session names can collide across roots (future identity design deferred).
- `config`, `state`, `fileutil`: atomic writers duplicated; no file sync. Config migration writes during Load and ignores failure. Malformed config triggers first-run wizard, risking replacement. Roots/default/auth mutate before Save. State Load/Save suppress errors and may return partially decoded state. Favorites/recents do not follow moves/deletes, recents ignore active root.
- UI/widgets/view: empty root pickers can panic; clone-new-domain routes into project creation; stale pending group/domain/repo intent can leak between workflows; spinner ticks forever; resize is globally consumed before PTY/editor receive it; flat/collapsed tree picker collision detection inspects presentation items rather than projects. Tree list rebuild reads filesystem repeatedly and ignores `.ddignore` for empty groups.
- Documentation/build: README claims Go 1.21+, `go.mod` specifies 1.25.6; dependency metadata needs deliberate verification. No build metadata or version command. Download claims are premature.

## History

The latest commit adds the whole editor and tmux management in one 2,121-line change. That explains duplicated persistence and incomplete draft/save synchronization. Earlier README-only commits do not alter runtime behavior. No published history will be rewritten.

## Plan and boundaries

Incremental milestones: workspace/scanning safety; process/template correctness; persistence/editor transactions; external integrations and lifecycle; UI/cache/state correctness; pinned validation/release workflow and documentation. Protect each milestone with behavioral tests before local Conventional Commits. Preserve existing bounded Group/Subgroup model, visuals and feature scope. No remote actions or release publication.
