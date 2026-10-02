# TUI presentation architecture

## Baseline (Pass 2B)

The root `model` owns the loaded `core.Workspace`, project index, configuration,
persistence state, Bubbles list, flow enum, and screen drafts. `Update` dispatches
async results and screen handlers; the deferred tail schedules requested scans
and tmux refreshes. IDs reject stale async results. Preserve that contract.

`update_list.go` mixes navigation, search, root/preset selection, favorites,
recents, tmux and action entry points. Operation handlers are already separated
into creation, clone, move, group and root/delete files. `treelist.go` flattens
the snapshot; it must never trigger scans on navigation. Tree mode uses node
keys and bounded indentation; flat search includes location/root disambiguation.

`View` dispatches ~30 flow states. `screens.go` owns text inputs, destination/root
pickers, confirmations, OAuth and loading; `widgets.go` owns selectors/help.
The editor has collection, preset, template, settings and nested window/pane/step
layers; it edits deep copies and validates proposals before safe persistence.
Root actions return to Settings. PTY execution owns its own scrollable output.

Issues: fixed 62/70/76-column dialogs, list height subtracting 22 rows, an
eight-row wordmark, scattered colors, opaque selections/badges/help/OAuth/PTY
titles, duplicate key/help strings, fixed input widths, and unbounded help/forms.
Bubbles also supplies opaque list-title styles and inverse-video input cursors.
The loading spinner has operation IDs; input cursors otherwise create independent
blink loops. Main rendering does not perform external I/O.

## Target structure

`internal/ui` holds semantic theme, cell-aware layout, focus, shared frame/modal,
header/footer and text primitives. Keep this one package until a real boundary
requires another. Root dashboard/tree/inspector/palette/form adapters project
domain snapshots into components; components never invoke filesystem services.
Existing operation/persistence handlers remain authoritative.

`View` only selects a screen and bounds its output. Dashboard composition lives
outside it; screen updates route navigation separately from operation intents.
Bindings and contextual help share metadata. Modal state captures keys before
the dashboard. Escape closes modal, cancels search, clears selection, or returns
from a dedicated view, in that order; it never quits the main view.

## Responsive and focus rules

Minimum widths: workspace 24, projects 40, inspector 30, gutters 2. At 98 columns
use three panes, at 66 use workspace/projects, below 66 use one focused pane.
Inspector remains accessible as a dedicated pane at medium/narrow widths.
Below 24×8 show a bounded size message. Height reserves six chrome rows; allocate
the rest to visible content. Cap workspace/inspector/modal widths on huge screens.
Use ANSI display/grapheme width, never byte counts, for truncation and wrapping.

Tab/Shift+Tab cycle focus; focused titles use `>` and bold accent. In 3A the
workspace focus controls root scope and project focus controls the existing list.
`{`/`}` cycle roots directly, preserving an explicit fast shortcut. 3B expands
workspace focus into the recursive tree and adds inspector focus. `[`/`]` retain
collection tabs, `p`/`P` retain preset shortcuts, and existing action keys survive.

## Transparency and theme

There are **no backgrounds or filled surfaces**, including selected rows,
modals, blank padding, help, footer and headers. Semantic foreground roles:
Primary, Secondary, Accent, Muted, Faint, Success, Warning, Error, Info, Border,
BorderFocused, Selection, Favorite, Git and Tmux. Primary uses terminal default;
adaptive foregrounds accommodate light/dark defaults and degrade with Lip Gloss.
NO_COLOR removes color; focus/selection/error symbols remain meaningful.

Configure dependency defaults explicitly. Input cursors use an underline rather
than inverse video. `TestPresentationStyleInvariant` scans first-party Go style
calls; rendered tests inspect SGR including 40–47, 100–107, 48 indexed/RGB and
inverse video. Reset/default-background 0/49 are permitted. Never strip output
to make the test pass. Subprocess PTY content is sanitized by the existing parser.
Actual wallpaper continuity still needs compositor review; ANSI fixtures and
isolated real-terminal smoke provide automated evidence, not that observation.

## Subsequent milestones

3B implements `workspace_tree.go`, `dashboard.go`, `inspector.go`, `search.go`,
`palette.go`, `dashboard_navigation.go` and a two-line project delegate. Container
rows are cached from the loaded snapshot; project membership uses the chosen
root/domain/group scope and is independent of workspace expansion. Collapse
retains the selected project. Tab cycles all three panes; medium Inspector and
narrow focus become dedicated views. `v` hides/shows the workspace composition.
Live fuzzy search uses the existing small `fuzzy` dependency, preserves matching
selection, supports coalesced slash/query input, and restores the prior query on
cancel. Palette/help share action metadata; disabled actions state their reason.
Inspectors only render loaded facts and scroll wrapped full paths; stale recents
need a canonical loaded project before favorite/move/delete actions are enabled.
Workflow screens share `modalContent` and responsive `ModalAt`: full paths wrap,
PgUp/PgDown recover long details, hints stay visible, and resize clamps scrolling.
Inputs and pickers capture focus; no modal paints a backdrop.
3C: project-key selection, whole-batch move preflight and honest partial results.
3D: editor presentation, validation, draft/unsaved handling; keep persistence.
3E: one on-demand animation clock; foreground-only progress shimmer; environment
reduced motion; final Unicode/resize/PTY/golden/performance diagnostics.
Animation never drives operations; stale ticks are ignored and idle UI stops
scheduling. Avoid new persisted UI preferences unless clearly necessary.
