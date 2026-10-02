# DevDock

DevDock is a keyboard-driven terminal workspace manager for Linux. It discovers projects, launches tmux workspaces with saved layouts, scaffolds projects from templates, and optionally connects to GitHub through its device authorization flow.

The workspace hierarchy is `Root → Domain → Project or Group`. Groups can contain projects and further groups at any depth. DevDock imposes no hierarchy depth limit; filesystem and operating-system resource limits still apply. The current interface supports this model without a visual overhaul.

## Requirements and installation

- Linux amd64 or arm64. Linux-specific filesystem and PTY handling is intentional.
- Go **1.26 or newer** for source builds; CI tests the latest patches of Go 1.26 and 1.27. The previous Go 1.21 claim was incorrect.
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
| Tab / Shift+Tab | Switch roots, including all roots |
| `[` / `]` | Switch projects, recents, favorites, and tmux tabs |
| `f` | Toggle a favorite (unlimited) |
| `n` / `N` | Create a project/domain |
| `m` | Move a project between roots/domains/groups at any depth |
| `x` / `X` | Delete a project/domain |
| `G` / Ctrl+G | Create/delete a group at any depth |
| `a` / `A` | Add/remove a root |
| `e` | Open the preset/template editor |
| `g` | Connect GitHub, or refresh connected repositories |
| Esc | Cancel the current flow |
| `q` | Quit from the main screen |

Recursive project, domain, and group deletion requires typing the full target path. Project creation and moves reject existing destinations; moves never merge directories. A cross-filesystem move preserves symlinks and reports incomplete source cleanup as a partial failure. Favorites/recents follow successful moves and are pruned after successful deletions; explicitly removing a configured root also clears its favorites/recents, while temporarily unavailable configured roots retain their state.

Built-in presets are `walker`, `nvim`, and `dev-split`. Templates include Next.js, React + Vite, Go API/CLI, Python FastAPI/Script, Rust binary/library, Node.js, and Static HTML. Scaffolding failures stop subsequent steps and prevent a successful workspace launch. Templates execute trusted configured commands, including interactive commands in a PTY.

## Configuration and markers

Configuration lives in `~/.config/devdock/`:

| File | Purpose |
| --- | --- |
| `config.toml` | Absolute workspace roots, default preset, GitHub token and username |
| `presets.json` | Named tmux windows and recursive pane layouts |
| `templates.json` | Named scaffold steps, post-steps, and layout metadata |
| `state.json` | Favorites, recents, active tab, view/collapse preferences |

Missing preset/template files generate validated defaults. Invalid collections are reported and built-in defaults remain available without overwriting the invalid file. Editor saves validate the entire proposed collection, persist atomically, and then change active state. Duplicate names are rejected.

Example config without GitHub credentials:

```toml
roots = ["/home/you/projects", "/mnt/work/projects"]
default_preset = "nvim"
```

The configuration file contains a GitHub access token after authorization. It is saved with mode `0600`; do not commit it or attach it to issue reports.

A `.devdock` TOML marker can explicitly identify a `project` or `group`; project markers may set a display `name`. Legacy `type = "subgroup"` markers are read as ordinary groups. `.ddgroup` takes precedence and retains empty groups. Unmarked directories become groups when their descendants contain projects or explicit groups. A project is a discovery boundary: its source tree is never scanned as workspace hierarchy. Language markers and Git repositories identify projects. Hidden directories and common source/dependency/output directories (`src`, `node_modules`, `vendor`, `build`, `dist`, `target`, `__pycache__`, `venv`) are excluded as implicit workspace entries; explicit project/group markers override those descendant name exclusions. Hidden root entries remain excluded as domains. Directory symlinks below roots are not traversed, preventing cycles, escapes and alias discovery; configured roots may themselves be symlinks.

Root `.ddignore` files contain root-relative glob patterns, one per line. Existing domain-name patterns retain their meaning; nested patterns such as `apps/backend/archived` exclude that directory and its descendants. Patterns use Go `filepath.Match` semantics (`*` does not span separators; there is no special `**`). Blank lines and `#` comments are ignored; invalid patterns produce a scan warning.

