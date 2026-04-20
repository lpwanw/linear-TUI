---
name: linear-tui — Go/Bubble Tea TUI for Linear
date: 2026-04-20
owner: Tay
status: in_progress
blockedBy: []
blocks: []
---

# linear-tui — Go/Bubble Tea TUI for Linear

Ground-up rewrite of `linear_tui` (Ruby) in Go on the Charm stack. Same functional spec, same keymap, same pessimistic-mutation + SQLite-cache architecture. Ruby version shipped Phase 1 but hit render flicker and threading limits; Bubble Tea's Elm Architecture eliminates both. Secondary goal: reusable foundation for future TUIs.

**Resolved names:**
- Module path: `github.com/<org>/linear-tui` (org placeholder, set on `go mod init`)
- Binary: `lnr`
- Config dir: `${XDG_CONFIG_HOME}/linear_tui/` (reused, zero-migration from Ruby)
- Data dir: `${XDG_DATA_HOME}/linear_tui/` (same SQLite path — cache schema compatible)
- State dir: `${XDG_STATE_HOME}/linear_tui/`
- Min Go: 1.22

**Resolved open decisions:**
- **GraphQL client:** raw `net/http` + `encoding/json`. 6 queries + 2 mutations total in Phase 1-3. `genqlient` adds build step + generated code churn for low payoff. Revisit if query surface >15.
- **SQLite driver:** `modernc.org/sqlite` (pure Go). CGo-free cross-compile for 4 targets with no toolchain headaches. Benchmark only if cold boot exceeds 500ms budget.
- **Test strategy:** table tests for pure code (Update, motion parser, repos). `httptest.Server` for API client. Live integration gated by `LINEAR_LIVE=1` env; hits personal workspace, read-only queries only. No HTTP-response mocks.

## Phases

| # | Phase | Status | Effort | Goal |
|---|-------|--------|--------|------|
| 01 | Scaffold + Triage/Standup MVP | implemented (pending dogfood) | ~30 hrs | 5-day dogfood: daily standup without Linear web UI |
| 02 | Edit + Capture | todo | ~20 hrs | `c` capture, `o` `$EDITOR`, `i` inline edit, `:team` switch |
| 03 | Flow States | todo | ~20 hrs | Cycles view, `:branch`/`:pr`, comments, background poll |
| 04 | Polish + Publish | todo | ~20 hrs | Themes, `lnr doctor`, GoReleaser, launch posts |

## Dependencies

- **External:** Linear API key (`lin_api_*`), GitHub repo, GoReleaser + Homebrew tap (phase 4)
- **Internal:** Ruby `linear_tui` (reference only — architecture + keymap; no code reuse)
- **Blocking between phases:** Phase 2 requires stable Phase 1 mutation flow. Phase 3 needs `tea.ExecProcess` pattern proven in Phase 2 (`:branch`/`:pr`/`C` all shell out). Phase 4 gated on 3-week real dogfood window.

## Key Decisions (carried from Ruby plan, preserved verbatim)

- **Single workspace.** No runtime switching. Workspace implicit from API key.
- **Pessimistic mutations.** Spinner + await. Optimistic updates deferred indefinitely.
- **Pull-only sync.** Cache is derived state. Mutation = API call → on success, upsert cache row. Never push local state as canonical.
- **No GraphQL subscriptions.** Polling only. Optional 60s background `tea.Tick` in Phase 3.
- **Hand-written GraphQL queries.** No SDK. Constants in `internal/api/queries.go`.
- **Repo pattern for cache.** Plain structs + SQL. No ORM.
- **Pure Update/View.** No mutation inside Bubble Tea `Update`. All async = `tea.Cmd` returning `tea.Msg`. No goroutines outside commands.

## Research

- [Linear GraphQL API](../../../linear_tui/plans/260420-linr-ruby-tui-linear/research/researcher-01-linear-graphql-api.md) — reused verbatim (language-agnostic)
- [Markdown rendering](../../../linear_tui/plans/260420-linr-ruby-tui-linear/research/researcher-04-markdown-rendering.md) — Ruby used `tty-markdown`; Go uses `glamour` (same philosophy, Charm's own renderer)
- [Naming + prior art](../../../linear_tui/plans/260420-linr-ruby-tui-linear/research/researcher-03-naming-and-prior-art.md) — binary name `lnr` cleared
- Ruby TUI stack research does NOT apply (Go uses Bubble Tea, not tty-*)

## Resolved Questions

1. ✅ Module path placeholder `github.com/<org>/linear-tui`; confirm org before first push
2. ✅ Cache DB path shared with Ruby install: `${XDG_DATA_HOME}/linear_tui/cache.db`. Schema compatible (idempotent migrations re-run at startup). Dogfooders can drop in Go binary without losing cache.
3. ✅ `state.json` schema extended but backward-compatible: Ruby's `{view, cursor_index, selected_team_id}` intact; Go additions prefixed `go_` if any.
4. ✅ Cross-compile targets (phase 4): darwin-arm64, darwin-amd64, linux-amd64, linux-arm64. No Windows until user demand.

## Success Gates

- **Phase 1 DoD:** 5 consecutive weekday standups completed without opening Linear web. Cold boot <500ms (cache hit). Keystroke → paint <16ms. `go test ./...` green.
- **Phase 2 DoD:** `c` creates issue, `o` edits description via `$EDITOR`, both flows work on iTerm2 + Alacritty + tmux. No terminal state corruption after `tea.ExecProcess`.
- **Phase 3 DoD:** Can start a PR workflow end-to-end from `lnr` (pick issue → `:branch` → code → `:pr`). Background poll does not cause visible repaints.
- **Phase 4 DoD:** `brew install <tap>/lnr` works on 2 machines. Release announcement live on ≥1 platform. README has asciinema cast.

## Non-Goals (explicit)

- No OAuth (API key only in all 4 phases)
- No multi-workspace
- No offline mutation queue
- No custom field support
- No write access to comments in Phase 1-2
- No Bubble Tea framework extraction — stays in `internal/` until a 2nd TUI exists
