package app

import (
	"context"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/taynguyen/linear-tui/internal/cache"
	"github.com/taynguyen/linear-tui/internal/modals"
	"github.com/taynguyen/linear-tui/internal/vimmotion"
)

func (m Model) handleKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Command mode has highest priority — typed text shouldn't leak to motions.
	if m.cmdActive {
		return m.handleCmdKey(k)
	}
	if m.searchActive {
		return m.handleSearchKey(k)
	}
	if m.modal != nil {
		return m.handleModalKey(k)
	}
	if m.helpVisible {
		if isEscOrQuestion(k) {
			m.helpVisible = false
		}
		return m, nil
	}
	return m.handleNormalKey(k)
}

func isEscOrQuestion(k tea.KeyMsg) bool {
	if k.Type == tea.KeyEsc {
		return true
	}
	return k.String() == "?"
}

// ------- Normal mode -------

func (m Model) handleNormalKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case ":":
		m.cmdActive = true
		m.cmdInput.SetValue("")
		m.cmdInput.Focus()
		return m, nil
	case "/":
		m.searchActive = true
		m.searchInput.SetValue(m.searchQuery)
		m.searchInput.Focus()
		return m, nil
	case "?":
		m.helpVisible = true
		return m, nil
	case "esc":
		m.searchQuery = ""
		m.errorBanner = ""
		m.clampCursor()
		return m, nil
	case "1":
		m.persistCursor()
		m.view = ViewMyIssues
		m.restoreCursor()
		return m, nil
	case "2":
		m.persistCursor()
		m.view = ViewTriage
		m.restoreCursor()
		return m, nil
	case "r":
		return m.dispatchRefresh()
	case "s":
		return m.openStatePicker(), nil
	case "a":
		return m.openAssigneePicker(), nil
	case "p":
		return m.openPriorityPicker(), nil
	}
	return m.handleMotionKey(k)
}

func (m Model) handleMotionKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	vkey, ok := toVimKey(k)
	if !ok {
		return m, nil
	}
	now := time.Now()
	motion, emit := m.parser.Feed(vkey, now)
	if !emit {
		if m.parser.Pending() {
			return m, tea.Tick(vimmotion.AmbiguityTimeout+10*time.Millisecond, func(t time.Time) tea.Msg {
				return MotionTimeoutMsg{At: t}
			})
		}
		return m, nil
	}
	m.applyMotion(motion)
	return m, nil
}

func toVimKey(k tea.KeyMsg) (vimmotion.Key, bool) {
	switch k.Type {
	case tea.KeyCtrlD:
		return vimmotion.CtrlKey('d'), true
	case tea.KeyCtrlU:
		return vimmotion.CtrlKey('u'), true
	case tea.KeyCtrlF:
		return vimmotion.CtrlKey('f'), true
	case tea.KeyCtrlB:
		return vimmotion.CtrlKey('b'), true
	case tea.KeyRunes:
		if len(k.Runes) == 1 {
			r := k.Runes[0]
			return vimmotion.RuneKey(r), true
		}
	case tea.KeySpace:
		// unused; ignore
	}
	return vimmotion.Key{}, false
}

func (m *Model) applyMotion(motion vimmotion.Motion) {
	issues := m.currentIssues()
	n := len(issues)
	if n == 0 {
		m.cursor = 0
		return
	}
	halfPage := m.listPageSize() / 2
	if halfPage < 1 {
		halfPage = 1
	}
	page := m.listPageSize()
	if page < 1 {
		page = 1
	}
	count := motion.Count
	if count < 1 {
		count = 1
	}
	switch motion.Kind {
	case vimmotion.KindDown:
		m.cursor += count
	case vimmotion.KindUp:
		m.cursor -= count
	case vimmotion.KindTop:
		m.cursor = 0
	case vimmotion.KindBottom:
		m.cursor = n - 1
	case vimmotion.KindHalfPageDown:
		m.cursor += halfPage * count
	case vimmotion.KindHalfPageUp:
		m.cursor -= halfPage * count
	case vimmotion.KindPageDown:
		m.cursor += page * count
	case vimmotion.KindPageUp:
		m.cursor -= page * count
	}
	m.clampCursor()
}

func (m *Model) listPageSize() int {
	// Reserve 2 lines for header + 2 for status/banner.
	h := m.height - 4
	if h < 1 {
		return 10
	}
	return h
}

func (m *Model) persistCursor() {
	if m.view == ViewTriage && m.selectedTeamID != "" {
		m.cursorByTeam[m.selectedTeamID] = m.cursor
	}
}

func (m *Model) restoreCursor() {
	if m.view == ViewTriage && m.selectedTeamID != "" {
		if v, ok := m.cursorByTeam[m.selectedTeamID]; ok {
			m.cursor = v
		} else {
			m.cursor = 0
		}
	}
	m.clampCursor()
}

