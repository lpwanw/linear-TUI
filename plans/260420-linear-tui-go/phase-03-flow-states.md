# Phase 3 — Flow States

**Priority:** P1
**Effort:** ~20 hrs over 1-2 weeks
**Status:** blocked (on Phase 2 DoD)

## Context Links

- [Plan overview](./plan.md)
- [Phase 2](./phase-02-edit-capture.md)
- [Linear GraphQL research](../../../linear_tui/plans/260420-linr-ruby-tui-linear/research/researcher-01-linear-graphql-api.md) — cycles, comments

## Overview

Close the dev-loop. Current cycle view (`3`), `:branch` + `:pr` shell-outs, `K` view comments, `C` add comment via `$EDITOR`, `:recent`, optional 60s background poll. This is the "staying in terminal all day" phase.

## Key Insights

- Cycles: `team.activeCycle.issues` — one GraphQL query per team per refresh. Cache in `cycles` table with `(id, team_id, starts_at, ends_at)`.
- Branch naming convention: `{identifier-lowercased}-{title-slug}` (e.g. `eng-42-fix-null-pointer-in-sync`). Slug = lowercased, spaces → `-`, strip anything not `[a-z0-9-]`, trim to 60 chars.
- `gh pr create` exits 0 on success, prints PR URL. Capture stdout in `tea.ExecProcess` result msg and surface URL in status banner.
- Comments: `issue.comments(first: 10)` in query already landed in Phase 1 `IssueDetail` query (unused so far). Mutation `commentCreate(input: {issueId, body})` needed.
- Background poll: `tea.Tick(60*time.Second, func(t) tea.Msg { return pollTickMsg{} })`. On tick, dispatch a lightweight `sync.RefreshMyIssues` if idle (not syncing, not in modal). Cancel by not re-returning the Tick command.
- `:recent` = filter on `updatedAt > now - 7d` client-side against cache — no new query needed; cache refresh covers it.

## Requirements

### Functional

| Key / Command | Action |
|---|---|
| `3` | Switch to Current Cycle view (active cycle issues for selected team) |
| `K` | Toggle comments panel in detail pane |
| `C` | Add comment: `$EDITOR` tempfile → `commentCreate` |
| `:branch` | Shell out `git checkout -b {slug}`; on success, confirmation banner |
| `:pr` | Shell out `gh pr create --title "{identifier}: {title}" --body "Closes {identifier}"`; banner with resulting URL |
| `:recent` | Filter mode: issues with `updatedAt` in last 7 days, across all cached |

### Non-functional

- Background poll optional; off by default, enabled via `LNR_POLL=1` env. Zero visible repaint when poll occurs and no issues changed.
- `:branch` / `:pr` work from any CWD that's a git repo; error surfaces cleanly otherwise.
- Comments: `$EDITOR` empty-body → no-op (no empty comment creation).

## Architecture

New cache table:
```sql
CREATE TABLE IF NOT EXISTS cycles (
  id TEXT PRIMARY KEY,
  team_id TEXT NOT NULL REFERENCES teams(id),
  number INTEGER NOT NULL,
  name TEXT,
  starts_at TEXT,
  ends_at TEXT,
  synced_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_cycles_team ON cycles(team_id);
```

`issues` gains optional `cycle_id` column (ALTER TABLE, idempotent):
```sql
ALTER TABLE issues ADD COLUMN cycle_id TEXT REFERENCES cycles(id);
CREATE INDEX IF NOT EXISTS idx_issues_cycle ON issues(cycle_id);
```

New API:
- `internal/api/queries.go` — `ActiveCycleIssues`, extend `IssueDetail` usage to expose comments
- `internal/api/mutations.go` — `CommentCreate`

New messages:
```go
cycleLoadedMsg   struct{ teamID string; issues []Issue }
commentsLoadedMsg struct{ issueID string; comments []Comment }
commentPostedMsg  struct{ issueID string; comment Comment; err error }
branchCreatedMsg  struct{ name string; err error }
prCreatedMsg      struct{ url string; err error }
pollTickMsg       struct{}
```

