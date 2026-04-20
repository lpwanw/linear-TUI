# Phase 4 — Polish + Publish

**Priority:** P2
**Effort:** ~20 hrs over 2 weeks
**Status:** blocked (on Phase 3 DoD + 3-week dogfood)

## Context Links

- [Plan overview](./plan.md)
- [Phase 3](./phase-03-flow-states.md)
- [Naming + prior art research](../../../linear_tui/plans/260420-linr-ruby-tui-linear/research/researcher-03-naming-and-prior-art.md)

## Overview

Make it shippable. Theme system via Lipgloss, config file with keybinding overrides, `lnr doctor` diagnostics, README with asciinema + GIFs, cross-compile via GoReleaser, Homebrew tap, launch posts.

## Key Insights

- Lipgloss style sets can be swapped at runtime by passing a `Theme` struct down through `View`. No global mutable state.
- `goreleaser` handles multi-arch Go builds + checksums + GitHub release + Homebrew tap formula generation in one config. Zero CI custom scripting.
- Homebrew tap = separate repo `homebrew-<name>`; GoReleaser writes the formula file on release.
- `lnr doctor` — check: `LINEAR_API_KEY` set + valid, SQLite path writable, `$EDITOR` resolves, `gh` in PATH (optional), `git` in PATH (optional), terminal 256-color capability.
- Config file: TOML via `github.com/BurntSushi/toml`. Path: `${XDG_CONFIG_HOME}/linear_tui/config.toml`. Optional file; all fields have defaults. Keybinding overrides map `action name → key`.
- Asciinema cast + `agg` tool to convert to GIF for README.

## Requirements

### Functional

- 3 built-in themes: `gruvbox`, `catppuccin`, `tokyo-night`. Default = `tokyo-night`.
- Config file: `~/.config/linear_tui/config.toml` (XDG). Fields: `theme`, `keybindings`, `default_team` (key).
- `lnr doctor` subcommand — prints diagnostic checklist.
- `lnr --help` / `lnr --version`.
- `lnr sync` / `lnr refresh` subcommands (non-TUI) for scripts/cron.
- Homebrew install: `brew install <org>/lnr/lnr` (single tap, single formula).

### Non-functional

- Cross-compile clean for darwin-arm64, darwin-amd64, linux-amd64, linux-arm64 (no CGo thanks to `modernc.org/sqlite`).
- Binary size target: < 15MB stripped.
- Release process: push tag → CI runs `goreleaser release` → artifacts on GitHub + Homebrew tap updated. Zero manual steps.

## Architecture

New packages:
- `internal/theme/` — `Theme` struct with lipgloss styles for every semantic UI element. `Registry` = `map[string]Theme`. Load via name string from config.
- `internal/doctor/` — diagnostic checks.

Config flow:
```
main.go
  → config.Load() reads env + XDG paths
  → config.LoadFile(path) reads TOML if exists, else defaults
  → merge: env > file > defaults
  → theme.Registry[cfg.Theme] → passed into app.New(deps)
```

Config file schema:
```toml
theme = "tokyo-night"              # or "gruvbox" | "catppuccin"
default_team = "BUT"               # team key; used for Triage view on startup

[keybindings]
quit = "q"
my_issues = "1"
triage = "2"
cycle = "3"
state_picker = "s"
# … etc. every mutable binding
```

## Related Code Files

**Create:**
- `internal/theme/theme.go` + `gruvbox.go` + `catppuccin.go` + `tokyo_night.go`
- `internal/doctor/doctor.go`
- `cmd/lnr/doctor.go` — subcommand routing
- `.goreleaser.yaml`
- `docs/` — `keybindings.md`, `theming.md`
- `README.md` — full rewrite w/ asciinema + GIFs
- GitHub Actions release workflow: `.github/workflows/release.yml`

**Modify:**
- `internal/app/view.go` + all `views/*` — accept `theme.Theme` param
- `internal/config/config.go` — TOML file load, merge with env
- `cmd/lnr/main.go` — subcommand dispatch (`doctor`, `sync`, `refresh`, default = TUI)

## Implementation Steps

### 1. Theme system (hrs 0-5)
1. Extract all hardcoded lipgloss styles from `views/` and `modals/` into `internal/theme/Theme` struct fields (`ListCursor`, `StateBadge`, `PriorityUrgent`, `StatusLine`, `ErrorBanner`, etc.).
2. Thread `theme.Theme` through views via the root Model (copy on config load; no mutation).
3. Implement `gruvbox`, `catppuccin`, `tokyo-night` color palettes.
4. Snapshot tests: render same Model with each theme, assert different ANSI output.
5. `NO_COLOR` still forces monochrome regardless of theme.

### 2. Config file (hrs 5-8)
1. Add `github.com/BurntSushi/toml` dep.
2. `config.LoadFile(path)` — returns struct with optional fields. Missing file → zero struct.
3. Keybinding override map: `map[string]string` where key = action name (enumerated constant), value = key chord. Validate at load; unknown action → warn, continue.
4. Merge order: flag > env > file > defaults. Record sources for `doctor`.
5. Doc page `docs/theming.md` + `docs/keybindings.md` listing all action names.

