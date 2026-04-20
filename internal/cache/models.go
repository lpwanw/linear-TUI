package cache

import "time"

type Team struct {
	ID          string
	Key         string
	Name        string
	Description string
	SyncedAt    time.Time
}

type User struct {
	ID       string
	Name     string
	Email    string
	IsMe     bool
	Active   bool
	SyncedAt time.Time
}

type WorkflowState struct {
	ID       string
	TeamID   string
	Name     string
	Type     string // triage, backlog, unstarted, started, completed, canceled
	Color    string
	SyncedAt time.Time
}

type Issue struct {
	ID          string
	Identifier  string
	Title       string
	Description string
	URL         string
	StateID     string
	Priority    int
	AssigneeID  string // "" if unassigned
	TeamID      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ArchivedAt  *time.Time
	SyncedAt    time.Time
}

const (
	StateTypeTriage    = "triage"
	StateTypeBacklog   = "backlog"
	StateTypeUnstarted = "unstarted"
	StateTypeStarted   = "started"
	StateTypeCompleted = "completed"
	StateTypeCanceled  = "canceled"
)

const (
	PriorityNone   = 0
	PriorityUrgent = 1
	PriorityHigh   = 2
	PriorityNormal = 3
	PriorityLow    = 4
)

func PriorityLabel(p int) string {
	switch p {
	case PriorityUrgent:
		return "urgent"
	case PriorityHigh:
		return "high"
	case PriorityNormal:
		return "normal"
	case PriorityLow:
		return "low"
	default:
		return "none"
	}
}