## Related Code Files

**Create:**
- `internal/views/cycle.go` — cycle view
- `internal/views/comments.go` — comments panel renderer
- `internal/app/git.go` — branch/PR shell-out helpers

**Modify:**
- `internal/cache/schema.go` — cycles table + `issues.cycle_id`
- `internal/cache/repos.go` — `CycleRepo`; `IssueRepo` query-by-cycle + query-recent
- `internal/sync/sync.go` — `RefreshCycle(ctx, teamID) tea.Cmd`, `LoadComments(ctx, issueID) tea.Cmd`, `PostComment(ctx, issueID, body) tea.Cmd`
- `internal/api/queries.go` + `mutations.go`
- `internal/app/keys.go` — `3`, `K`, `C`
- `internal/app/update.go` — new msg handling + optional background poll
- `internal/app/view.go` — cycle view + comments panel overlay

## Implementation Steps

### 1. Cycles (hrs 0-6)
1. Cache schema migration (idempotent `ALTER TABLE`).
2. `CycleRepo` — upsert, find-by-team, query active cycle.
3. GraphQL query `ActiveCycleIssues($teamId)` — returns `team.activeCycle.{id, number, name, startsAt, endsAt, issues.nodes {…same issue shape…}}`.
4. `sync.RefreshCycle(ctx, teamID) tea.Cmd` — upserts cycle + issues with `cycle_id` populated.
5. `internal/views/cycle.go` — same row layout as my_issues but sourced from cached issues where `cycle_id = currentCycle.id`. Header shows `Cycle #{n} · {daysRemaining}d left`.
6. Bind `3` key → switch view + dispatch refresh if stale (>5min since last cycle sync).
7. Tests: cycle repo roundtrip, cycle query fixture, view render snapshot.

### 2. Comments (hrs 6-11)
1. Comment model struct: `{ID, IssueID, Body, AuthorName, CreatedAt}`.
2. Cache table (optional in phase 3 — can store in-memory per issue and re-fetch on `K`):
   ```sql
   CREATE TABLE IF NOT EXISTS comments (
     id TEXT PRIMARY KEY,
     issue_id TEXT NOT NULL REFERENCES issues(id),
     body TEXT,
     author_id TEXT,
     created_at TEXT,
     synced_at TEXT
   );
   CREATE INDEX IF NOT EXISTS idx_comments_issue ON comments(issue_id);
   ```
3. `sync.LoadComments(ctx, issueID)` — uses existing `IssueDetail` query; upserts comments.
4. `K` toggles comments panel in detail pane. Layout: description on top 60%, comments list below 40% (scrollable via `bubbles/viewport`). Lazy-load on first open.
5. `C` opens `$EDITOR` tempfile (reuse Phase 2 editor helper); on exit with non-empty body, dispatch `sync.PostComment`.
6. `commentCreate` mutation: `mutation { commentCreate(input: {issueId, body}) { success, comment {id body createdAt user{name}} } }`.
7. Tests: comment upsert, panel render, post comment happy path.

### 3. `:branch` (hrs 11-13)
1. Parse current issue → compute slug: `{identifier lowercase}-{title slug}`. Slug fn is pure, table-testable.
2. `internal/app/git.go` — `createBranch(name string) tea.Cmd`:
   - Check `git rev-parse --is-inside-work-tree` — if not a repo, emit error msg
   - Run `git checkout -b <name>` via `exec.Command` (no need for `tea.ExecProcess` — quick non-interactive)
   - Emit `branchCreatedMsg`
3. On success: banner `Created branch eng-42-fix-sync` for 5s.
4. On failure: banner with error (e.g. branch exists, not a repo).
5. Unit test slug fn. Integration test skipped (requires git repo fixture).

