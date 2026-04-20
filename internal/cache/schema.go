package cache

import (
	"database/sql"
	"fmt"
)

var baseSchema = []string{
	`CREATE TABLE IF NOT EXISTS teams (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		synced_at TEXT NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT,
		is_me INTEGER NOT NULL DEFAULT 0,
		active INTEGER NOT NULL DEFAULT 1,
		synced_at TEXT NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS workflow_states (
		id TEXT PRIMARY KEY,
		team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		color TEXT,
		synced_at TEXT NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS issues (
		id TEXT PRIMARY KEY,
		identifier TEXT NOT NULL,
		title TEXT NOT NULL,
		description TEXT,
		url TEXT,
		state_id TEXT REFERENCES workflow_states(id),
		priority INTEGER NOT NULL DEFAULT 0,
		assignee_id TEXT REFERENCES users(id),
		team_id TEXT NOT NULL REFERENCES teams(id),
		created_at TEXT,
		updated_at TEXT,
		archived_at TEXT,
		synced_at TEXT NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_issues_assignee_state ON issues(assignee_id, state_id);`,
	`CREATE INDEX IF NOT EXISTS idx_issues_team_state ON issues(team_id, state_id);`,
	`CREATE INDEX IF NOT EXISTS idx_issues_updated_at ON issues(updated_at DESC);`,
	`CREATE INDEX IF NOT EXISTS idx_workflow_states_team ON workflow_states(team_id);`,
}

func Migrate(db *sql.DB) error {
	for _, stmt := range baseSchema {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w\nstmt: %s", err, stmt)
		}
	}
	return ensureColumns(db)
}

func ensureColumns(db *sql.DB) error {
	// Idempotent ALTER-TABLE pattern for future-proofing. Add entries when schema evolves.
	type addCol struct {
		table string
		col   string
		decl  string
	}
	future := []addCol{
		// { table: "issues", col: "cycle_id", decl: "TEXT" }, // phase 3
	}
	for _, a := range future {
		exists, err := columnExists(db, a.table, a.col)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		stmt := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", a.table, a.col, a.decl)
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("add column %s.%s: %w", a.table, a.col, err)
		}
	}
	return nil
}

func columnExists(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s);", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, rows.Err()
		}
	}
	return false, rows.Err()
}
