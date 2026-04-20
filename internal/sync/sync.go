package sync

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/taynguyen/linear-tui/internal/api"
	"github.com/taynguyen/linear-tui/internal/cache"
)

// Service orchestrates pulls from the Linear API into the SQLite cache.
type Service struct {
	client *api.Client
	repos  *cache.Repos
}

func New(client *api.Client, repos *cache.Repos) *Service {
	return &Service{client: client, repos: repos}
}

// BootstrapResultMsg carries everything fetched during initial sync. The
// model merges the whole payload in a single Update step, which keeps the
// Bubble Tea dispatch trivial (no multi-msg emission from one Cmd).
type BootstrapResultMsg struct {
	ViewerID     string
	Teams        []cache.Team
	Users        []cache.User
	States       map[string][]cache.WorkflowState // teamID -> states
	MyIssues     []cache.Issue
	TeamTriage   map[string][]cache.Issue // teamID -> issues
}

// Bootstrap runs the full initial sync and returns a single BootstrapResultMsg.
// On any failure, returns SyncErrorMsg.
func (s *Service) Bootstrap(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		viewer, teams, err := s.fetchViewerAndTeams(ctx)
		if err != nil {
			return SyncErrorMsg{Err: err}
		}
		users, err := s.fetchUsers(ctx)
		if err != nil {
			return SyncErrorMsg{Err: err}
		}
		statesByTeam := map[string][]cache.WorkflowState{}
		for _, t := range teams {
			states, err := s.fetchWorkflowStates(ctx, t.ID)
			if err != nil {
				return SyncErrorMsg{Err: err}
			}
			statesByTeam[t.ID] = states
		}
		mine, err := s.fetchMyIssues(ctx)
		if err != nil {
			return SyncErrorMsg{Err: err}
		}
		triage := map[string][]cache.Issue{}
		for _, t := range teams {
			list, err := s.fetchTriage(ctx, t.ID)
			if err != nil {
				return SyncErrorMsg{Err: err}
			}
			triage[t.ID] = list
		}
		return BootstrapResultMsg{
			ViewerID:   viewer.ID,
			Teams:      teams,
			Users:      users,
			States:     statesByTeam,
			MyIssues:   mine,
			TeamTriage: triage,
		}
	}
}

// RefreshMyIssues re-fetches the My Issues query and upserts cache.
func (s *Service) RefreshMyIssues(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		issues, err := s.fetchMyIssues(ctx)
		if err != nil {
			return RefreshDoneMsg{View: "my_issues", Err: err}
		}
		return MyIssuesLoadedMsg{Issues: issues}
	}
}

// RefreshTriage re-fetches triage for a team.
func (s *Service) RefreshTriage(ctx context.Context, teamID string) tea.Cmd {
	return func() tea.Msg {
		issues, err := s.fetchTriage(ctx, teamID)
		if err != nil {
			return RefreshDoneMsg{View: "triage", Err: err}
		}
		return TriageLoadedMsg{TeamID: teamID, Issues: issues}
	}
}

// FullSync = Bootstrap, but model keeps existing state until DoneMsg fires.
func (s *Service) FullSync(ctx context.Context) tea.Cmd { return s.Bootstrap(ctx) }

// UpdateIssue dispatches an `issueUpdate` mutation with the supplied input.
func (s *Service) UpdateIssue(ctx context.Context, issueID string, input map[string]any) tea.Cmd {
	return func() tea.Msg {
		issue, err := s.runIssueUpdate(ctx, issueID, input)
		if err != nil {
			return MutationDoneMsg{Err: err}
		}
		return MutationDoneMsg{Issue: issue}
	}
}

// -- fetchers --

func (s *Service) fetchViewerAndTeams(ctx context.Context) (*viewerPayload, []cache.Team, error) {
	var out struct {
		Viewer viewerPayload `json:"viewer"`
		Teams  struct {
			Nodes []teamPayload `json:"nodes"`
		} `json:"teams"`
	}
	if err := s.client.Query(ctx, api.QueryViewerAndTeams, nil, &out); err != nil {
		return nil, nil, err
	}
	now := time.Now()
	teams := make([]cache.Team, 0, len(out.Teams.Nodes))
	for _, t := range out.Teams.Nodes {
		ct := cache.Team{ID: t.ID, Key: t.Key, Name: t.Name, Description: t.Description, SyncedAt: now}
		if err := s.repos.Teams.Upsert(ct); err != nil {
			return nil, nil, err
		}
		teams = append(teams, ct)
	}
	// Mark viewer's cache user as IsMe (row upserted when we fetch users, so just upsert placeholder now)
	_ = s.repos.Users.Upsert(cache.User{ID: out.Viewer.ID, Name: out.Viewer.Name, Email: out.Viewer.Email, IsMe: true, Active: true, SyncedAt: now})
	return &out.Viewer, teams, nil
}

