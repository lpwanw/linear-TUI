package app

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/taynguyen/linear-tui/internal/api"
	"github.com/taynguyen/linear-tui/internal/cache"
	"github.com/taynguyen/linear-tui/internal/config"
	"github.com/taynguyen/linear-tui/internal/modals"
	"github.com/taynguyen/linear-tui/internal/state"
	"github.com/taynguyen/linear-tui/internal/sync"
	"github.com/taynguyen/linear-tui/internal/views"
	"github.com/taynguyen/linear-tui/internal/vimmotion"
)

type ViewName string

const (
	ViewMyIssues ViewName = "my_issues"
	ViewTriage   ViewName = "triage"
)

// Deps groups the dependencies the model needs to run. Constructed in main.
type Deps struct {
	Cfg      *config.Config
	Client   *api.Client
	Repos    *cache.Repos
	Sync     *sync.Service
	Restored state.Snapshot
}

// Model is the Bubble Tea root. Copy-on-update (Elm style).
type Model struct {
	deps Deps

	view         ViewName
	cursor       int
	cursorByTeam map[string]int

	issuesMy     []cache.Issue
	issuesTriage map[string][]cache.Issue // teamID -> issues

	viewerID       string
	teams          []cache.Team
	users          []cache.User
	selectedTeamID string
	stateByID      map[string]cache.WorkflowState // all teams
	userByID       map[string]cache.User

	width, height int

	syncing      bool
	lastSyncedAt time.Time
	errorBanner  string
	infoBanner   string

	// Modes
	modal         *modals.Picker
	mutatingID    string
	cmdActive     bool
	cmdInput      textinput.Model
	searchActive  bool
	searchCommit  bool
	searchInput   textinput.Model
	searchQuery   string
	helpVisible   bool
	captureActive bool

	// Sub-components
	parser   *vimmotion.Parser
	spinner  spinner.Model
	detailVP viewport.Model
	detailR  *views.DetailRenderer
}

// New constructs a Model from Deps. Does not start I/O.
func New(deps Deps) Model {
	ci := textinput.New()
	ci.Prompt = ":"
	ci.CharLimit = 256
	ci.Placeholder = ""

	si := textinput.New()
	si.Prompt = "/"
	si.CharLimit = 256

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	view := ViewMyIssues
	if v := ViewName(deps.Restored.View); v == ViewMyIssues || v == ViewTriage {
		view = v
	}

	m := Model{
		deps:           deps,
		view:           view,
		cursor:         deps.Restored.CursorIndex,
		cursorByTeam:   deps.Restored.CursorPerTeam,
		selectedTeamID: deps.Restored.SelectedTeamID,
		issuesTriage:   map[string][]cache.Issue{},
		stateByID:      map[string]cache.WorkflowState{},
		userByID:       map[string]cache.User{},

		cmdInput:    ci,
		searchInput: si,
		spinner:     sp,
		parser:      vimmotion.NewParser(),
		detailVP:    viewport.New(0, 0),
		detailR:     views.NewDetailRenderer(60),
	}
	if m.cursorByTeam == nil {
		m.cursorByTeam = map[string]int{}
	}
	// Prime from cache if available (offline first paint).
	m.loadFromCache()
	return m
}

// Init returns the initial Cmd: spinner tick + bootstrap sync.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.deps.Sync.Bootstrap(context.Background()),
		enterSyncingCmd(),
	)
}

// Snapshot returns a serializable view of the model for state.json persistence.
func (m Model) Snapshot() state.Snapshot {
	cpt := map[string]int{}
	for k, v := range m.cursorByTeam {
		cpt[k] = v
	}
	return state.Snapshot{
		View:            string(m.view),
		CursorIndex:     m.cursor,
		SelectedTeamID:  m.selectedTeamID,
		LastSyncedAtUTC: m.lastSyncedAt.UTC().Format(time.RFC3339Nano),
		CursorPerTeam:   cpt,
	}
}

func enterSyncingCmd() tea.Cmd {
	return func() tea.Msg { return enterSyncingMsg{} }
}

type enterSyncingMsg struct{}

// loadFromCache pre-populates collections from the local SQLite so the first
// paint is instant (even before bootstrap returns).
func (m *Model) loadFromCache() {
	if teams, err := m.deps.Repos.Teams.All(); err == nil {
		m.teams = teams
		if m.selectedTeamID == "" && len(teams) > 0 {
			m.selectedTeamID = teams[0].ID
		}
		for _, t := range teams {
			if states, err := m.deps.Repos.States.ByTeam(t.ID); err == nil {
				for _, s := range states {
					m.stateByID[s.ID] = s
				}
			}
		}
	}
	if users, err := m.deps.Repos.Users.All(); err == nil {
		m.users = users
		for _, u := range users {
			m.userByID[u.ID] = u
			if u.IsMe {
				m.viewerID = u.ID
			}
		}
	}
	if me, err := m.deps.Repos.Users.Me(); err == nil && me != nil {
		m.viewerID = me.ID
	}
	if m.viewerID != "" {
		if mine, err := m.deps.Repos.Issues.AssignedTo(m.viewerID, []string{cache.StateTypeCompleted, cache.StateTypeCanceled}); err == nil {
			m.issuesMy = mine
		}
	}
	for _, t := range m.teams {
		if triage, err := m.deps.Repos.Issues.UnassignedInTeam(t.ID, []string{cache.StateTypeTriage, cache.StateTypeBacklog}); err == nil {
			m.issuesTriage[t.ID] = triage
		}
	}
}

// currentIssues returns the active slice for the current view + filter.
func (m *Model) currentIssues() []cache.Issue {
	var base []cache.Issue
	switch m.view {
	case ViewMyIssues:
		base = m.issuesMy
	case ViewTriage:
		base = m.issuesTriage[m.selectedTeamID]
	}
	if m.searchQuery == "" {
		return base
	}
	q := strings.ToLower(m.searchQuery)
	out := make([]cache.Issue, 0, len(base))
	for _, iss := range base {
		if strings.Contains(strings.ToLower(iss.Identifier), q) || strings.Contains(strings.ToLower(iss.Title), q) {
			out = append(out, iss)
		}
	}
	return out
}

func (m *Model) selectedIssue() *cache.Issue {
	iss := m.currentIssues()
	if len(iss) == 0 {
		return nil
	}
	c := m.cursor
	if c < 0 {
		c = 0
	}
	if c >= len(iss) {
		c = len(iss) - 1
	}
	return &iss[c]
}

func (m *Model) clampCursor() {
	n := len(m.currentIssues())
	if n == 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= n {
		m.cursor = n - 1
	}
}