### 3. `lnr doctor` (hrs 8-11)
1. `internal/doctor/doctor.go` — exposes `Run(ctx) []Check`. Each `Check = {Name, OK, Message, Hint}`.
2. Checks:
   - `LINEAR_API_KEY` set + length check (`lin_api_` prefix)
   - Live API `viewer` query succeeds (reuses api client)
   - SQLite path writable
   - Config file syntax valid if present
   - `$EDITOR` resolves to an executable in PATH
   - `gh` in PATH (warning, not failure)
   - `git` in PATH (warning)
   - Terminal `$TERM` in a known-good list (warning otherwise)
3. `cmd/lnr/doctor.go` — formats output as readable checklist with pass/fail/warn. Exit 0 if no failures, 1 if any.

### 4. Subcommand dispatch (hrs 11-12)
1. `cmd/lnr/main.go` parses `os.Args[1]`: `doctor`, `sync`, `refresh`, `--help`, `--version`, default → TUI.
2. `sync` / `refresh` run the sync service without bubble tea, print progress to stderr, exit.
3. Use `flag` stdlib; skip cobra (YAGNI — 4 subcommands).

### 5. GoReleaser (hrs 12-15)
1. `.goreleaser.yaml`:
   - `builds` — 4 targets: darwin/arm64, darwin/amd64, linux/amd64, linux/arm64. CGO_ENABLED=0.
   - `archives` — tar.gz per target + source.tar.gz
   - `checksum` — sha256
   - `release` — GitHub release from current tag
   - `brews` — generate formula in `homebrew-lnr` tap repo
2. GitHub Actions `release.yml` — triggers on `v*` tag, runs `goreleaser release --clean`.
3. Create `<org>/homebrew-lnr` repo. Add `HOMEBREW_TAP_TOKEN` secret (classic PAT with `repo` scope).
4. Dry-run: `goreleaser release --snapshot --clean` locally — verify 4 binaries + formula.

### 6. README + demos (hrs 15-18)
1. Record asciinema cast: startup → my_issues → triage → mutation → capture → PR.
2. Convert to GIF via `agg` (limit ~1MB).
3. README sections: Features, Install (Homebrew + `go install` + binary download), Quickstart (`export LINEAR_API_KEY=...; lnr`), Keybindings table, Config, Theming, `lnr doctor`, Contributing, License.
4. Link to asciinema.org cast + embedded GIF.
5. Screenshot set (3 themes).

### 7. Launch (hrs 18-20)
1. r/golang post — architecture writeup angle (Bubble Tea case study).
2. r/commandline post — feature showcase.
3. HN Show post — daily-driver framing.
4. dev.to — cross-post of r/golang article.
5. Tweet thread w/ GIF + link.
6. Respond to feedback for 1 week; bug-fix pass via patch release.

## Todo List

- [ ] Extract hardcoded styles into `theme.Theme`
- [ ] Implement 3 themes + snapshot tests
- [ ] Config file loader (TOML) + keybinding override validation
- [ ] `docs/keybindings.md` + `docs/theming.md`
- [ ] `lnr doctor` subcommand + all checks
- [ ] `lnr sync` / `lnr refresh` non-TUI subcommands
- [ ] `.goreleaser.yaml` + release workflow
- [ ] Homebrew tap repo bootstrap
- [ ] Dry-run release locally
- [ ] Asciinema cast + GIF
- [ ] README full rewrite
- [ ] Launch posts drafted + posted
- [ ] 1-week post-launch bug triage

## Success Criteria

- `brew install <org>/lnr/lnr` on a fresh Mac boots the TUI in <3min end-to-end
- 3 themes swappable via config; `NO_COLOR=1` still works
- `lnr doctor` detects all documented failure modes (tested by deliberately breaking each: no key, bad key, no SQLite dir, no `$EDITOR`)
- GoReleaser produces signed checksums; all 4 binaries < 15MB
- ≥1 launch post gets >10 upvotes; README gets >10 stars in first week (soft targets)
- No critical bug in first 2 weeks post-launch

## Risk Assessment

| Risk | Mitigation |
|---|---|
| Homebrew tap permissions / PAT expiry | Use fine-grained PAT w/ single-repo scope; document renewal in repo readme |
| GoReleaser config drift across versions | Pin GoReleaser version in release workflow |
| Theme snapshot tests brittle vs OS terminal differences | Test via `lipgloss.SetColorProfile(termenv.TrueColor)` for determinism |
| Launch-day volume overwhelms single maintainer | Set issue templates; auto-label with actions-bot; schedule 2 quiet weeks post-launch |
| Asciinema cast size blows past README budget | Trim cast to 45s; use `agg --fps 15` for smaller GIF |

## Security Considerations

- Release binaries checksummed; SHA256 published alongside.
- `lnr doctor` never echoes the API key — only `set`/`missing`/`invalid`.
- Homebrew formula does not embed secrets.
- No telemetry, no update checks — user controls upgrade cadence.

## Next Steps

- Post-release: Phase 5 (speculative) — optimistic mutations, multi-workspace, or framework extraction after a 2nd TUI exists
- Maintenance: weekly API schema smoke test (live integration); quarterly dependency bump
