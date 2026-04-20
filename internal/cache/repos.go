package cache

import "database/sql"

type Repos struct {
	Issues *IssueRepo
	Teams  *TeamRepo
	Users  *UserRepo
	States *WorkflowStateRepo
}

func NewRepos(db *sql.DB) *Repos {
	return &Repos{
		Issues: &IssueRepo{db: db},
		Teams:  &TeamRepo{db: db},
		Users:  &UserRepo{db: db},
		States: &WorkflowStateRepo{db: db},
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func fromNullStr(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
