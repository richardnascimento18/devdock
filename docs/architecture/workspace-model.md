# Workspace model (Pass 2)

## Current problems

Projects currently carry Root, Domain, Group and Subgroup strings. Scanner
classification recursively searches descendants, then collection and group
pickers search them again. Depth limits (five for projects, three for group
pickers) omit projects and intermediate ancestry. Separate collapse maps and
root/domain/name tmux sessions collide or cannot represent deeper locations.

## Target and invariants

A Location contains an absolute configured Root, one Domain component, and an
ordered GroupPath of validated name components. Project embeds Location and
retains its display Name and current Path; the directory basename may differ
from a marker's display name. There is no special nested-group type.

The canonical workspace snapshot contains recursive Root/Domain/Group nodes
and a flat project index produced by the same scan. Navigation, destinations,
search and GitHub linking consume that snapshot, never rescan on selection or
collapse. Generic flattening preserves logical depth; rendered indentation is
capped at four levels and names precede compact location breadcrumbs.

## Discovery

Visit each relevant directory once. Local explicit markers and language
markers establish project boundaries; never traverse a project's source tree.
`.ddgroup` takes precedence and explicit `.devdock` group markers retain empty
groups. Legacy `.devdock` type `subgroup` is accepted only as a group marker at
the input boundary. Unmarked containers are retained as groups when their
single traversal discovers workspace children. Hidden/tool/output directories
are skipped as implicit hierarchy entries (explicit descendant markers override this). Root `.ddignore` patterns remain
root-relative, extending matching to nested paths without changing domain
matching. Malformed markers, inaccessible directories and invalid ignore files
produce contextual partial-scan warnings. Directory symlinks below roots are
not traversed, so aliases, escape links and cycles cannot duplicate discovery.
Configured roots themselves may remain symlinks. No application depth limit.

## Operations and identity

Core Location resolution owns validation and containment. Create, template,
clone, group and move workflows pass typed locations. A move separates
preflight planning, execution (with revalidation), and state reconciliation.
Exclusive rename/cross-device copy and rooted deletion retain Pass 1 safety.
Partial moves retain both paths and report the cleanup failure.

Project identity is the cleaned absolute Path (a defined ProjectKey), not the
display name. NodeKey encodes a typed Location without delimiter ambiguity;
root, domain and every full group path are distinct. No metadata is written to
discovered repositories to invent stable IDs. DevDock moves remap favorites
and recents to the resulting location, preserving recent order and timestamps.
External moves cannot reliably be recognized; unavailable configured roots do
not trigger destructive pruning. Deliberate root removal prunes its entries.
Favorites are unlimited; the existing fifty-entry recent history remains.

Tmux names use a readable sanitized basename plus a short SHA-256 path hash.
Existing legacy sessions remain available in the sessions tab; automatic
launch does not attach to potentially ambiguous old names or kill sessions.
A location change gets a new session identity; existing sessions remain intact.

## Persistence migration

Version state.json. Read old collapsed_groups/collapsed_subgroups only in a
legacy decoding structure, mapping `root::domain::group[::subgroup]` to full
Location NodeKeys. Preserve favorites (absolute-path keys), recents, tab and
view state. Infer recent GroupPath from its path and root/domain, rather than
trusting truncated old ancestry. Reject malformed or future schemas without
writing the input file; migration is deterministic and idempotent, persisted
atomically on the next successful state save. Ambiguous legacy delimiter keys
cannot be recovered safely and must surface an error, retaining the disk file.
