# Pass 3B — workspace dashboard

Branch: `feat/workspace-dashboard`, from reconciled staging
`1e00eb6696b1db93ccff25cdb20a158bc5a7074d`. Production beta.6 is its ancestor,
with identical trees and only permanent remote branches at the boundary.

## Scope

Cached recursive workspace navigation, separate project rows, contextual
inspector, responsive three/two/one-pane composition, compact header/footer,
live location-aware fuzzy search, contextual command palette and grouped help.
Tab/Shift+Tab cycle workspace/projects/inspector; inspector becomes a dedicated
view below the wide breakpoint. Tree guides cap indentation while full logical
location remains available. Same-name projects include root/location identity.
Filtering, focus and expansion use the loaded snapshot; rendering performs no
filesystem or network operations. Existing actions remain routed through their
approved operation handlers. No domain/persistence behavior was replaced.

Workflow forms now share transparent, responsive modals with retained hints,
wrapped full paths and bounded PgUp/PgDown detail scrolling. Stale recent entries
can open, but require a canonical loaded project for mutation. Palette and help
share action metadata and explain unavailable actions.

## Validation

- Full check script, exact Go 1.26.0 tests, amd64/arm64 release builds,
  SHA256SUMS and expanded binary PTY smoke passed. Logs: [pass-3b](pass-3b/).
- 117 top-level Go tests, 267 tests/subtests, 8 Python tests; 22 paired
  ANSI/plain render fixtures. Race/vet/staticcheck/govulncheck/actionlint passed.
- Interaction coverage: focus and resize, snapshot-only scope/collapse,
  search preservation/cancellation/location, palette execute/disable/cancel,
  deep Unicode inspector and full-path confirmation scrolling/cancellation.
- Source-style and emitted-ANSI checks still reject explicit backgrounds and
  inverse video across terminal color profiles and default light/dark modes.
- PTY smoke covers workspace, inspector, search, palette, Settings persistence,
  preset/template forms, help, OAuth entry/cancel, six sizes and shutdown.
- Rendering 1,000/5,000 projects: ~270 μs/frame and ~100 KB; fuzzy filtering:
  ~0.71/3.59 ms and ~127/757 KB on this workstation. Diagnostic, no timing gate.

Two bugs found during validation: coalesced slash/query terminal input skipped
search entry, and modal padding caused a second wrap that hid the footer.
Both are fixed with focused transition/render regressions. Long text uses
terminal cell widths; full paths remain recoverable by scrolling.

The existing small fuzzy dependency becomes direct; no dependency version added.
Compositor wallpaper review remains pending the user's final inspection;
headless ANSI/PTY evidence does not claim a manual Caelestia review.

## Promotion

Target prerelease: `v1.1.0-beta.7`. Protected feature → staging → production
PRs, hosted CI, actual downloaded release checks and production → staging
reconciliation must all complete before beginning 3C. Live evidence is preserved
for the final independent-review package.

## Remaining Pass 3 work

Bulk workflows (3C), editor redesign (3D), shared motion/reduced motion and final
polish/coverage (3E). Pass 4 backlog remains separate.
