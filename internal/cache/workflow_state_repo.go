package cache

import (
	"database/sql"
	"time"
)

type WorkflowStateRepo struct{ db *sql.DB }

func (r *WorkflowStateRepo) Upsert(s WorkflowState) error {
	_, err := r.db.Exec(`
		INSERT INTO workflow_states (id, team_id, name, type, color, synced_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			team_id=excluded.team_id,
			name=excluded.name,
			type=excluded.type,
			color=excluded.color,
			synced_at=excluded.synced_at;`,
		s.ID, s.TeamID, s.Name, s.Type, s.Color,
		s.SyncedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (r *WorkflowStateRepo) ByTeam(teamID string) ([]WorkflowState, error) {
	rows, err := r.db.Query(`SELECT id, team_id, name, type, COALESCE(color,''), synced_at FROM workflow_states WHERE team_id=? ORDER BY type, name`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WorkflowState
	for rows.Next() {
		var s WorkflowState
		var synced string
		if err := rows.Scan(&s.ID, &s.TeamID, &s.Name, &s.Type, &s.Color, &synced); err != nil {
			return nil, err
		}
		s.SyncedAt, _ = time.Parse(time.RFC3339Nano, synced)
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *WorkflowStateRepo) FindByID(id string) (*WorkflowState, error) {
	row := r.db.QueryRow(`SELECT id, team_id, name, type, COALESCE(color,''), synced_at FROM workflow_states WHERE id=?`, id)
	var s WorkflowState
	var synced string
	err := row.Scan(&s.ID, &s.TeamID, &s.Name, &s.Type, &s.Color, &synced)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.SyncedAt, _ = time.Parse(time.RFC3339Nano, synced)
	return &s, nil
}

func (r *WorkflowStateRepo) FindByTeamAndName(teamID, name string) (*WorkflowState, error) {
	row := r.db.QueryRow(`SELECT id, team_id, name, type, COALESCE(color,''), synced_at FROM workflow_states WHERE team_id=? AND lower(name)=lower(?) LIMIT 1`, teamID, name)
	var s WorkflowState
	var synced string
	err := row.Scan(&s.ID, &s.TeamID, &s.Name, &s.Type, &s.Color, &synced)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.SyncedAt, _ = time.Parse(time.RFC3339Nano, synced)
	return &s, nil
}
