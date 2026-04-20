package app

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/taynguyen/linear-tui/internal/cache"
	"github.com/taynguyen/linear-tui/internal/sync"
)

// Update is Bubble Tea's pure dispatch entrypoint.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
		_, detailW := paneWidths(m.width)
		m.detailR.SetWidth(detailW - 2)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(v)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case enterSyncingMsg:
		m.syncing = true
		return m, nil

	case MotionTimeoutMsg:
		m.parser.Timeout(v.At)
		return m, nil

	case BannerClearMsg:
		m.errorBanner = ""
		m.infoBanner = ""
		return m, nil

	case sync.BootstrapResultMsg:
		m.viewerID = v.ViewerID
		m.teams = v.Teams
		if m.selectedTeamID == "" && len(m.teams) > 0 {
			m.selectedTeamID = m.teams[0].ID
		}
		m.users = v.Users
		for _, u := range v.Users {
			m.userByID[u.ID] = u
		}
		for _, states := range v.States {
			for _, s := range states {
				m.stateByID[s.ID] = s
			}
		}
		m.issuesMy = v.MyIssues
		for teamID, list := range v.TeamTriage {
			m.issuesTriage[teamID] = list
		}
		m.syncing = false
		m.lastSyncedAt = time.Now()
		m.clampCursor()
		return m, nil

	case sync.MyIssuesLoadedMsg:
		m.issuesMy = v.Issues
		m.syncing = false
		m.lastSyncedAt = time.Now()
		m.clampCursor()
		return m, nil

	case sync.TriageLoadedMsg:
		m.issuesTriage[v.TeamID] = v.Issues
		m.syncing = false
		m.lastSyncedAt = time.Now()
		if v.TeamID == m.selectedTeamID {
			m.clampCursor()
		}
		return m, nil

	case sync.RefreshDoneMsg:
		m.syncing = false
		m.lastSyncedAt = time.Now()
		if v.Err != nil {
			m.errorBanner = "refresh failed: " + v.Err.Error()
			return m, clearBannerAfter(4 * time.Second)
		}
		return m, nil

	case sync.MutationDoneMsg:
		m.syncing = false
		m.mutatingID = ""
		if v.Err != nil {
			m.errorBanner = "mutation failed: " + v.Err.Error()
			return m, clearBannerAfter(4 * time.Second)
		}
		m.modal = nil
		if v.Issue != nil {
			m.applyIssueUpdate(*v.Issue)
			m.detailR.Invalidate(v.Issue.ID)
		}
		return m, nil

	case sync.SyncErrorMsg:
		m.syncing = false
		m.errorBanner = "sync error: " + v.Err.Error()
		return m, clearBannerAfter(4 * time.Second)
	}
	return m, nil
}

// applyIssueUpdate patches an updated issue into the in-memory slices.
// Mutations can change state or assignee, so an issue may move in/out of each view's set.
func (m *Model) applyIssueUpdate(updated cache.Issue) {
	// My Issues: remove if completed/canceled, otherwise upsert.
	stateType := ""
	if ws, ok := m.stateByID[updated.StateID]; ok {
		stateType = ws.Type
	}
	finished := stateType == cache.StateTypeCompleted || stateType == cache.StateTypeCanceled

	if updated.AssigneeID == m.viewerID && m.viewerID != "" && !finished {
		m.issuesMy = upsertIssue(m.issuesMy, updated)
	} else {
		m.issuesMy = removeIssue(m.issuesMy, updated.ID)
	}

	// Triage: keep if team matches and unassigned and in triage/backlog.
	for teamID, list := range m.issuesTriage {
		if updated.TeamID != teamID {
			m.issuesTriage[teamID] = removeIssue(list, updated.ID)
			continue
		}
		inTriage := (stateType == cache.StateTypeTriage || stateType == cache.StateTypeBacklog)
		if updated.AssigneeID == "" && inTriage {
			m.issuesTriage[teamID] = upsertIssue(list, updated)
		} else {
			m.issuesTriage[teamID] = removeIssue(list, updated.ID)
		}
	}
	m.clampCursor()
}

func upsertIssue(list []cache.Issue, i cache.Issue) []cache.Issue {
	for idx := range list {
		if list[idx].ID == i.ID {
			list[idx] = i
			return list
		}
	}
	return append(list, i)
}

func removeIssue(list []cache.Issue, id string) []cache.Issue {
	for idx := range list {
		if list[idx].ID == id {
			return append(list[:idx], list[idx+1:]...)
		}
	}
	return list
}

func clearBannerAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return BannerClearMsg{} })
}

func paneWidths(w int) (list, detail int) {
	if w <= 0 {
		return 0, 0
	}
	list = int(float64(w) * 0.4)
	if list < 20 {
		list = 20
	}
	detail = w - list
	if detail < 20 {
		detail = w - 20
		list = 20
	}
	return
}
