# Phase 1 — Scaffold + Triage/Standup MVP

**Priority:** P0 (blocks everything)
**Effort:** ~30 hrs over 2 weeks
**Status:** todo

## Context Links

- [Plan overview](./plan.md)
- [Linear GraphQL research](../../../linear_tui/plans/260420-linr-ruby-tui-linear/research/researcher-01-linear-graphql-api.md)
- [Ruby Phase 1 plan (architectural reference)](../../../linear_tui/plans/260420-linr-ruby-tui-linear/phase-01-scaffold-triage-mvp.md)

## Overview

Scaffold the Go module + Bubble Tea `Program`. Build minimum path: `lnr` → splash → first-run bootstrap sync → My Issues / Triage / Detail split-pane view → vim-nav → modal pickers (`s`/`a`/`p`) → pessimistic `issueUpdate` → refresh → quit. Dogfood 5 consecutive standup days.

Everything under "Functional Requirements" in the bootstrap spec lives in this phase.

## Key Insights

- Bubble Tea = strict Elm Architecture: `Model`/`Update(msg) (Model, tea.Cmd)`/`View(Model) string`. Every async op (HTTP, DB, IO) must be returned as a `tea.Cmd` — running a goroutine inline breaks the mental model and the tests.
- `tea.Cmd` = `func() tea.Msg`. Wrap API calls so they emit a typed result message; `Update` handles the message and produces the next model.
- `bubbles/list` is tempting but opinionated — we want a custom renderer for the table-row layout and split pane, so use `bubbles/viewport` for the detail pane only and hand-render the list.
- `lipgloss.JoinHorizontal` handles split-pane composition cleanly; no manual column math.
- Linear auth header: `Authorization: <key>` (no `Bearer`). Keys prefix `lin_api_`.
- Linear errors = HTTP 200 + `errors[]`. Check `extensions.code` for `UNAUTHENTICATED` / `RATE_LIMIT_EXCEEDED`.
- `glamour.TermRenderer` — width-aware, respects `NO_COLOR`. Instantiate once per pane-width change, reuse across renders. Heavy to construct.
- `modernc.org/sqlite` registers as driver `sqlite`. Open via `database/sql`. WAL set via `PRAGMA journal_mode=WAL` after `Open`.
- Vim motion parser is pure state machine — pure functions, 100% table-testable. Do not couple to Bubble Tea.
- Full TUI repaint on each keystroke is fine at 200-row lists; don't optimize until profiling.

## Requirements

### Functional (must match Ruby keymap verbatim)

| Key / Command | Action |
|---|---|
| `1` / `2` | Switch view: My Issues / Triage |
| `j` `k` | Cursor down/up |
| `gg` `G` | Jump to top / bottom |
| `Ctrl-d` `Ctrl-u` | Half-page down/up |
| `Ctrl-f` `Ctrl-b` | Page down/up |
| `{count}j` `{count}k` | Numeric prefix (e.g. `5j`) |
| `s` | Modal: state picker |
| `a` | Modal: assignee picker |
| `p` | Modal: priority picker |
| `/` | Enter live filter |
| `r` | Refresh current view |
| `:sync` / `:refresh` | Full / partial resync |
| `:view my_issues\|triage` | Switch view |
| `:state <name>` / `:assign <name\|me\|none>` / `:priority <n\|name>` | Text-mode mutations |
| `:open` | `open`/`xdg-open` on `issue.url` |
| `:q` / `:qa` / `:quit` | Quit |
| `?` | Toggle full-screen help overlay |
| `Esc` | Close modal / cancel search / dismiss error |
| `q` | Quit (from normal mode) |

### Non-functional

- Cold boot (cache hit) < 500ms (includes DB open, state restore, first paint)
- Keystroke → next paint < 16ms
- First-ever launch (full bootstrap sync) tolerable, spinner visible, partial results paint as they land
- `NO_COLOR=1` env disables all color (lipgloss + glamour respect it)
- SIGWINCH handled by Bubble Tea automatically via `tea.WindowSizeMsg`
- All panics during mutation land in status-line error banner, not crash

## Architecture

