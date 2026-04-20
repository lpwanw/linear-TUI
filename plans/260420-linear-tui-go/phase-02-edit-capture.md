# Phase 2 — Edit + Capture

**Priority:** P1
**Effort:** ~20 hrs over 1-2 weeks
**Status:** blocked (on Phase 1 DoD)

## Context Links

- [Plan overview](./plan.md)
- [Phase 1](./phase-01-scaffold-triage-mvp.md)
- [Linear GraphQL research](../../../linear_tui/plans/260420-linr-ruby-tui-linear/research/researcher-01-linear-graphql-api.md) — `issueCreate` mutation section

## Overview

Add write-heavy flows. `c` quick capture, `o` description edit in `$EDITOR`, `i` inline title edit, `:team <KEY>` triage team switch. Prove `tea.ExecProcess` terminal suspend/resume before Phase 3 needs it for git + gh.

## Key Insights

- `tea.ExecProcess(cmd, func(err error) tea.Msg)` — releases the terminal, runs the external process, re-acquires alt-screen on return. Bubble Tea handles the alt-screen dance; user code just needs to emit a result msg.
- `$EDITOR` workflow: write tempfile → `exec.Command(editor, tempfile)` via `tea.ExecProcess` → on exit, read tempfile → dispatch mutation → clean up tempfile.
- `bubbles/textinput` handles inline edit. For quick capture, a small 3-state sub-model (title → team → submit) is simpler than a full form framework.
- `issueCreate` mutation is minimal: `{input: {teamId, title}}` suffices. Do not collect state/priority/assignee at capture — set them after via existing pickers. Keeps capture fast.
- `:team <KEY>` only changes the active triage team; does not need re-sync of everything, only `RefreshTriage(teamID)` for the new team if not cached.

## Requirements

### Functional

| Key / Command | Action |
|---|---|
| `c` | Quick capture: prompt title → pick team → submit → `issueCreate` |
| `o` | Open selected issue's description in `$EDITOR`; on save, call `issueUpdate` |
| `i` | Inline title edit (textinput overlay on list row); Enter commits via `issueUpdate` |
| `:team <KEY>` | Switch triage team to team with matching `key` (e.g. `BUT`); restore last cursor position for that team |

### Non-functional

- `$EDITOR` suspend/resume works on iTerm2, Alacritty, and inside tmux
- Terminal state clean after `$EDITOR` exits (no leftover cursor, no alt-screen desync)
- Inline edit does not freeze other keystrokes (impossible by design — textinput is just another mode)
- Capture flow <3s end-to-end (title typed → issue created in Linear)

## Architecture

New modes:
- `captureMode` — 3-state: `typingTitle` → `pickingTeam` → `submitting`
- `editMode` — ephemeral; active only while `$EDITOR` runs (blocking)
- `inlineEditMode` — textinput overlaying the current list row

New messages:
```go
editorExitedMsg     struct{ issueID string; newDescription string; err error }
issueCreatedMsg     struct{ issue Issue; err error }
titleEditedMsg      struct{ issueID string; newTitle string }
teamSwitchedMsg     struct{ teamID string }
```

New API:
- `internal/api/mutations.go` — add `IssueCreate` const
- `internal/sync/sync.go` — add `CreateIssue(ctx, input) tea.Cmd`, `UpdateDescription(ctx, id, desc) tea.Cmd`

## Related Code Files

**Create:**
- `internal/modals/capture.go` — 3-state capture sub-model
- `internal/app/editor.go` — `$EDITOR` suspend/resume helpers (tempfile mgmt)

**Modify:**
- `internal/app/keys.go` — bind `c` / `o` / `i`
- `internal/app/update.go` — handle new msg types
- `internal/app/view.go` — render capture modal + inline textinput
- `internal/api/mutations.go` — add `IssueCreate` const
- `internal/sync/sync.go` — new methods
- `internal/views/triage.go` — respect active team from model
- `cmd/lnr/main.go` — no change

## Implementation Steps

### 1. `$EDITOR` integration (hrs 0-5)
1. `internal/app/editor.go` — `OpenDescriptionEditor(issueID, currentDesc string) tea.Cmd`:
   - Resolve editor: `$EDITOR` → `$VISUAL` → `vi`
   - Create tempfile `${TMPDIR}/lnr-{issueID}-{nonce}.md` with `0600`
   - Write current description
   - Build `exec.Command(editor, tempfile)`; wire `cmd.Stdin/Stdout/Stderr` to `/dev/tty`
   - Return `tea.ExecProcess(cmd, func(err error) tea.Msg { … read file … return editorExitedMsg })`
   - In message handler: if unchanged, no-op; else dispatch `sync.UpdateDescription` mutation
2. Teardown: on `editorExitedMsg` (success or error), `os.Remove(tempfile)` always.
3. Manual tests: iTerm2, Alacritty, tmux inside each. Verify alt-screen restored, cursor visible, no stray escape codes.
4. Unit test the tempfile path + nonce logic; skip the exec portion.

