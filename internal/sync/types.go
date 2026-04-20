package sync

import "time"

// GraphQL decode types. These mirror the shapes in api/queries.go.

type viewerPayload struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type teamPayload struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Key         string `json:"key"`
	Description string `json:"description"`
}

type userPayload struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	IsMe   bool   `json:"isMe"`
	Active bool   `json:"active"`
}

type workflowStatePayload struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Color string `json:"color"`
}

type issuePayload struct {
	ID          string     `json:"id"`
	Identifier  string     `json:"identifier"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	URL         string     `json:"url"`
	Priority    int        `json:"priority"`
	State       *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"state"`
	Assignee *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"assignee"`
	Team *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Key  string `json:"key"`
	} `json:"team"`
	CreatedAt  *time.Time `json:"createdAt"`
	UpdatedAt  *time.Time `json:"updatedAt"`
	ArchivedAt *time.Time `json:"archivedAt"`
}

type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}