func (s *Service) fetchUsers(ctx context.Context) ([]cache.User, error) {
	const pageSize = 50
	var all []cache.User
	var cursor string
	now := time.Now()
	for {
		vars := map[string]any{"first": pageSize}
		if cursor != "" {
			vars["after"] = cursor
		}
		var out struct {
			Users struct {
				PageInfo pageInfo      `json:"pageInfo"`
				Nodes    []userPayload `json:"nodes"`
			} `json:"users"`
		}
		if err := s.client.Query(ctx, api.QueryWorkspaceUsers, vars, &out); err != nil {
			return nil, err
		}
		for _, u := range out.Users.Nodes {
			cu := cache.User{ID: u.ID, Name: u.Name, Email: u.Email, IsMe: u.IsMe, Active: u.Active, SyncedAt: now}
			if err := s.repos.Users.Upsert(cu); err != nil {
				return nil, err
			}
			all = append(all, cu)
		}
		if !out.Users.PageInfo.HasNextPage {
			break
		}
		cursor = out.Users.PageInfo.EndCursor
	}
	return all, nil
}

func (s *Service) fetchWorkflowStates(ctx context.Context, teamID string) ([]cache.WorkflowState, error) {
	var out struct {
		Team struct {
			ID     string `json:"id"`
			States struct {
				Nodes []workflowStatePayload `json:"nodes"`
			} `json:"states"`
		} `json:"team"`
	}
	if err := s.client.Query(ctx, api.QueryTeamWorkflowStates, map[string]any{"teamId": teamID}, &out); err != nil {
		return nil, err
	}
	now := time.Now()
	states := make([]cache.WorkflowState, 0, len(out.Team.States.Nodes))
	for _, ws := range out.Team.States.Nodes {
		cws := cache.WorkflowState{ID: ws.ID, TeamID: teamID, Name: ws.Name, Type: ws.Type, Color: ws.Color, SyncedAt: now}
		if err := s.repos.States.Upsert(cws); err != nil {
			return nil, err
		}
		states = append(states, cws)
	}
	return states, nil
}

func (s *Service) fetchMyIssues(ctx context.Context) ([]cache.Issue, error) {
	const pageSize = 25
	var all []cache.Issue
	var cursor string
	for {
		vars := map[string]any{"first": pageSize}
		if cursor != "" {
			vars["after"] = cursor
		}
		var out struct {
			Viewer struct {
				AssignedIssues struct {
					PageInfo pageInfo       `json:"pageInfo"`
					Nodes    []issuePayload `json:"nodes"`
				} `json:"assignedIssues"`
			} `json:"viewer"`
		}
		if err := s.client.Query(ctx, api.QueryMyIssues, vars, &out); err != nil {
			return nil, err
		}
		for _, p := range out.Viewer.AssignedIssues.Nodes {
			ci := issueFromPayload(p)
			if err := s.repos.Issues.Upsert(ci); err != nil {
				return nil, err
			}
			all = append(all, ci)
		}
		if !out.Viewer.AssignedIssues.PageInfo.HasNextPage {
			break
		}
		cursor = out.Viewer.AssignedIssues.PageInfo.EndCursor
	}
	return all, nil
}

func (s *Service) fetchTriage(ctx context.Context, teamID string) ([]cache.Issue, error) {
	const pageSize = 25
	var all []cache.Issue
	var cursor string
	for {
		vars := map[string]any{"teamId": teamID, "first": pageSize}
		if cursor != "" {
			vars["after"] = cursor
		}
		var out struct {
			Team struct {
				ID     string `json:"id"`
				Issues struct {
					PageInfo pageInfo       `json:"pageInfo"`
					Nodes    []issuePayload `json:"nodes"`
				} `json:"issues"`
			} `json:"team"`
		}
		if err := s.client.Query(ctx, api.QueryTeamTriage, vars, &out); err != nil {
			return nil, err
		}
		for _, p := range out.Team.Issues.Nodes {
			ci := issueFromPayload(p)
			if err := s.repos.Issues.Upsert(ci); err != nil {
				return nil, err
			}
			all = append(all, ci)
		}
		if !out.Team.Issues.PageInfo.HasNextPage {
			break
		}
		cursor = out.Team.Issues.PageInfo.EndCursor
	}
	return all, nil
}

func (s *Service) runIssueUpdate(ctx context.Context, id string, input map[string]any) (*cache.Issue, error) {
	var out struct {
		IssueUpdate struct {
			Success bool         `json:"success"`
			Issue   issuePayload `json:"issue"`
		} `json:"issueUpdate"`
	}
	if err := s.client.Query(ctx, api.MutationIssueUpdate, map[string]any{"id": id, "input": input}, &out); err != nil {
		return nil, err
	}
	ci := issueFromPayload(out.IssueUpdate.Issue)
	if err := s.repos.Issues.Upsert(ci); err != nil {
		return nil, err
	}
	return &ci, nil
}

func issueFromPayload(p issuePayload) cache.Issue {
	ci := cache.Issue{
		ID:          p.ID,
		Identifier:  p.Identifier,
		Title:       p.Title,
		Description: p.Description,
		URL:         p.URL,
		Priority:    p.Priority,
		SyncedAt:    time.Now(),
	}
	if p.State != nil {
		ci.StateID = p.State.ID
	}
	if p.Assignee != nil {
		ci.AssigneeID = p.Assignee.ID
	}
	if p.Team != nil {
		ci.TeamID = p.Team.ID
	}
	if p.CreatedAt != nil {
		ci.CreatedAt = *p.CreatedAt
	}
	if p.UpdatedAt != nil {
		ci.UpdatedAt = *p.UpdatedAt
	}
	if p.ArchivedAt != nil {
		t := *p.ArchivedAt
		ci.ArchivedAt = &t
	}
	return ci
}