### 2. Description update mutation (hrs 5-7)
1. Reuse existing `IssueUpdate` mutation const (supports `description` field).
2. `sync.UpdateDescription(ctx, issueID, newDesc) tea.Cmd` wraps it; emits `mutationCompleteMsg`.
3. Upsert issue on success (existing path).
4. Test: `httptest.Server` + real SQLite roundtrip.

### 3. `c` quick capture (hrs 7-11)
1. `internal/modals/capture.go` — sub-model with 3 states:
   - `typingTitle`: `bubbles/textinput` — Enter advances, Esc cancels
   - `pickingTeam`: `bubbles/list` of workspace teams — Enter submits, Esc → back to title
   - `submitting`: spinner, disabled input
2. On submit, emit `captureSubmitMsg{title, teamID}`. Root Update dispatches `sync.CreateIssue(ctx, input)`, keeps modal in `submitting` state.
3. On `issueCreatedMsg{issue, nil}`: upsert, close modal, maybe jump cursor to new issue.
4. On `issueCreatedMsg{issue, err}`: show error inside modal, return to `typingTitle` for retry.
5. Add `IssueCreate` mutation const to `internal/api/mutations.go`.

### 4. `i` inline title edit (hrs 11-14)
1. On `i`, enter `inlineEditMode`: render `bubbles/textinput` in place of the cursor-row's title cell, pre-filled with current title.
2. Enter → dispatch `sync.UpdateTitle(ctx, id, title)` (reuses `issueUpdate` w/ `title` field).
3. Esc → cancel.
4. Rendering: adjust `views/*_render` to accept an optional "editing row" hook; if row index matches, render textinput instead of the title cell. Keep table width alignment.
5. Test: command parser, row-override rendering (snapshot style — strip ANSI).

### 5. `:team <KEY>` switch (hrs 14-16)
1. Command parser: `team <KEY>` looks up `teams` by key (case-insensitive). Unknown key → error banner.
2. On match: set `selectedTeamID`, persist in `state.json`, re-render triage. If no cached issues for that team, dispatch `sync.RefreshTriage(ctx, teamID)`; show spinner.
3. Cursor position: remember per-team cursor in a `map[string]int` on Model so `:team BUT` / `:team FEAT` flipping preserves place. Persist with state.json.

### 6. Polish + help overlay update (hrs 16-18)
1. Add new bindings to `?` help overlay.
2. Capture flow: confirmation line after creation `Created ENG-123`.
3. `$EDITOR` flow: if file empty on exit, treat as cancel (no mutation).
4. Lockout: while modal is in `submitting`, ignore all input except Esc (cancels only if mutation hasn't started yet).

### 7. Dogfood (hrs 18-20)
1. One week of capture-heavy use: every triage issue gets captured via `c`, every standup update happens via `o` or `i`.
2. Log friction in `DOGFOOD_LOG.md`.
3. Adjust based on pain points.

## Todo List

- [ ] `internal/app/editor.go` — `tea.ExecProcess` wrapper + tempfile mgmt
- [ ] Manual test $EDITOR on iTerm2 + Alacritty + tmux
- [ ] `sync.UpdateDescription` / `UpdateTitle` / `CreateIssue` methods + tests
- [ ] `internal/modals/capture.go` — 3-state sub-model
- [ ] Inline title edit (`i`)
- [ ] `:team <KEY>` command + per-team cursor persistence
- [ ] `IssueCreate` const + client test
- [ ] Help overlay update
- [ ] 1-week dogfood log

## Success Criteria

- `c` flow creates issue in Linear inside 3s (typed → confirmation)
- `o` suspends TUI cleanly; resume shows updated description in detail pane
- `i` edits title inline with no visual jank; Enter saves, Esc cancels
- `:team BUT` switches triage view; cursor restored if previously visited
- Terminal state intact after any editor exit (alt-screen off → on seamless)
- All Phase 1 tests still green; new tests added for capture + editor wrapper
- 1 consecutive week of no-fallback dogfood — no "had to open web UI for X"

## Risk Assessment

| Risk | Mitigation |
|---|---|
| `tea.ExecProcess` leaves tmux in broken state | Test early + often on tmux 3.x; add fallback `lnr doctor` check in Phase 4 for terminfo sanity |
| Description edit race: user edits while API refresh overwrites locally | Re-fetch latest description into tempfile right before opening editor; warn if remote changed while editing |
| Tempfile leak on crash | Register `defer os.Remove` in both success + error paths; sweep `${TMPDIR}/lnr-*.md` on startup |
| Capture flow trap: bad API key first noticed here | Phase 1 already validates at bootstrap; fallback error modal w/ copyable message |
| Inline edit width glitch — textinput exceeds row width | Clip textinput width to title column; truncate display, keep full buffer |

## Security Considerations

- Tempfile path under `$TMPDIR` with `0600` perms + random nonce
- Never log tempfile contents (could contain sensitive issue data)
- `$EDITOR` value not validated — user's env, user's problem (documented in README)

## Next Steps

- Unblocks: Phase 3 — `:branch`, `:pr`, `C` (comment) all reuse `tea.ExecProcess` pattern + the editor-tempfile helpers
- Phase 3 depends on: stable `tea.ExecProcess` flow, `IssueCreate` + `IssueUpdate` pipelines proven
