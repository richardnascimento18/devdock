# Pass 3F manual acceptance — required before merging

Status: **PENDING OWNER FEEDBACK**. This is a candidate, not an accepted pass.

Branch: `fix/pass-3f-ux-acceptance`. Target: `1.1.0-beta.11`.
The acceptance checkpoint supplies the exact candidate SHA and binary. Run that
binary directly, using your normal DevDock configuration, in your actual
Arch Linux + Caelestia transparent terminal. Do not replace the installed beta.10
until you are satisfied with the candidate.

## Mounted workspace move gate — mandatory

Use the actual `/run/media/.../devdrive/...` workspace where beta.10 failed.
Temporary-directory tests do not fulfill this gate.

- **A. Standalone:** select a project, `m`, move location A → B. Record PASS/FAIL and any error.
- **B. Bulk:** select multiple projects with Space, `m`, move to another location. Record PASS/FAIL and any error.
- **C. Collision:** attempt a move where the destination name already exists. It must reject the move and retain both projects.
- **D. State:** after success, destination content is intact, source is gone, and refresh/restart shows the new location.
- **E. Favorite/recent:** favorite and open a test project before moving; the favorite and recent reference must follow the successful move.

If either move still reports `invalid argument`, Pass 3F is not accepted.
Also inspect real legacy `.devdock` projects: blank/missing types should appear;
malformed or unknown explicit types should retain a readable warning in `!`.

## UX and visual gate

Reply briefly with PASS/FAIL or the changes you want:

- **Project list:** technologies immediately recognizable? Git/GitHub/local status immediately recognizable? Selected project obvious? Location visible but subordinate? Same-name projects distinguishable?
- **Workspace:** `Alt+1` reaches Workspace from Projects in one action; `Alt+2/3` reach Projects/Inspector; Tab still cycles. Backspace / Ctrl+U / Ctrl+A reach parent/root/all. `Ctrl+P`, `scope <query>`, Enter selects a group and returns to Projects. Check search and selection behavior.
- **Inspector:** useful for details, without requiring it for basic project identification?
- **Editors:** wide Presets / Windows / Properties and Templates / Steps / Properties keep context? Normal field edits stay in the detail pane? Medium two-column and narrow breadcrumb flows work after resize? Save/cancel/discard behavior predictable? Settings roots/default editing usable?
- **Tmux:** preset chooser has no raw ANSI fragments; default marker and long names render cleanly. Session names understandable; kill confirmations use readable project identity and target the correct session.
- **Warnings:** concise severity/count/action on dashboard; `!` reveals complete paths/errors.
- **Creation/palette/help:** create a project, inspect scope selection, search, multi-selection and Help; existing actions remain reachable.
- **Visuals:** Caelestia/wallpaper remains visible everywhere, including selected rows, modals and fields? Wordmark/full empty-state logo appropriate? Shimmer waits briefly and feels restrained? Reduced motion remains static?
- **Resize:** inspect at wide, medium and narrow dimensions during navigation and editing.

Explicitly state **“I approve this candidate for merge/promotion”** only when both
move and visual gates pass. Silence does not count as approval. Corrections stay
on this feature branch and repeat validation/acceptance.

## Owner feedback record

Candidate SHA: supplied at checkpoint / recorded in review-package `candidate.json`.
Mounted standalone result: pending.
Mounted bulk result: pending.
Collision / state / favorite / recent result: pending.
Legacy workspace marker result: pending.
Caelestia visual/transparency result: pending.
Requested corrections: pending.
Explicit approval: **NOT RECEIVED**.
