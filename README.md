# linear-tui (lnr)

Vim-style terminal UI for [Linear](https://linear.app). Single workspace,
pessimistic mutations, local SQLite cache for instant startup.

> Go + Bubble Tea rewrite of the Ruby proof-of-concept. Phase 1 of 4.

## Install

Requires Go 1.22+.

```bash
go install github.com/taynguyen/linear-tui/cmd/lnr@latest
```

Or from source:

```bash
git clone https://github.com/taynguyen/linear-tui
cd linear-tui
make build   # produces ./lnr
```

## Setup

Generate a personal API key at Linear → Settings → Account → Security & Access.
The key starts with `lin_api_`.

```bash
export LINEAR_API_KEY=lin_api_xxxxxxxxxxxxxxxxxx
lnr
```

## Keybindings

| Key | Action |
|---|---|
| `1` / `2` | Switch view: My Issues / Triage |
| `j` `k` | Cursor down / up |
| `gg` `G` | Jump top / bottom |
| `Ctrl-d` `Ctrl-u` | Half-page down / up |
| `Ctrl-f` `Ctrl-b` | Page down / up |
| `{n}j` / `{n}k` | Numeric prefix (e.g. `5j`) |
| `s` / `a` / `p` | State / assignee / priority picker |
| `/` | Live filter |
| `r` | Refresh current view |
| `?` | Toggle help overlay |
| `:` | Command mode |
| `esc` | Close modal / cancel search / dismiss error |
| `q` | Quit |

## Commands

| Command | Action |
|---|---|
| `:q` / `:qa` / `:quit` | Quit |
| `:sync` | Full resync |
| `:refresh` | Refresh current view |
| `:view my_issues\|triage` | Switch view |
| `:state <name>` | Change state of selected issue |
| `:assign <name\|me\|none>` | Change assignee |
| `:priority <n\|name>` | Change priority (0–4 or urgent/high/normal/low/none) |
| `:open` | Open issue URL in default browser |

## Paths (XDG)

- Cache DB: `${XDG_DATA_HOME:-~/.local/share}/linear_tui/cache.db`
- State:    `${XDG_STATE_HOME:-~/.local/state}/linear_tui/state.json`
- Config:   `${XDG_CONFIG_HOME:-~/.config}/linear_tui/` (Phase 4)

## Architecture

```
cmd/lnr/main.go
  ↓
config.Load() → cache.Open() → api.NewClient() → sync.Service
  ↓
tea.NewProgram(app.Model, tea.WithAltScreen()).Run()

app.Model (Elm Architecture)
  ├── Init()   → sync.Bootstrap(ctx) as tea.Cmd
  ├── Update() → pure; returns (Model, tea.Cmd)
  └── View()   → pure; returns rendered string
```

All async = `tea.Cmd`. No goroutines in Model. All state changes = pure Update.

## Status

Phase 1 MVP (Triage + Standup). Phases 2–4 planned; see `plans/`.

## License

MIT
