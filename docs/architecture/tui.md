# TUI ownership and rendering

The root package is the Bubble Tea adapter. `Update` handles typed completion
messages, sizes presentation, and delegates input to the active route. `View`
selects bounded presentation and never performs external I/O. Commands capture
requests and dependencies before starting work; IDs reject stale completions.

## State and routing

The model groups navigation, search, creation drafts, launch requests, OAuth,
scans, repository loading and tmux snapshots by lifecycle. Navigation owns the
loaded recursive workspace, scope, focus and derived row/index caches. Creation
starts from a fresh draft. Bulk and group workflows retain explicit phases.

One exclusive route value identifies the active screen or workflow. Its route
class identifies dashboard, overlay, operation, editor, terminal, help or palette
ownership. This preserves existing transitions without independent screen/modal
flags that could consume the same key. Operations reject dashboard action keys;
Escape retains the established cancellation behavior of each workflow.

`route_input.go` owns input dispatch; command adapters own external requests.
Workspace operations return application results. Scanning, creation, domain/group
creation, moves, clone and deletion run outside input handlers. Short preference
commits use injected stores synchronously, preserving validation, rollback and
ordering. They are a remaining latency boundary; changes to asynchronous save
ordering require separate durability and failure-transition tests.

## Presentation components

Dashboard, recursive workspace tree, inspector, search and palette render loaded
snapshots. Collection membership follows scope independently of tree expansion.
Collapse preserves the selected project. Navigation and animation never rescan.
The tmux inspector labels its loaded session snapshot as cached.

Editors own detached drafts, dirty-state checks and nested navigation; application
preferences coordinate accepted proposals and store failures. `ptyScreen` owns
keyboard translation, scrolling, logs and terminal presentation. Application
scaffolding owns sequence/finalization and infrastructure owns processes.

`internal/ui` owns semantic foreground styles, responsive layout, shared
frames/modals, display-width clipping, input presentation and technology badges.
Detection returns semantic labels. Diagnostics remain plain data until rendered.

## Responsive navigation

At 98 columns the dashboard has three panes; at 66 it has workspace/projects;
below 66 it shows the focused pane. The inspector remains accessible at every
supported width. Below 24×8, show a bounded size message. Full logical group
depth survives the capped visual indentation. Wrapped details and modals scroll.

Tab/Shift+Tab cycle pane focus. `{`/`}` switch root scope directly, `[`/`]` switch
collections, and `p`/`P` switch presets. Palette/help use shared action metadata.
Existing project, group, editor and search keys retain their meanings.

## Terminal safety and transparency

Apply `ui.SafeText` or `ui.SafeBlock` to external data before trusted styling.
`ui.InputView` renders a detached copy without modifying the editable identity.
Animated labels obey the same rule. Clip styled output by grapheme/display width;
never run the sanitizer over an already styled composition.

All first-party surfaces use foreground colors, borders, effects and spacing.
No pane, modal, selected row or cursor uses a background or reverse video.
Input cursors use underline. Tests inspect style calls and rendered SGR, including
indexed/RGB and colon forms. Wallpaper continuity needs real compositor review.

A single generation-checked animation clock runs only while activity is visible.
`DEVDOCK_REDUCED_MOTION=1` (also true/on), nonempty `NO_COLOR`, or `TERM=dumb`
selects static progress without changing operation execution.

## Regression evidence

Tracked ANSI/plain render pairs cover responsive layouts, recursive trees,
collections, modal workflows, editors, OAuth, PTY and motion behavior. Resize,
navigation, adversarial display data and stale messages have behavioral tests.
Isolated PTY smoke exercises actual terminal input and restoration; fixtures
cannot substitute for owner checks of the desktop and real external workflows.
