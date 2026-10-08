# Application and infrastructure boundaries

DevDock keeps its Bubble Tea adapter in the root package. `internal/app`
coordinates cohesive workspace, preference and scaffold operations using plain
Go requests/results. Infrastructure packages never return Tea commands/messages.
The adapter captures requests in commands and translates completion events.

## Paths and stores

Startup resolves `app.Paths` once: home, configuration directory, config, state,
presets, templates and reserved journal paths. Production constructs bound stores;
compatibility wrappers remain available but application logic does not resolve
HOME repeatedly. Tests bind stores to temporary paths independently of HOME.

Config/state/preset/template packages own schemas, validation, serialization and
atomic single-file writes. `app.Preferences` coordinates config proposals, default
preset renames and root removal with explicit compensation on later write failure.
Narrow writer interfaces permit deterministic failures. Editor proposals become
active only after acceptance. State reconciliation after a completed filesystem
mutation remains pending on write failure, rather than pretending the filesystem
rolled back. Directory durability and multi-file crash reconciliation remain
separate hardening work; no journal or transaction manager is implemented.

## Workspace operations and repositories

`app.Workspaces` coordinates snapshots, project creation, cloning and repository
association. Explicit filesystem functions and a local Git client support failure
injection. `app.Move` and bulk planning/execution own preflight and revalidation
sequencing; `internal/core` remains authoritative for containment, collisions,
portable moves, partial results and recursive workspace discovery.

`internal/git` owns local Git commands, repository detection and canonical remote
parsing. Repository discovery asks Git, supporting worktree `.git` files and
rejecting accidental parent repositories. `internal/github` owns HTTP/API and
OAuth. Association uses canonical remote owner/repository identity, never the
local folder name. Exhaustive remote and OAuth protocol matrices remain separate
integration hardening; package extraction does not redefine their semantics.

## Scaffolding and terminal transport

`template.Execution` owns step/post-step sequencing. `PrepareStep` owns argument,
variable and output-path preparation; `ExecutePrepared` owns builtin and ordinary
command execution. `app.Scaffold` chooses ordinary/output execution or interactive
PTY transport, then finalizes the project marker and optional Git initialization.
Both execution paths share preparation and sequencing while retaining different
I/O needs. Commands with output files never use an interactive PTY.

`internal/pty` owns session creation, reading, writing, resizing, exit status and
cleanup. Events are plain Go values. `internal/terminal` interprets a bounded
stream, including fragmented UTF-8, cursor/redraw operations and discarded OSC
payloads. It never forwards subprocess escape sequences to the owner's terminal.
The root adapter renders interpreted text using the safe-display boundary.

## Processes and tmux

`internal/process` supplies context-aware command construction, process-group
cancellation, a bounded wait policy and bounded captured output. PTYs retain
session-specific lifecycle handling. Root signal cancellation reaches Git,
GitHub, tmux and scaffolding. Exhaustive descendant/signal behavior needs separate
real-process tests; process groups alone are not a universal sandbox.

The tmux client owns execution against an explicit context, runner and optional
socket. Pure launch planning captures a preset before execution. Launch errors
report the session and whether this attempt created it; they never destroy an
existing session. Noninteractive command output is captured, while attach/switch
operations retain terminal interaction after the TUI shuts down. Tests use fake
runners or isolated executables and never access the normal tmux server.

## Diagnostics and tests

`app.Diagnostic` retains severity, summary, detail, cause and operation without
ANSI. Presentation chooses colors, paging and clipping. Technology detection
returns semantic metadata; `internal/ui` maps that metadata to badges.

`internal/testutil` provides temporary Git repositories/identities, controlled
executables and writer adapters. Fault tests target observable effects, including
retained drafts, compensation, cancellation, partial moves and late completion.
Full logs and machine-specific execution evidence belong in ignored
`/.devdock-work/`, rather than tracked architecture documentation.