```
cmd/lnr/main.go
  │
  ├─ config.Load() → XDG paths + LINEAR_API_KEY env check
  ├─ cache.Open(path) → *sql.DB (WAL)
  ├─ cache.Migrate(db) → idempotent
  ├─ api.NewClient(key, httpClient)
  ├─ state.Restore(path) → app.Restore
  └─ tea.NewProgram(app.New(deps), tea.WithAltScreen()).Run()

app.Model (root)
  │
  ├─ currentView: my_issues | triage | help
  ├─ modal: nil | statePicker | assigneePicker | priorityPicker
  ├─ commandMode: bool  + commandBuffer
  ├─ searchMode:  bool  + searchBuffer
  ├─ motionBuf (vim parser)
  ├─ viewState (my_issues / triage sub-model)
  ├─ detailVP (bubbles/viewport)
  ├─ syncing: bool  (drives spinner)
  ├─ errorBanner: string?
  ├─ width / height (tea.WindowSizeMsg)
  ├─ viewer, teams, users (domain cache refs)
  └─ deps: api.Client, cache.Repos, sync.Service
```

### Layers

| Layer | Packages |
|---|---|
| Presentation | `internal/app`, `internal/views`, `internal/modals` |
| Domain | `internal/vimmotion`, inline `tea.Msg` types per package |
| Data | `internal/api`, `internal/cache`, `internal/sync`, `internal/config` |

### Package layout (as in spec)

```
linear-tui/
├── cmd/lnr/main.go
├── internal/
│   ├── app/         model.go update.go view.go keys.go msgs.go
│   ├── api/         client.go queries.go mutations.go errors.go
│   ├── cache/       db.go schema.go repos.go (split by entity if >200 lines)
│   ├── sync/        sync.go msgs.go
│   ├── views/       my_issues.go triage.go detail.go
│   ├── modals/      state_picker.go assignee_picker.go priority_picker.go
│   ├── vimmotion/   parser.go parser_test.go
│   ├── config/      config.go xdg.go
│   └── state/       store.go   (state.json persistence)
├── go.mod
├── go.sum
├── README.md
└── LICENSE (MIT)
```

### Key `tea.Msg` types

```go
// internal/app/msgs.go (shared, or per-package as appropriate)
type (
    bootstrapCompleteMsg struct{ viewer Viewer; teams []Team }
    myIssuesLoadedMsg    struct{ issues []Issue }
    triageLoadedMsg      struct{ teamID string; issues []Issue }
    mutationStartedMsg   struct{ issueID string; kind string }
    mutationCompleteMsg  struct{ issue Issue; err error }
    refreshDoneMsg       struct{ view string; err error }
    apiErrorMsg          struct{ err error }
    motionMsg            struct{ motion vimmotion.Motion } // emitted by parser
)
```

## Related Code Files

**Create:**
- All files in `cmd/` + `internal/` per layout above
- `go.mod` / `go.sum` via `go mod init` + `go get`
- `README.md` — install, env var, keybindings table, screenshot placeholder
- `LICENSE` (MIT)
- `.gitignore` — `lnr`, `dist/`, `*.db`, `coverage.out`
- `.github/workflows/ci.yml` — `go vet`, `go test ./...`, `staticcheck`
- `Makefile` — `build`, `test`, `run`, `lint`

**Modify:** none (fresh repo)

## Implementation Steps

### 1. Scaffold (hrs 0-2)
1. `mkdir linear-tui && cd linear-tui && go mod init github.com/<org>/linear-tui` — confirm org with Tay first
2. Create dir skeleton + empty `.go` files with correct `package` declarations
3. `cmd/lnr/main.go` prints `linear-tui v0.0.0` and exits — proves build
4. Add deps: `go get github.com/charmbracelet/bubbletea github.com/charmbracelet/bubbles github.com/charmbracelet/lipgloss github.com/charmbracelet/glamour modernc.org/sqlite`
5. `go build ./...` green
6. Add `.gitignore`, `LICENSE`, placeholder `README.md`
7. Add `Makefile` + CI workflow
8. `git init` + initial commit `chore: scaffold module layout`

### 2. Config (hrs 2-3)
1. `internal/config/xdg.go` — resolve `XDG_CONFIG_HOME` / `XDG_DATA_HOME` / `XDG_STATE_HOME` with `~/.config` / `~/.local/share` / `~/.local/state` fallbacks
2. `internal/config/config.go` — `Load()` returns struct with paths + `APIKey`. Reads `LINEAR_API_KEY`; returns typed error `ErrMissingAPIKey` when absent (actionable message: `export LINEAR_API_KEY=lin_api_...`)
3. Ensure dirs exist (`os.MkdirAll`) lazily at first use, not at `Load`
4. Table tests for XDG resolution + missing-key error

