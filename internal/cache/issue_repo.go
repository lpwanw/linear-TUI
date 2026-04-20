package cache

import (
	"database/sql"
	"strings"
	"time"
)

type IssueRepo struct{ db *sql.DB }

const issueCols = `i.id, i.identifier, i.title, COALESCE(i.description,''), COALESCE(i.url,''), COALESCE(i.state_id,''), i.priority, COALESCE(i.assignee_id,''), i.team_id, COALESCE(i.created_at,''), COALESCE(i.updated_at,''), i.archived_at, i.synced_at`

func (r *IssueRepo) Upsert(i Issue) error {
	var archived any
	if i.ArchivedAt != nil {
		archived = i.ArchivedAt.UTC().Format(time.RFC3339Nano)
	}
	_, err := r.db.Exec(`
		INSERT INTO issues (id, identifier, title, description, url, state_id, priority, assignee_id, team_id, created_at, updated_at, archived_at, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			identifier=excluded.identifier,
			title=excluded.title,
			description=excluded.description,
			url=excluded.url,
			state_id=excluded.state_id,
			priority=excluded.priority,
			assignee_id=excluded.assignee_id,
			team_id=excluded.team_id,
			created_at=excluded.created_at,
			updated_at=excluded.updated_at,
			archived_at=excluded.archived_at,
			synced_at=excluded.synced_at;`,
		i.ID, i.Identifier, i.Title, i.Description, i.URL,
		nullStr(i.StateID), i.Priority, nullStr(i.AssigneeID), i.TeamID,
		fmtTime(i.CreatedAt), fmtTime(i.UpdatedAt), archived,
		i.SyncedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (r *IssueRepo) FindByID(id string) (*Issue, error) {
	row := r.db.QueryRow(`SELECT `+issueCols+` FROM issues i WHERE i.id=?`, id)
	return scanIssue(row.Scan)
}

func (r *IssueRepo) AssignedTo(userID string, excludedStateTypes []string) ([]Issue, error) {
	if userID == "" {
		return nil, nil
	}
	q := `SELECT ` + issueCols + ` FROM issues i`
	args := []any{userID}
	conds := []string{"i.assignee_id = ?"}
	if len(excludedStateTypes) > 0 {
		q += ` LEFT JOIN workflow_states ws ON ws.id = i.state_id`
		placeholders := strings.TrimRight(strings.Repeat("?,", len(excludedStateTypes)), ",")
		conds = append(conds, "(ws.type IS NULL OR ws.type NOT IN ("+placeholders+"))")
		for _, t := range excludedStateTypes {
			args = append(args, t)
		}
	}
	q += ` WHERE ` + strings.Join(conds, " AND ") + ` ORDER BY i.updated_at DESC`
	return queryIssues(r.db, q, args...)
}

func (r *IssueRepo) UnassignedInTeam(teamID string, includedStateTypes []string) ([]Issue, error) {
	q := `SELECT ` + issueCols + ` FROM issues i`
	args := []any{teamID}
	conds := []string{"i.team_id = ?", "i.assignee_id IS NULL"}
	if len(includedStateTypes) > 0 {
		q += ` LEFT JOIN workflow_states ws ON ws.id = i.state_id`
		placeholders := strings.TrimRight(strings.Repeat("?,", len(includedStateTypes)), ",")
		conds = append(conds, "ws.type IN ("+placeholders+")")
		for _, t := range includedStateTypes {
			args = append(args, t)
		}
	}
	q += ` WHERE ` + strings.Join(conds, " AND ") + ` ORDER BY i.updated_at DESC`
	return queryIssues(r.db, q, args...)
}

func fmtTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func scanIssue(scan func(...any) error) (*Issue, error) {
	var i Issue
	var archived sql.NullString
	var created, updated, synced string
	err := scan(&i.ID, &i.Identifier, &i.Title, &i.Description, &i.URL, &i.StateID, &i.Priority, &i.AssigneeID, &i.TeamID, &created, &updated, &archived, &synced)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if created != "" {
		i.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	}
	if updated != "" {
		i.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	}
	if synced != "" {
		i.SyncedAt, _ = time.Parse(time.RFC3339Nano, synced)
	}
	if archived.Valid && archived.String != "" {
		t, _ := time.Parse(time.RFC3339Nano, archived.String)
		i.ArchivedAt = &t
	}
	return &i, nil
}

func queryIssues(db *sql.DB, q string, args ...any) ([]Issue, error) {
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Issue
	for rows.Next() {
		i, err := scanIssue(rows.Scan)
		if err != nil {
			return nil, err
		}
		if i != nil {
			out = append(out, *i)
		}
	}
	return out, rows.Err()
}
