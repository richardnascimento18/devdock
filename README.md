<div align="center">

```
                                    ██████╗ ███████╗██╗   ██╗██████╗  ██████╗  ██████╗██╗  ██╗
                                    ██╔══██╗██╔════╝██║   ██║██╔══██╗██╔═══██╗██╔════╝██║ ██╔╝
                                    ██║  ██║█████╗  ██║   ██║██║  ██║██║   ██║██║     █████╔╝ 
                                    ██║  ██║██╔══╝  ╚██╗ ██╔╝██║  ██║██║   ██║██║     ██╔═██╗ 
                                    ██████╔╝███████╗ ╚████╔╝ ██████╔╝╚██████╔╝╚██████╗██║  ██╗
                                    ╚═════╝ ╚══════╝  ╚═══╝  ╚═════╝  ╚═════╝  ╚═════╝╚═╝  ╚═╝
```

*your terminal workspace manager*

[![Go](https://img.shields.io/badge/Go-1.21+-8B5CF6?style=flat-square&logo=go&logoColor=white)](https://golang.org)
[![tmux](https://img.shields.io/badge/tmux-required-A78BFA?style=flat-square&logo=tmux&logoColor=white)](https://github.com/tmux/tmux)
[![License](https://img.shields.io/badge/license-MIT-C4B5FD?style=flat-square)](LICENSE)

</div>

<br>

DevDock is a keyboard-driven TUI that allows you to easily start working on your projects by automating the creation of TMUX Sessions with your desired layout, the creation of new projects and your project's scaffolding tool command all in one go. It works out of the box, no manual configuration needed!

<br>

---

## Building from source

> DevDock's GitHub integration requires a **GitHub OAuth App Client ID** injected at build time. Although I personally recommend installing one of the releases to use DevDock's official OAuth App, you can use your own OAuth App at build time, by using the exact command below:

```sh
go build \
  -ldflags "-X 'github.com/richardnascimento18/devdock/internal/github.ClientID=YOUR_CLIENT_ID'" \
  -o devdock .
```

Building without it will produce a binary where all GitHub features, such as connecting an account, creating repos, cloning, will silently fail.

Alternatively, skip the build flag and export the variable instead:

```sh
export DEVDOCK_GITHUB_CLIENT_ID=your_client_id
```

---

<br>

## Features in DevDock

<br>

### 🔍 &nbsp;Project discovery

By following a folder structure (root > category/domain > project), DevDock scans your roots and classifies directories automatically. It also supports grouping and subgrouping, to allow for projects that relate to each other to live together, without being scattered in domains (root > domain > group > project[s]). Override automatic detection anytime with `.devdock` or `.ddgroup` marker files.

<br>

### 🖥️ &nbsp;tmux workspace presets

When you select a project from the list, DevDock creates (or re-attaches to) a tmux session named after it. The layout is driven by a **preset** (a named configuration that defines windows, pane splits, and the command to run in each).

Built-in presets:

| Preset | Description |
|---|---|
| `walker` | `nvim` · `terminal` · `ai-chat` (opencode) |
| `nvim` | `nvim` · `terminal` |
| `dev-split` | Editor + two side terminals in a single window |

Customise presets freely in `~/.config/devdock/presets.json`. They support any number of windows, horizontal/vertical splits, sized panes, and per-pane commands.

<br>

### 🧱 &nbsp;Project templates

Scaffold new projects from **templates**. These are sequences of commands and/or filesystem operations (`touch`, `mkdir`) that run in a PTY, so interactive CLIs work correctly out of the box.

Built-in templates:

| Template | Stack |
|---|---|
| `Next.js` | `create-next-app` (interactive) |
| `React + Vite` | Vite · TypeScript |
| `Go API` | Go modules · Gin |
| `Go CLI` | Go modules · Cobra |
| `Python FastAPI` | venv · FastAPI · uvicorn |
| `Python Script` | venv · bare Python |
| `Rust` | `cargo new` binary crate |
| `Rust Library` | `cargo new --lib` |
| `Node.js` | `npm init` |
| `Static HTML` | HTML · CSS · JS skeleton |

Define your own templates in `~/.config/devdock/templates.json` using the same step-based format.

<br>

### 🐙 &nbsp;GitHub integration

Connect a GitHub account once with `g` via the **device OAuth flow**. After authenticating, DevDock:

- Loads all your repositories and shows uncloned ones inline in the project list
- Matches local projects to their remote repos and badges them in the UI
- Creates a **new GitHub repo** (public or private) during project creation and pushes an initial commit automatically
- **Clones any of your repos** directly into a chosen domain from the TUI

<br>

### ⭐ &nbsp;Favorites & recents

Star any project with `f` (up to 5 favorites). Recent projects are tracked automatically on every open. Both are accessible via the tab bar (`[` / `]`) so your most-used projects are always one keypress away.

<br>

### 🗂️ &nbsp;Full workspace management

Everything you need, without leaving the TUI:

- `n` — create a new project (optionally with a template and a GitHub repo)
- `x` / `X` — delete a project or an entire domain (with confirmation)
- `m` — move a project to a different root, domain, or group
- `N` — create a new domain in any root
- `G` / `ctrl+g` — create or delete groups
- `a` / `A` — add or remove root directories
- `.ddignore` — ignore specific domains per root using glob patterns

<br>


## Installation

**Requirements:** Go 1.21+, tmux

```sh
git clone https://github.com/richardnascimento18/devdock
cd devdock

go build \
  -ldflags "-X 'github.com/richardnascimento18/devdock/internal/github.ClientID=YOUR_CLIENT_ID'" \
  -o devdock .

mv devdock /usr/local/bin/
```

On first run, DevDock will ask for a root directory and write its config to `~/.config/devdock/`. Default preset and template files are generated there automatically.

---

<br>

## Configuration

All configuration lives in `~/.config/devdock/`:

| File | Purpose |
|---|---|
| `config.toml` | Root directories, default preset, GitHub credentials |
| `presets.json` | tmux workspace presets |
| `templates.json` | Project scaffolding templates |

The config directory is created automatically on first run. You should never need to edit `config.toml` by hand — everything is manageable from within the TUI.

---

<br>

## Tech stack

| | |
|---|---|
| [Bubble Tea](https://github.com/charmbracelet/bubbletea) | TUI framework |
| [Bubbles](https://github.com/charmbracelet/bubbles) | TUI components |
| [Lip Gloss](https://github.com/charmbracelet/lipgloss) | Terminal styling |
| [go-toml](https://github.com/pelletier/go-toml) | TOML config parsing |
| tmux | Workspace management |

---

<br>

<div align="center">

*Built for developers who live in the terminal.*

</div>