### 3. SQLite cache (hrs 3-7)
1. `internal/cache/db.go` — `Open(path string) (*sql.DB, error)` using `modernc.org/sqlite` driver. Set `PRAGMA journal_mode=WAL`, `PRAGMA synchronous=NORMAL`, `PRAGMA foreign_keys=ON`.
2. `internal/cache/schema.go` — `Migrate(db *sql.DB) error`, idempotent. Tables + indexes per spec. Add columns if missing via `ALTER TABLE` wrapped in column-exists check (query `PRAGMA table_info(...)`).
3. `internal/cache/repos.go` — `Repos` struct bundling `IssueRepo`, `TeamRepo`, `UserRepo`, `StateRepo`. Each = plain struct w/ `*sql.DB` field. Methods: `Upsert`, `FindByID`, domain queries (`AssignedTo`, `UnassignedIn`).
4. Split `repos.go` into `issue_repo.go` / `team_repo.go` / `user_repo.go` / `workflow_state_repo.go` once >200 lines.
5. Table tests per repo using in-memory `file::memory:?cache=shared` DB; round-trip upsert + query; covers indexes indirectly via `EXPLAIN QUERY PLAN` assertion for the key queries.

### 4. Linear API client (hrs 7-11)
1. `internal/api/client.go` — `NewClient(apiKey string, http *http.Client) *Client`. Default `http = &http.Client{Timeout: 30*time.Second}`.
2. `Client.Query(ctx, query string, vars map[string]any, out any) error` — marshals `{query, variables}`, POSTs to `https://api.linear.app/graphql`, unmarshals into `{data: out, errors: []}`. Typed errors:
   - `ErrUnauthenticated`, `ErrRateLimit{ResetAt time.Time}`, `ErrAPI{Code, Message}`, `ErrNetwork{Err}`
3. Rate-limit handling: on `ErrRateLimit`, return error — caller decides retry. Phase 1 caller: bootstrap retries once after sleep; user-triggered refresh surfaces banner.
4. `internal/api/queries.go` — const strings for `ViewerAndTeams`, `MyIssues`, `TeamTriageIssues`, `TeamWorkflowStates`, `WorkspaceUsers`, `IssueDetail`.
5. `internal/api/mutations.go` — const string for `IssueUpdate`.
6. Unit tests: `httptest.Server` fixture returns canned GraphQL responses; covers auth error path, rate limit path, happy path.
7. Integration test file `client_live_test.go` guarded by `//go:build live` build tag + `LINEAR_LIVE=1` env check; hits real `viewer` query only.

### 5. Sync service (hrs 11-13)
1. `internal/sync/sync.go` — `Service` holds `*api.Client` + `*cache.Repos`.
2. `Service.Bootstrap(ctx) tea.Cmd` — returns a `tea.Cmd` that runs the full sequence (viewer+teams → users → workflow_states per team → my_issues → triage per team) and emits a sequence of messages: `BootstrappedViewerMsg`, `UsersLoadedMsg`, `StatesLoadedMsg{teamID}`, `MyIssuesLoadedMsg`, `TriageLoadedMsg{teamID}`, `BootstrapDoneMsg` or `SyncErrorMsg`. Model appends each result incrementally — spinner stays on until `BootstrapDoneMsg`.
3. `Service.RefreshMyIssues(ctx) tea.Cmd`, `Service.RefreshTriage(ctx, teamID) tea.Cmd` — for `r` keybinding.
4. `Service.FullSync(ctx) tea.Cmd` — for `:sync`.
5. All writes go through repo `Upsert`. Timestamp rows with `synced_at = now()`.
6. Tests: use `httptest.Server` + real SQLite; assert DB rows post-sync.

### 6. Vim motion parser (hrs 13-15)
1. `internal/vimmotion/parser.go` — pure state machine.
2. API:
   ```go
   type Motion struct { Kind Kind; Count int } // Kind = Down/Up/Top/Bottom/HalfPageDown/HalfPageUp/PageDown/PageUp
   type Parser struct { buf []rune; count int; lastKeyAt time.Time }
   func (p *Parser) Feed(r rune, now time.Time) (Motion, bool) // true = emit
   func (p *Parser) Timeout(now time.Time) bool                // for 300ms resolution of `g`
   ```
3. 300ms ambiguity timeout: model fires a `tea.Tick(300*time.Millisecond)` after any ambiguous key; on tick, call `parser.Timeout()` to flush.
4. Table tests: every motion + count combo (`j`, `5j`, `gg`, `G`, `10G`, `Ctrl-d`, …). ~50 cases.