// ------- Refresh -------

func (m Model) dispatchRefresh() (tea.Model, tea.Cmd) {
	m.syncing = true
	ctx := context.Background()
	switch m.view {
	case ViewMyIssues:
		return m, m.deps.Sync.RefreshMyIssues(ctx)
	case ViewTriage:
		if m.selectedTeamID == "" {
			m.syncing = false
			m.errorBanner = "no team selected"
			return m, clearBannerAfter(3 * time.Second)
		}
		return m, m.deps.Sync.RefreshTriage(ctx, m.selectedTeamID)
	}
	m.syncing = false
	return m, nil
}

// ------- Modal openers -------

func (m Model) openStatePicker() Model {
	iss := m.selectedIssue()
	if iss == nil {
		m.errorBanner = "no issue selected"
		return m
	}
	states, _ := m.deps.Repos.States.ByTeam(iss.TeamID)
	items := make([]modals.Item, 0, len(states))
	for _, s := range states {
		items = append(items, modals.Item{ID: s.ID, Label: s.Name + "  " + s.Type, Value: s.ID})
	}
	m.modal = &modals.Picker{Kind: modals.KindState, Title: "Change state", Items: items}
	return m
}

func (m Model) openAssigneePicker() Model {
	iss := m.selectedIssue()
	if iss == nil {
		m.errorBanner = "no issue selected"
		return m
	}
	users, _ := m.deps.Repos.Users.All()
	items := []modals.Item{{ID: "", Label: "(unassigned)", Value: nil}}
	for _, u := range users {
		items = append(items, modals.Item{ID: u.ID, Label: u.Name, Value: u.ID})
	}
	m.modal = &modals.Picker{Kind: modals.KindAssignee, Title: "Change assignee", Items: items}
	return m
}

func (m Model) openPriorityPicker() Model {
	iss := m.selectedIssue()
	if iss == nil {
		m.errorBanner = "no issue selected"
		return m
	}
	items := []modals.Item{
		{Label: "urgent", Value: cache.PriorityUrgent},
		{Label: "high", Value: cache.PriorityHigh},
		{Label: "normal", Value: cache.PriorityNormal},
		{Label: "low", Value: cache.PriorityLow},
		{Label: "none", Value: cache.PriorityNone},
	}
	m.modal = &modals.Picker{Kind: modals.KindPriority, Title: "Change priority", Items: items}
	return m
}

// ------- Modal key handling -------

func (m Model) handleModalKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.mutatingID != "" {
		// While mutation is in flight, only Esc (ignored — API in progress) handled.
		return m, nil
	}
	switch k.String() {
	case "esc":
		m.modal = nil
		return m, nil
	case "j", "down":
		m.modal.Down()
		return m, nil
	case "k", "up":
		m.modal.Up()
		return m, nil
	case "enter":
		return m.commitModal()
	}
	return m, nil
}

func (m Model) commitModal() (tea.Model, tea.Cmd) {
	iss := m.selectedIssue()
	if iss == nil || m.modal == nil {
		m.modal = nil
		return m, nil
	}
	sel := m.modal.Selected()
	if sel == nil {
		m.modal = nil
		return m, nil
	}
	var input map[string]any
	switch m.modal.Kind {
	case modals.KindState:
		input = map[string]any{"stateId": sel.Value}
	case modals.KindAssignee:
		if sel.Value == nil {
			input = map[string]any{"assigneeId": nil}
		} else {
			input = map[string]any{"assigneeId": sel.Value}
		}
	case modals.KindPriority:
		input = map[string]any{"priority": sel.Value}
	}
	m.mutatingID = iss.ID
	m.syncing = true
	return m, m.deps.Sync.UpdateIssue(context.Background(), iss.ID, input)
}

// ------- Search key handling -------

func (m Model) handleSearchKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.Type {
	case tea.KeyEsc:
		m.searchActive = false
		m.searchQuery = ""
		m.searchInput.Blur()
		m.clampCursor()
		return m, nil
	case tea.KeyEnter:
		m.searchActive = false
		m.searchQuery = strings.TrimSpace(m.searchInput.Value())
		m.searchInput.Blur()
		m.cursor = 0
		m.clampCursor()
		return m, nil
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(k)
	m.searchQuery = m.searchInput.Value()
	m.cursor = 0
	m.clampCursor()
	return m, cmd
}

// ------- Command mode key handling -------

func (m Model) handleCmdKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.Type {
	case tea.KeyEsc:
		m.cmdActive = false
		m.cmdInput.Blur()
		return m, nil
	case tea.KeyEnter:
		cmdText := strings.TrimSpace(m.cmdInput.Value())
		m.cmdActive = false
		m.cmdInput.Blur()
		return m.runCommand(cmdText)
	}
	var cmd tea.Cmd
	m.cmdInput, cmd = m.cmdInput.Update(k)
	return m, cmd
}

