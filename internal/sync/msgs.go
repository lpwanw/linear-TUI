package sync

import (
	"github.com/taynguyen/linear-tui/internal/cache"
)

// Messages emitted by sync commands. All are tea.Msg (empty interface).

type MyIssuesLoadedMsg struct{ Issues []cache.Issue }

type TriageLoadedMsg struct {
	TeamID string
	Issues []cache.Issue
}

type RefreshDoneMsg struct {
	View string
	Err  error
}

type MutationDoneMsg struct {
	Issue *cache.Issue
	Err   error
}

type SyncErrorMsg struct{ Err error }
