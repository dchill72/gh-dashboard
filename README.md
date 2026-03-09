# gh-dashboard

A terminal UI for browsing GitHub (or GitHub Enterprise) pull requests where you are a requested reviewer.

Built with [bubbletea](https://github.com/charmbracelet/bubbletea), [bubbles](https://github.com/charmbracelet/bubbles), [lipgloss](https://github.com/charmbracelet/lipgloss), and [glamour](https://github.com/charmbracelet/glamour).

## Features

- Split-pane layout: PR list on the left, rendered PR description on the right
- Live filter by title or repo (`/`)
- Cycle review state filter: All → Pending → Approved → Changes Requested (`r`)
- Sort by date ascending/descending (`s`)
- Read/unread tracking — unread PRs are blue, read are green; state is persisted to disk
- Markdown rendering of PR descriptions via glamour
- Open PR in your default browser (`o`)
- Supports GitHub.com and GitHub Enterprise (configured per install)

## Install

### Prebuilt binary (recommended)

```sh
curl -fsSL https://raw.githubusercontent.com/dchill72/gh-dashboard/main/install.sh | sh
```

Or review first, then run locally:

```sh
curl -fsSLO https://raw.githubusercontent.com/dchill72/gh-dashboard/main/install.sh
sh install.sh
```

Optional install variables:

```sh
# install a specific version (accepts v-prefixed tags or plain versions)
GH_DASHBOARD_VERSION=v0.1.0 sh install.sh
GH_DASHBOARD_VERSION=0.1.0 sh install.sh

# install from another repo (forks/private mirrors)
GH_DASHBOARD_REPO=owner/repo sh install.sh

# install to a custom path
INSTALL_DIR="$HOME/.local/bin" sh install.sh
```

### Build from source

```sh
go install .
```

This places `gh-dashboard` in your `$GOPATH/bin` (ensure it is in `$PATH`).

## Releases

Tagged pushes like `v0.1.0` trigger the release workflow in `.github/workflows/release.yml`, which uses GoReleaser to publish checksummed tarballs for:

- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`

Release tags are `vX.Y.Z`, while archive names use `X.Y.Z` (without the `v`). The installer handles this automatically.

## Setup

### 1. Config file

Copy the example config and edit it:

```sh
mkdir -p ~/.config/gh-dashboard
cp config.example.toml ~/.config/gh-dashboard/config.toml
```

`~/.config/gh-dashboard/config.toml`:

```toml
[github]
# "github.com" or your GHE hostname e.g. "github.example.com"
host = "github.com"

# All repos in an org (omit repos to include all)
[[orgs]]
name = "my-org"

# Only specific repos in an org
[[orgs]]
name = "another-org"
repos = ["my-repo", "other-repo"]
```

### 2. Authentication

Export a GitHub personal access token with `repo` and `read:org` scopes:

```sh
export GITHUB_TOKEN=ghp_...
```

The same token is used for all configured orgs/hosts.

### 3. Build and run

```sh
go build -o gh-dashboard .
GITHUB_TOKEN=ghp_... ./gh-dashboard
```

## Keybindings

| Key | Action |
| --- | --- |
| `↑` / `↓` / `j` / `k` | Navigate the PR list |
| `/` | Enter filter mode — live filters by title or repo |
| `Esc` | Exit filter mode and clear filter text |
| `s` | Toggle sort: newest first / oldest first |
| `r` | Cycle review filter: All → Pending → Approved → Changes Requested |
| `o` | Open the selected PR in your default browser |
| `m` | Toggle read/unread for the selected PR |
| `F5` | Refresh PR list from GitHub |
| `PgUp` / `PgDn` | Scroll the detail pane |
| `q` / `Ctrl+C` | Quit |

## State

Read/unread status is saved to `~/.config/gh-dashboard/state.json`. PRs are automatically marked as read when you navigate to them. Press `m` to toggle.

## Project layout

```text
main.go
config.example.toml
Makefile
internal/
  config/
    config.go          — TOML config loading
    config_test.go     — unit tests
  github/
    types.go           — PR and OrgQuery types
    client.go          — GitHub GraphQL API client (viewer login, paginated search)
  logger/
    logger.go          — file-based logger, enabled via LOGGING=1
  state/
    state.go           — read/unread persistence (~/.config/gh-dashboard/state.json)
  ui/
    model.go           — Model struct, layout helpers, filter/sort, render cache
    update.go          — key handling, tea.Cmd factories
    view.go            — split-pane rendering (list, detail, header, footer)
    styles.go          — lipgloss colour palette and styles
    browser.go         — cross-platform browser opener (xdg-open / open / start)
```

## Development

```sh
make build       # compile to ./gh-dashboard
make run         # build and run (requires GITHUB_TOKEN in env)
make test        # run all tests with verbose output
make vet         # run go vet
make lint        # run golangci-lint (downloaded on first use via go run)
make tidy        # run go mod tidy
make clean       # remove the compiled binary
```

Enable debug logging to `./logs/<timestamp>.log`:

```sh
LOGGING=1 GITHUB_TOKEN=ghp_... ./gh-dashboard
```
