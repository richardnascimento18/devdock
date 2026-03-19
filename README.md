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

DevDock scans your project roots, lets you navigate your entire workspace in a keyboard-driven TUI, and drops you straight into a fully configured **tmux session** — windows, panes, and commands ready to go. No config files to edit, no scripts to maintain.

<br>

---

## ⚠️ Building from source

> DevDock's GitHub integration requires a **GitHub OAuth App Client ID** injected at build time. Building without it will produce a binary where all GitHub features — connecting an account, creating repos, cloning — silently fail.

```sh
go build \
  -ldflags "-X 'github.com/richardnascimento18/devdock/internal/github.ClientID=YOUR_CLIENT_ID'" \
  -o devdock .
```

**To get a Client ID:**

1. Go to **GitHub → Settings → Developer settings → OAuth Apps → New OAuth App**
2. Set the callback URL to anything (e.g. `http://localhost`) — DevDock uses the device flow, so it is never called
3. Enable **Device Flow** on the app settings page
4. Copy the **Client ID** and pass it to the build command above

Alternatively, skip the build flag and export the variable instead:

```sh
export DEVDOCK_GITHUB_CLIENT_ID=your_client_id
```

---

<br>

## ✦ Features

<br>

### 🔍 &nbsp;Smart project discovery

DevDock scans your roots and classifies directories automatically — no manual registration. A directory becomes a **project** when it contains known markers (`go.mod`, `package.json`, `Cargo.toml`, `pyproject.toml`, `requirements.txt`, and many more). **Groups** and **subgroups** give you nested organisation inside a domain without any extra config. Override automatic detection anytime with `.devdock` or `.ddgroup` marker files.

Detection results are **cached in memory** after the first scan — navigating between roots and toggling groups is instant. Press `r` to force a fresh re-scan from disk.

<br>

### 🖥️ &nbsp;tmux workspace presets

When you open a project, DevDock creates (or re-attaches to) a tmux session named after it. The layout is driven by a **preset** — a named configuration that defines windows, pane splits, and the command to run in each.

Built-in presets:

| Preset | Description |
|---|---|
| `walker` | `nvim` · `terminal` · `ai-chat` (opencode) |
| `nvim` | `nvim` · `terminal` |
| `dev-split` | Editor + two side terminals in a single window |

Customise presets freely in `~/.config/devdock/presets.json`. They support any number of windows, horizontal/vertical splits, sized panes, and per-pane commands. Cycle through presets from the launcher itself with `p` / `P` — no need to leave.

<br>

### 🧱 &nbsp;Project templates

Scaffold new projects from **templates** — sequences of commands and filesystem operations (`touch`, `mkdir`) that run in a PTY, so interactive CLIs work correctly out of the box.

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

Connect a GitHub account once with `g` via the **device OAuth flow** — just open a URL and enter a code, no token copy-pasting. After authenticating, DevDock:

- Loads all your repositories and shows uncloned ones inline in the project list
- Matches local projects to their remote repos and badges them in the UI
- Creates a **new GitHub repo** (public or private) during project creation and pushes an initial commit automatically
- **Clones any of your repos** directly into a chosen domain from the TUI

<br>

### 🌲 &nbsp;Tree & flat views

Press `v` to toggle between **tree view** — projects nested under collapsible group headers — and **flat view** — every project on a single line with a breadcrumb prefix. Collapse and expand groups with `space` or `enter`. Your preferred view is remembered across sessions.

<br>

### ⭐ &nbsp;Favorites & recents

Star any project with `f` (up to 5 favorites). Recent projects are tracked automatically on every open. Both are accessible via the tab bar (`[` / `]`) so your most-used projects are always one keypress away.

<br>

### 🗂️ &nbsp;Full workspace management

Everything you need, without leaving the TUI:

- `n` — create a new project (optionally with a template and a GitHub repo)
- `x` / `X` — delete a project or an entire domain (with confirmation)
- `m` — move a project to a different root, domain, or group — across roots if needed
- `N` — create a new domain in any root
- `G` / `ctrl+g` — create or delete groups
- `a` / `A` — add or remove root directories
- `.ddignore` — ignore specific domains per root using glob patterns

<br>

### 🔎 &nbsp;Search & filter

Press `/` in the Search tab to fuzzy-filter the project list. Confirm with `enter` to lock the filter; clear it with `c` or `esc`. If you search for a name with no match, a **create project** shortcut appears inline — go from idea to scaffold in seconds.

---

<br>

## 🚀 &nbsp;Installation

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

## ⚙️ &nbsp;Configuration

All configuration lives in `~/.config/devdock/`:

| File | Purpose |
|---|---|
| `config.toml` | Root directories, default preset, GitHub credentials |
| `presets.json` | tmux workspace presets |
| `templates.json` | Project scaffolding templates |

The config directory is created automatically on first run. You should never need to edit `config.toml` by hand — everything is manageable from within the TUI.

---

<br>

## ⌨️ &nbsp;Keybindings

| Key | Action |
|---|---|
| `enter` | Open project · toggle group |
| `tab` / `shift+tab` | Cycle root |
| `[` / `]` | Cycle tab (Search · Recents · Favorites) |
| `space` | Collapse / expand group |
| `v` | Toggle tree / flat view |
| `n` | New project |
| `N` | New domain |
| `x` | Delete project |
| `X` | Delete domain |
| `m` | Move project |
| `f` | Toggle favorite |
| `p` / `P` | Cycle preset forward / back |
| `g` | Connect GitHub · refresh repos |
| `G` | Create group |
| `ctrl+g` | Delete group |
| `r` | Refresh (re-scan from disk) |
| `a` / `A` | Add / remove root |
| `/` | Filter (Search tab) |
| `c` / `esc` | Clear filter |
| `?` | Help |
| `ctrl+c` | Quit |

---

<br>

## 🛠️ &nbsp;Tech stack

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