### 7. App root: Model/Update/View (hrs 15-19)
1. `internal/app/model.go` — `Model` struct as described in Architecture. `New(deps)` constructor. `Init() tea.Cmd` returns `sync.Bootstrap(ctx)`.
2. `internal/app/update.go` — single `Update(msg) (Model, tea.Cmd)` entrypoint; dispatches by message type:
   - `tea.KeyMsg` → `keys.Handle(m, msg)` — routes by mode (normal / command / search / modal)
   - `tea.WindowSizeMsg` → recompute list/detail widths, rebuild `glamour.TermRenderer`
   - Sync messages → merge results into model, clear spinner when `BootstrapDoneMsg`
   - `mutationCompleteMsg` → clear `syncing`, upsert issue, close modal, set error if any
   - `tea.Tick` → motion timeout
3. `internal/app/view.go` — pure `View(m) string`. Composition order: header (view name + count) → split pane (list | detail) → status line (sync state, last_synced, error banner) → optional modal overlay centered via `lipgloss.Place`, optional help overlay covering everything.
4. Keep `update.go` under 200 lines by extracting per-mode handlers to `keys.go`.

### 8. Views — list + detail (hrs 19-22)
1. `internal/views/my_issues.go` / `triage.go` — each exposes `Render(issues []Issue, cursor int, width int) string`. Shared row layout via private helper: `priorityIcon | IDENT | title | state badge | updated ago`. Lipgloss styles for priority colors (0=dim, 1=red, 2=yellow, 3=default, 4=dim-blue).
2. `internal/views/detail.go` — renders selected issue in the right pane via `bubbles/viewport`. Title (bold) + meta block + description (via `glamour.TermRenderer.Render(desc)`). Memoize rendered markdown keyed by `(issueID, width)` — invalidate on upsert or resize.
3. Split pane: `lipgloss.JoinHorizontal(lipgloss.Top, listPane, detailPane)`. Widths = `0.4*width` / `0.6*width`.
4. Status line: `[view] n/total • synced Xs ago • {mode}` + error banner (dismiss via Esc).

### 9. Modals (hrs 22-24)
1. `internal/modals/` — each picker = small Bubble Tea sub-model (Init/Update/View). Driven by `bubbles/list` (OK here — modal is simple list).
2. `state_picker.go` — items = workflow states of current issue's team.
3. `assignee_picker.go` — items = workspace users + synthetic `(unassigned)` entry.
4. `priority_picker.go` — items = fixed `urgent/high/normal/low/none` → ints `1/2/3/4/0`.
5. On Enter: picker emits `mutationRequestedMsg{issueID, kind, value}`. Root Update starts the mutation `tea.Cmd`, sets `syncing=true`, leaves modal open. On `mutationCompleteMsg` success: upsert cache + close modal. On failure: render error inside modal, keep open.
6. Esc closes modal.

### 10. Search + Command mode (hrs 24-26)
1. Search mode entered on `/`. `bubbles/textinput` drives buffer. Every keystroke recomputes filtered issue slice: case-insensitive substring on `identifier` or `title`. Enter commits (textinput closes, cursor returns to list, filter persists). Esc clears filter + exits search.
2. Header shows `Triage (3/47)` when filtered.
3. Command mode entered on `:`. `bubbles/textinput` drives buffer. Enter parses:
   - `q`/`qa`/`quit` → `tea.Quit`
   - `view <name>` → switch view
   - `state <name>` / `assign <spec>` / `priority <spec>` → resolve to ID via repos, dispatch mutation `tea.Cmd` (same pipeline as modals)
   - `open` → `exec.Command("open"|"xdg-open", issue.url).Start()` (detached)
   - `sync` / `refresh` → delegate to sync service
4. Unknown command → error banner.
5. Table tests for command parser.

### 11. State persistence (hrs 26-27)
1. `internal/state/store.go` — `Save(path, State) error`, `Load(path) (State, error)`.
2. Fields: `View string`, `CursorIndex int`, `SelectedTeamID string`, `LastSyncedAt time.Time`.
3. Hook Bubble Tea's `tea.QuitMsg` path or wrap `Program.Run()` to call `Save` on exit.
4. On `Init`, `Load` and populate Model.

### 12. Help overlay + polish (hrs 27-29)
1. `?` toggles full-screen overlay listing keybindings. Built from a static table.
2. Status-line spinner: `bubbles/spinner` ticking while `syncing=true`. Make sure tick stops when spinner hides (no wasted renders).
3. Priority colors + state-type colors tuned against dark terminal; test `NO_COLOR=1` renders ASCII-only.
4. README: install, env var, keybindings table, architecture note, screenshot placeholder.

### 13. Dogfood prep (hrs 29-30)
1. Manual test checklist: bootstrap, quit, restore, search, all 3 modals, `:open`, `:sync`, `r`, help overlay.
2. Build single binary: `go build -o lnr ./cmd/lnr`. Install to `~/bin`.
3. 5-day dogfood starts. Keep a `DOGFOOD_LOG.md` of rough edges.