State schema 2 replaces the old group/subgroup collapse maps with full-location node keys. Existing favorites, recents, tab and view preferences migrate on load and persist on the next successful atomic save. Malformed, ambiguous or future-schema state is reported and protected from overwrite. Projects use their absolute path as their state key; marker display names are separate. DevDock moves remap favorites and recents while preserving recent timestamps and order. External filesystem moves cannot reliably be recognized. Recent history retains at most 50 entries. If a state write fails after a filesystem operation, the new references stay in memory for a later save attempt, and the failure is reported.

Tree indentation is capped at four visual levels. Full logical ancestry remains in the snapshot, and compact breadcrumbs plus the selected location distinguish deep entries. Search operates on the loaded flat project index even when groups are collapsed. Location pickers scroll through domains and every nested group, with a stable path hash to distinguish breadcrumbs that truncate to the same suffix. Selected rows show the same discriminator; deletion screens display the complete wrapped target path. Root pickers show full paths rather than ambiguous basenames. Group creation accepts a slash-separated path of validated components. No navigation/filter/collapse action rescans the filesystem.

Tmux project sessions use `devdock-<directory-basename>-<16-hex-path-hash>`, avoiding same-name collisions across groups and roots. Display-name changes keep the session identity; path changes create a new identity. Existing legacy sessions remain accessible in the tmux-sessions tab; DevDock does not automatically attach to ambiguous old names or kill them.

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

## Development and releases

```sh
GOBIN=/tmp/devdock-tools make tools
GOBIN=/tmp/devdock-tools make check  # uses GOBIN on PATH; install ShellCheck for CI parity
make check       # formatting, module consistency, version/workflow validation, tests,
                 # race tests, vet, staticcheck, govulncheck, build
make test
make race
```

All tests use temporary workspaces and fakes for GitHub/tmux/editor boundaries. PTY tests run small local child processes. Race tests require Linux, CGO, and a C compiler; release binaries are built with CGO disabled. `govulncheck` needs network access to the advisory database.

`VERSION` is an explicitly reviewed Semantic Version. The hosted beta lineage preserves historical `v1.0.0-beta` and `v1.1.0-beta`; the CI/CD rehearsal published [`v1.1.0-beta.1`](https://github.com/richardnascimento18/devdock/releases/tag/v1.1.0-beta.1). Build local Linux artifacts and checksums with:

```sh
DEVDOCK_GITHUB_CLIENT_ID=YOUR_PUBLIC_CLIENT_ID make release
./dist/devdock_$(cat VERSION)_linux_amd64 --version
(cd dist && sha256sum --check SHA256SUMS)
```

Builds embed version, full commit SHA, and commit timestamp. No script guesses or increments release versions. Ordinary `go build` reports development metadata.

Publication requires repository variable `DEVDOCK_RELEASES_ENABLED` to be exactly `true`. Unset/`false` disables publication while verification/builds still run, including initial production bootstrap. The official public Client ID remains in `DEVDOCK_GITHUB_CLIENT_ID`.

The permanent branch model is `production` (stable/default/release) and `staging` (integration). Legacy `main` was deleted after the live release rehearsal and history-reachability verification; it is not part of the permanent workflow.

Future work uses a purpose-specific branch → PR into `staging` → verified promotion PR from `staging` into `production` → verified release. Direct development on these permanent branches is prohibited after Pass 1. Read [CONTRIBUTING.md](CONTRIBUTING.md) for checks, exact branch protections, release setup, and hotfix procedure. [The audit](docs/pass-1-audit.md) and [engineering report](docs/pass-1-report.md) describe this pass and its remaining limits.

Licensed under [MIT](LICENSE).

The [Pass 1B live engineering report](docs/pass-1b-report.md) records verified branch protections, hosted CI runs, release artifacts, and remaining limitations.
