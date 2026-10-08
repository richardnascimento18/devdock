# DevDock

DevDock is a keyboard-driven terminal workspace manager for Linux. It discovers projects, launches tmux workspaces with saved layouts, scaffolds projects from templates, and optionally connects to GitHub through its device authorization flow.

The workspace hierarchy is `Root → Domain → Project or Group`. Groups can contain projects and further groups at any depth. DevDock imposes no hierarchy depth limit; filesystem and operating-system resource limits still apply. The responsive interface keeps workspace scope, project identification, and project details in separate panes.

## Requirements and installation

- Linux amd64 or arm64. Linux-specific filesystem and PTY handling is intentional.
- Go **1.26.9+ or 1.27.2+** within those supported series for source builds. See the [compiler policy](CONTRIBUTING.md#compiler-security-policy).
- tmux for opening workspaces; Git for GitHub cloning/linking. Template commands and preset commands require their own tools, such as Node.js, Python, Cargo, nvim, or opencode.
- Python 3 and Bash for development validation and release scripts.

Linux binaries and checksums are distributed through the [GitHub releases page](https://github.com/richardnascimento18/devdock/releases). The release pipeline publishes explicit beta/prerelease versions until stable-release hardening is complete. A downloaded binary needs no Go installation. Download the matching `devdock_<version>_linux_amd64` or `devdock_<version>_linux_arm64` and `SHA256SUMS`, verify the matching checksum, then install it:

```sh
sha256sum --ignore-missing --check SHA256SUMS
mkdir -p ~/.local/bin
install -m 755 devdock_<version>_linux_amd64 ~/.local/bin/devdock
```

Use the arm64 filename on arm64. Ensure `~/.local/bin` is in your PATH.

For a source build:

```sh
git clone https://github.com/richardnascimento18/devdock.git
cd devdock
go build -o devdock .
./devdock --version
./devdock
```

First run asks for an existing workspace root and saves configuration. A malformed existing config produces an error rather than being silently replaced. Projects are scanned from each root's domain directories; unreadable/missing roots are reported while available projects remain usable.

## Current functionality

| Key | Action |
| --- | --- |
| Enter | Open a project with the selected tmux preset |
| `p` / `P` | Switch/select a preset |
| `v` | Toggle tree/flat project view |
| Tab / Shift+Tab | Cycle Workspace / Projects / Inspector |
| Alt+1 / Alt+2 / Alt+3 | Focus Workspace / Projects / Inspector directly |
| Backspace / Ctrl+U / Ctrl+A | Parent scope / current root / all workspaces (outside text entry) |
| Ctrl+P | Command palette; type `scope <query>` to select a workspace/group |
| `/` | Search projects within the current scope |
| Space | Select projects for bulk move/favorite |
| `[` / `]` | Switch projects, recents, favorites, and tmux tabs |
| `f` | Toggle a favorite (unlimited) |
| `n` / `N` | Create a project/domain |
| `m` | Move a project between roots/domains/groups at any depth |
| `x` / `X` | Delete a project/domain |
| `G` / Ctrl+G | Create/delete a group at any depth |
| `a` / `A` | Add/remove a root |
| `e` | Open the configuration, preset, and template editor |
| `g` | Connect GitHub, or refresh connected repositories |
| Esc | Cancel the current flow |
| `q` | Quit from the main screen |

Recursive project, domain, and group deletion requires typing the full target path. Project creation and moves reject existing destinations; moves never merge directories. Filesystems without `renameat2(RENAME_NOREPLACE)` use a portable rename after a fresh destination check. That fallback cannot provide atomic no-replace against an external concurrent writer; concurrency hardening remains deferred to Pass 4. A cross-filesystem move preserves symlinks and reports incomplete source cleanup as a partial failure. Favorites/recents follow successful moves and are pruned after successful deletions; explicitly removing a configured root also clears its favorites/recents, while temporarily unavailable configured roots retain their state.

Built-in presets are `walker`, `nvim`, and `dev-split`. Templates include Next.js, React + Vite, Go API/CLI, Python FastAPI/Script, Rust binary/library, Node.js, and Static HTML. Scaffolding failures stop subsequent steps and prevent a successful workspace launch. Templates execute trusted configured commands, including interactive commands in a PTY.

## Configuration and markers

Configuration lives in `~/.config/devdock/`:

| File | Purpose |
| --- | --- |
| `config.toml` | Absolute workspace roots, default preset, GitHub token and username |
| `presets.json` | Named tmux windows and recursive pane layouts |
| `templates.json` | Named scaffold steps, post-steps, and layout metadata |
| `state.json` | Favorites, recents, active tab, view/collapse preferences |

Missing preset/template files generate validated defaults. Invalid collections are reported and built-in defaults remain available without overwriting the invalid file. Editor saves validate the entire proposed collection, persist atomically, and then change active state. Duplicate names are rejected. Renaming the configured default preset updates its configuration reference as part of the save; a write failure preserves the draft and compensates the configuration change.

Example config without GitHub credentials:

```toml
roots = ["/home/user/projects", "/mnt/work/projects"]
default_preset = "nvim"
```

The editor opened with `e` has **Presets**, **Templates**, and **Settings** tabs. Settings edits the existing default preset (an existing preset name, or blank to select the first preset) and invokes the existing add/remove-root workflows. Root removal prunes associated favorites, recents, and collapsed nodes; unavailable roots remain configured until deliberately removed. A configuration save clones the proposal, validates the whole config, writes atomically, and then replaces live state. Failed saves preserve the draft and previous config; cancellation and unchanged saves do not write. Default-preset changes take effect immediately and survive restart. Ignore patterns remain in `.ddignore`; view preferences remain in `state.json`. At 110+ columns, preset/template editors retain collection, windows/steps, and properties in three columns. At 74–109 columns they use collection + detail; narrower terminals keep contextual page flows. Normal property and pane edits update the detail area. `Alt+1` browses the collection while retaining the active draft; Enter/Esc asks before discarding unsaved changes. `Alt+2` accepts validated window/step fields and returns to items, and `Alt+3` resumes properties. Save/cancel and full-collection validation retain their existing semantics. Settings root actions also retain their section context at 74+ columns. The editor does not display or edit OAuth credentials or internal values.

The configuration file contains a GitHub access token after authorization. It is saved with mode `0600`; do not commit it or attach it to issue reports.

Configured roots must own separate workspace trees: exact duplicates and parent/child roots are rejected using path components, so `/code` and `/code2` remain valid siblings. Existing paths are also compared after symlink resolution to reject physical aliases and physical parent/child overlap. Unavailable, broken-link, or inaccessible roots remain configured when physical identity cannot be resolved; lexical checks still apply, and scan warnings report unavailable workspaces. An old config containing a definite conflict produces a startup error identifying both roots and the config file. DevDock leaves that file unchanged; edit its roots list and restart. Add-root actions in the list and Settings use the same validation. Explicit root removal retains the existing state cleanup and does not delete the directory.

A `.devdock` TOML marker can explicitly identify a `project` or `group`; project markers may set a display `name`. Valid legacy markers with a missing or blank `type` are projects, including markers with other historical fields. `type = "subgroup"` remains a group alias. Malformed TOML and unknown explicit nonblank types produce scan warnings. Markers are never rewritten automatically. `.ddgroup` takes precedence and retains empty groups. Unmarked directories become groups when their descendants contain projects or explicit groups. A project is a discovery boundary: its source tree is never scanned as workspace hierarchy. Language markers and Git repositories identify projects. Hidden directories and common source/dependency/output directories (`src`, `node_modules`, `vendor`, `build`, `dist`, `target`, `__pycache__`, `venv`) are excluded as implicit workspace entries; explicit project/group markers override those descendant name exclusions. Hidden root entries remain excluded as domains. Directory symlinks below roots are not traversed, preventing cycles, escapes and alias discovery; configured roots may themselves be symlinks.

Root `.ddignore` files contain root-relative glob patterns, one per line. Existing domain-name patterns retain their meaning; nested patterns such as `apps/backend/archived` exclude that directory and its descendants. Patterns use Go `filepath.Match` semantics (`*` does not span separators; there is no special `**`). Blank lines and `#` comments are ignored; invalid patterns produce a scan warning.

State schema 2 replaces the old group/subgroup collapse maps with full-location node keys. Existing favorites, recents, tab and view preferences migrate on load and persist on the next successful atomic save. Malformed, ambiguous or future-schema state is reported and protected from overwrite. Projects use their absolute path as their state key; marker display names are separate. DevDock moves remap favorites and recents while preserving recent timestamps and order. External filesystem moves cannot reliably be recognized. Recent history retains at most 50 entries. If a state write fails after a filesystem operation, the new references stay in memory for a later save attempt, and the failure is reported.

Tree indentation is capped at four visual levels. Full logical ancestry remains in the snapshot, and compact breadcrumbs plus the selected location distinguish deep entries. Search operates on the loaded flat project index even when groups are collapsed. Location pickers scroll through domains and every nested group, with a stable path hash to distinguish breadcrumbs that truncate to the same suffix. Technical discriminators in project rows appear only when compact same-name locations collide; deletion screens display the complete wrapped target path. Root pickers show full paths rather than ambiguous basenames. Group creation accepts a slash-separated path of validated components. No navigation/filter/collapse action rescans the filesystem.

Tmux project sessions use `devdock-<directory-basename>-<16-hex-path-hash>`, avoiding same-name collisions across groups and roots. Display-name changes keep the session identity; path changes create a new identity. Project display names and locations identify current sessions in the UI. Kill confirmation uses the display name, or a readable full project path for duplicate names, while the backend still targets the exact hashed session. Technical IDs remain available in the Inspector. Existing legacy sessions remain accessible in the tmux-sessions tab; DevDock does not automatically attach to ambiguous old names or kill them.

Template step commands support quoted arguments such as `command --title "Hello world"`. They are parsed into arguments and executed directly, without shell expansion. Variables are `{{project_name}}`, `{{project_path}}`, `{{domain}}`, and `{{root}}`. Shell operators, `$()`, environment expansion, and glob expansion are not implicitly evaluated. Use the step's `output` field to capture stdout into a project-relative file instead of `>` redirection. Builtin `touch`, `mkdir`, and `rm`, and output paths, must remain strict descendants of the project and cannot traverse symlinks or `..`; `rm .` is rejected. Explicit shell-mode steps are unsupported. Custom commands remain trusted executable code; this confinement is for builtin filesystem actions, not an OS sandbox for arbitrary commands.

## GitHub authorization and custom builds

DevDock uses a GitHub **OAuth App** with **device flow enabled**. The public `client_id` identifies the application; this flow does not require a client secret. DevDock accepts no OAuth client-secret setting and does not compile one into its binary. See [GitHub's device-flow documentation](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps).

Official release builds obtain the public Client ID from the repository variable `DEVDOCK_GITHUB_CLIENT_ID` and embed it with a linker flag. The official public Client ID is configured in that repository variable and was embedded in the CI-built `v1.1.0-beta.1` release. No Client ID is hardcoded into source; no Client Secret is used.

To use your own OAuth app, create it in GitHub Settings → Developer settings → OAuth Apps, enable device flow, and copy its **Client ID**. Clone DevDock and build:

```sh
go build -o devdock \
  -ldflags "-X github.com/richardnascimento18/devdock/internal/github.ClientID=YOUR_PUBLIC_CLIENT_ID" .
```

Alternatively set a runtime override:

```sh
DEVDOCK_GITHUB_CLIENT_ID=YOUR_PUBLIC_CLIENT_ID ./devdock
```

The runtime value takes precedence over the embedded ID. Building without an ID leaves local workspace features available and shows an explicit error when authorization is requested. Auth cancellation stops polling, and late responses cannot connect an abandoned flow. Once connected, DevDock lists repositories, matches GitHub remotes, creates/pushes new repositories, and clones into collision-checked destinations.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development checks, package ownership,
and the protected staging/production workflow.

Licensed under [MIT](LICENSE).