func (m Model) runCommand(text string) (tea.Model, tea.Cmd) {
	if text == "" {
		return m, nil
	}
	parts := strings.Fields(text)
	head := parts[0]
	args := parts[1:]
	switch head {
	case "q", "qa", "quit":
		return m, tea.Quit
	case "sync":
		m.syncing = true
		return m, m.deps.Sync.FullSync(context.Background())
	case "refresh":
		return m.dispatchRefresh()
	case "view":
		return m.cmdSwitchView(args), nil
	case "open":
		return m.cmdOpenURL()
	case "state":
		return m.cmdChangeState(strings.Join(args, " "))
	case "assign":
		return m.cmdChangeAssignee(strings.Join(args, " "))
	case "priority":
		return m.cmdChangePriority(strings.Join(args, " "))
	}
	m.errorBanner = "unknown command: " + head
	return m, clearBannerAfter(3 * time.Second)
}

func (m Model) cmdSwitchView(args []string) Model {
	if len(args) == 0 {
		m.errorBanner = "usage: :view my_issues|triage"
		return m
	}
	switch args[0] {
	case "my_issues":
		m.persistCursor()
		m.view = ViewMyIssues
		m.restoreCursor()
	case "triage":
		m.persistCursor()
		m.view = ViewTriage
		m.restoreCursor()
	default:
		m.errorBanner = "unknown view: " + args[0]
	}
	return m
}

func (m Model) cmdOpenURL() (tea.Model, tea.Cmd) {
	iss := m.selectedIssue()
	if iss == nil || iss.URL == "" {
		m.errorBanner = "no URL"
		return m, clearBannerAfter(3 * time.Second)
	}
	opener := "xdg-open"
	if runtime.GOOS == "darwin" {
		opener = "open"
	}
	_ = exec.Command(opener, iss.URL).Start()
	return m, nil
}

func (m Model) cmdChangeState(name string) (tea.Model, tea.Cmd) {
	iss := m.selectedIssue()
	if iss == nil || name == "" {
		m.errorBanner = "usage: :state <name>"
		return m, clearBannerAfter(3 * time.Second)
	}
	ws, _ := m.deps.Repos.States.FindByTeamAndName(iss.TeamID, name)
	if ws == nil {
		m.errorBanner = "no state: " + name
		return m, clearBannerAfter(3 * time.Second)
	}
	m.mutatingID = iss.ID
	m.syncing = true
	return m, m.deps.Sync.UpdateIssue(context.Background(), iss.ID, map[string]any{"stateId": ws.ID})
}

func (m Model) cmdChangeAssignee(spec string) (tea.Model, tea.Cmd) {
	iss := m.selectedIssue()
	if iss == nil || spec == "" {
		m.errorBanner = "usage: :assign <name|me|none>"
		return m, clearBannerAfter(3 * time.Second)
	}
	var input map[string]any
	switch strings.ToLower(spec) {
	case "none", "unassigned", "nobody":
		input = map[string]any{"assigneeId": nil}
	case "me":
		if m.viewerID == "" {
			m.errorBanner = "viewer not loaded yet"
			return m, clearBannerAfter(3 * time.Second)
		}
		input = map[string]any{"assigneeId": m.viewerID}
	default:
		u, _ := m.deps.Repos.Users.FindByName(spec)
		if u == nil {
			m.errorBanner = "no user: " + spec
			return m, clearBannerAfter(3 * time.Second)
		}
		input = map[string]any{"assigneeId": u.ID}
	}
	m.mutatingID = iss.ID
	m.syncing = true
	return m, m.deps.Sync.UpdateIssue(context.Background(), iss.ID, input)
}

func (m Model) cmdChangePriority(spec string) (tea.Model, tea.Cmd) {
	iss := m.selectedIssue()
	if iss == nil || spec == "" {
		m.errorBanner = "usage: :priority <n|name>"
		return m, clearBannerAfter(3 * time.Second)
	}
	var pri int
	switch strings.ToLower(spec) {
	case "urgent":
		pri = cache.PriorityUrgent
	case "high":
		pri = cache.PriorityHigh
	case "normal":
		pri = cache.PriorityNormal
	case "low":
		pri = cache.PriorityLow
	case "none":
		pri = cache.PriorityNone
	default:
		n, err := strconv.Atoi(spec)
		if err != nil || n < 0 || n > 4 {
			m.errorBanner = "bad priority: " + spec
			return m, clearBannerAfter(3 * time.Second)
		}
		pri = n
	}
	m.mutatingID = iss.ID
	m.syncing = true
	return m, m.deps.Sync.UpdateIssue(context.Background(), iss.ID, map[string]any{"priority": pri})
}