### 4. `:pr` (hrs 13-15)
1. Check `gh` in PATH. If missing, banner with install hint.
2. Build args: `gh pr create --title "{identifier}: {title}" --body "Closes {identifier}"`. `gh` itself handles auth.
3. `exec.Command("gh", …)`, capture stdout. Parse last line for URL (simple `http` prefix check).
4. Emit `prCreatedMsg{url, err}`. On success, banner with URL; copy to clipboard if `$CLIPBOARD_CMD` or common `pbcopy`/`wl-copy`/`xclip` available (best-effort).
5. Tests: arg-building pure fn.

### 5. `:recent` (hrs 15-16)
1. Command parser: `recent` → set `recentFilter=true`.
2. View filters cached issues by `updatedAt > now - 7d` (union across all cached, not just selected team).
3. Header: `Recent (n)`.

### 6. Background poll (hrs 16-18)
1. If `LNR_POLL=1`, `Init()` returns a batch of bootstrap + `tea.Tick(60s, pollTickMsg)`.
2. On `pollTickMsg` — if `syncing==false && modal==nil`, dispatch `sync.RefreshMyIssues`. Always re-return the Tick.
3. Merge results with cached. Only repaint if any issue's `updatedAt` changed.
4. Status-line indicator: small dot `•` after `synced Xs ago` when a poll is in-flight.

### 7. Polish + dogfood (hrs 18-20)
1. Help overlay update — `3`, `K`, `C`, `:branch`, `:pr`, `:recent`.
2. Cycle view color: dim issues past `endsAt`.
3. `DOGFOOD_LOG.md` for one more week. Track: did I do an entire PR cycle without leaving `lnr`?

## Todo List

- [ ] Cache migration: `cycles` table + `issues.cycle_id`
- [ ] `CycleRepo` + tests
- [ ] `ActiveCycleIssues` query + fixture test
- [ ] `views/cycle.go` + `3` key binding
- [ ] Comments: cache + `K` toggle + lazy load
- [ ] `commentCreate` mutation + `C` `$EDITOR` flow
- [ ] `:branch` — slug fn + git shell-out
- [ ] `:pr` — `gh` shell-out + URL parse + optional clipboard
- [ ] `:recent` filter
- [ ] `LNR_POLL=1` background poll
- [ ] Help overlay + DOGFOOD log

## Success Criteria

- Full PR cycle in `lnr`: pick triage issue → `s` assign to me → `:branch` → code in `$EDITOR` (not in `lnr`) → `:pr` → URL in banner. Time from triage to PR < 2min.
- `3` cycle view loads in <200ms after first sync; stale data triggers background refresh without blocking UI.
- `C` + `$EDITOR` posts comment; appears in panel on next `K` toggle.
- Background poll with `LNR_POLL=1` runs for 1hr without visible jank.
- Zero opens of Linear web in a full workweek of coding.

## Risk Assessment

| Risk | Mitigation |
|---|---|
| `gh` not installed → bad UX | Pre-flight check in `:pr` with install hint; `lnr doctor` covers this in Phase 4 |
| Background poll fires during modal → weird repaint | Gate on `syncing==false && modal==nil`; queue and re-tick if busy |
| Comment query grows as issue gets noisy | Paginate (already `first: 10` in query); `K` has "load more" action if needed — defer to Phase 4 |
| `:branch` conflicts with existing branch | Rely on `git` error; surface cleanly in banner |
| Slug fn edge cases (unicode titles) | Unicode → lowercase via `strings.ToLower`, drop non-`[a-z0-9-]` after |

## Security Considerations

- `:branch` / `:pr` shell-outs use `exec.Command` with args slice — no shell interpolation possible even with exotic titles.
- `gh` auth is external; `lnr` never sees the token.
- Background poll logs no data; only flips the status-line dot.

## Next Steps

- Unblocks: Phase 4 — once 3 weeks of Phase 3 dogfood pass, freeze scope and ship
- Phase 4 depends on: stable background poll behavior, full PR workflow proven, no crashes in 3-week window
