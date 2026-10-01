# DevDock

DevDock is a keyboard-driven terminal workspace manager for Linux. It discovers projects, launches tmux workspaces with saved layouts, scaffolds projects from templates, and optionally connects to GitHub through its device authorization flow.

The current hierarchy is `root / domain / project`, with groups and subgroups inside domains. This stabilization pass preserves that model and the existing interface.

## Requirements and installation

- Linux amd64 or arm64. Linux-specific filesystem and PTY handling is intentional.
- Go **1.26 or newer** for source builds; CI tests the latest patches of Go 1.26 and 1.27. The previous Go 1.21 claim was incorrect.
- tmux for opening workspaces; Git for GitHub cloning/linking. Template commands and preset commands require their own tools, such as Node.js, Python, Cargo, nvim, or opencode.
- Python 3 and Bash for development validation and release scripts.

Release infrastructure is prepared, but this pass does **not** publish a release. When published, official binaries will be available on the [GitHub releases page](https://github.com/richardnascimento18/devdock/releases). A downloaded binary needs no Go installation. Download the matching `devdock_<version>_linux_amd64` or `devdock_<version>_linux_arm64` and `SHA256SUMS`, verify the matching checksum, then install it:

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
| `f` | Toggle a favorite (maximum five) |
| `n` / `N` | Create a project/domain |
| `m` | Move a project between current roots/domains/groups |
| `x` / `X` | Delete a project/domain |
| `G` / Ctrl+G | Create/delete a group |
| `a` / `A` | Add/remove a root |
| `e` | Open the preset/template editor |
| `g` | Connect GitHub, or refresh connected repositories |
| Esc | Cancel the current flow |
| `q` | Quit from the main screen |

Recursive project, domain, and group deletion requires typing the full target path. Project creation and moves reject existing destinations; moves never merge directories. A cross-filesystem move preserves symlinks and reports incomplete source cleanup as a partial failure. Favorites/recents follow successful moves and are pruned after successful deletions; temporarily unavailable roots retain their state.

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

A project `.devdock` TOML file can override detection with `type` (`project`, `group`, or `subgroup`) and optional `name` fields. A `.ddgroup` marker identifies a group/subgroup directory. Root `.ddignore` files contain domain-name glob patterns, one per line; blank lines and comments beginning with `#` are ignored. Invalid patterns are reported.

Template step commands support quoted arguments such as `command --title "Hello world"`. They are parsed into arguments and executed directly, without shell expansion. Variables are `{{project_name}}`, `{{project_path}}`, `{{domain}}`, and `{{root}}`. Shell operators, `$()`, environment expansion, and glob expansion are not implicitly evaluated. Use the step's `output` field to capture stdout into a project-relative file instead of `>` redirection. Builtin `touch`, `mkdir`, and `rm`, and output paths, must remain strict descendants of the project and cannot traverse symlinks or `..`; `rm .` is rejected. Explicit shell-mode steps are unsupported. Custom commands remain trusted executable code; this confinement is for builtin filesystem actions, not an OS sandbox for arbitrary commands.

## GitHub authorization and custom builds

DevDock uses a GitHub **OAuth App** with **device flow enabled**. The public `client_id` identifies the application; this flow does not require a client secret. DevDock accepts no OAuth client-secret setting and does not compile one into its binary. See [GitHub's device-flow documentation](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps).

Official release builds obtain the public Client ID from the repository variable `DEVDOCK_GITHUB_CLIENT_ID` and embed it with a linker flag. The owner must register/configure the official app and set that variable before publication. No official ID is fabricated or committed in this pass.

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
make tools       # install pinned staticcheck, govulncheck, actionlint; add GOPATH/bin to PATH
make check       # formatting, module consistency, version/workflow validation, tests,
                 # race tests, vet, staticcheck, govulncheck, build
make test
make race
```

All tests use temporary workspaces and fakes for GitHub/tmux/editor boundaries. PTY tests run small local child processes. Race tests require Linux, CGO, and a C compiler; release binaries are built with CGO disabled. `govulncheck` needs network access to the advisory database.

`VERSION` is an explicitly reviewed Semantic Version, currently a pre-1.0 development prerelease. Build local Linux artifacts and checksums with:

```sh
DEVDOCK_GITHUB_CLIENT_ID=YOUR_PUBLIC_CLIENT_ID make release
./dist/devdock_$(cat VERSION)_linux_amd64 --version
(cd dist && sha256sum --check SHA256SUMS)
```

Builds embed version, full commit SHA, and commit timestamp. No script guesses or increments release versions. Ordinary `go build` reports development metadata.

Future work uses a purpose-specific branch → PR into `staging` → verified promotion PR from `staging` into `production` → verified release. Direct development on these permanent branches is prohibited after Pass 1. Read [CONTRIBUTING.md](CONTRIBUTING.md) for checks, exact branch protections, release setup, and hotfix procedure. [The audit](docs/pass-1-audit.md) and [engineering report](docs/pass-1-report.md) describe this pass and its remaining limits.

Licensed under [MIT](LICENSE).