## Todo List

- [x] Confirm Go module path — set to `github.com/taynguyen/linear-tui` (rename before publish if org changes)
- [x] `go mod init` + dir skeleton + CI workflow
- [x] `internal/config` + XDG tests (3 tests passing)
- [x] `internal/cache` schema + 4 repos + tests (4 tests passing, roundtrip + idempotent migrate + query filters)
- [x] `internal/api` client + typed errors + fixture tests (6 tests passing, `httptest.Server` covers happy/auth/rate-limit/malformed/variables)
- [x] `internal/sync` bootstrap + refresh + UpdateIssue as tea.Cmd
- [x] `internal/vimmotion` parser + table tests (7 tests, ~20 cases, counts + pending `g` + timeout)
- [x] `internal/app` Model/Update/View root + key routing (split across model.go / update.go / keys.go / view.go / msgs.go)
- [x] `internal/views` list + detail + glamour memoization (DetailRenderer caches per issue ID, invalidates on width change or mutation)
- [x] `internal/modals` state / assignee / priority (shared `Picker` sub-struct)
- [x] `/` search + `:` command mode (:q/:sync/:refresh/:view/:state/:assign/:priority/:open)
- [x] `internal/state` persistence (atomic tmpfile + rename, 0600 perms, 2 tests)
- [x] Help overlay (`?`) + status-line spinner + error banner with auto-dismiss
- [x] README + Makefile + CI workflow
- [x] `go test ./...` + `go vet ./...` all clean; binary builds 25MB (darwin-arm64)
- [ ] `staticcheck` — not installed locally; CI handles when added
- [ ] `//go:build live` Linear integration test
- [ ] Snapshot tests for views
- [ ] 5-day dogfood log

## Success Criteria

- `lnr` cold-boots to My Issues in <500ms (cache hit), first-ever launch completes bootstrap and renders partial results incrementally.
- Navigate 100+ issues with vim motions, no visible lag. Motion-buf resolution feels natural (`5j` jumps 5 lines, `gg` goes top).
- Change state/assignee/priority via both modal pickers and `:` commands; Linear web UI reflects the change within 1s.
- `/triage` live filter updates per keystroke; header count updates; Esc clears.
- 5 consecutive weekday mornings of standup completed without opening Linear web.
- All tests pass. `go vet` + `staticcheck` clean. `go build` cross-compiles to darwin-arm64 with no CGo warnings (verifies `modernc.org/sqlite` choice).

## Risk Assessment

| Risk | Mitigation |
|---|---|
| Goroutine sneaking into Update — breaks purity, tests flake | Lint rule: any goroutine in `internal/app` / `internal/views` is a code smell. Code review on every PR. All async must be a `tea.Cmd`. |
| `glamour.TermRenderer` construction in hot path blows 16ms budget | Construct once per width; cache rendered-markdown output per issue. Invalidate on upsert + resize only. |
| `modernc.org/sqlite` slower than CGo driver | Acceptable if boot stays <500ms. Benchmark read path with 500 issues; if close to budget, swap to `mattn/go-sqlite3` behind build tag. |
| `tea.ExecProcess` (needed Phase 2) not proven in Phase 1 — late surprise | Phase 1 doesn't need it; delay risk to Phase 2. |
| Linear API schema drift | Pin query constants; integration test against live API gated by `LINEAR_LIVE=1`, run weekly. |
| SIGWINCH storm during terminal resize causes flicker | Bubble Tea batches `tea.WindowSizeMsg`. Rebuild `glamour.TermRenderer` only on width change, not height. |
| Ruby cache.db schema incompat | Migrations are idempotent + additive; column-exists check before ALTER. Test side-by-side on a Ruby-produced DB file. |
| Scope creep into Phase 2 features | Hard stop: no `c`, no `o`, no `i`, no `:team` in Phase 1. |

## Security Considerations

- API key from env var only. Never written to cache.db or state.json.
- `state.json` opened with `0600` perms.
- Panic in `tea.Cmd` must not dump stack to stderr (Bubble Tea eats TUI output); recover and convert to `apiErrorMsg`.
- `:open` shells out with the URL as a single argv — no shell interpolation.
- Integration test file scrubs API key from any logged output.

## Next Steps

- Unblocks: Phase 2 (edit + capture) — shares mutation pipeline, `tea.ExecProcess` pattern
- Phase 2 depends on: stable Phase 1 mutation flow, width-aware glamour rendering (for description preview after `o` edit), search performance baseline
