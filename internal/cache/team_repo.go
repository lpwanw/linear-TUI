package cache

import (
	"database/sql"
	"time"
)

type TeamRepo struct{ db *sql.DB }

func (r *TeamRepo) Upsert(t Team) error {
	_, err := r.db.Exec(`
		INSERT INTO teams (id, key, name, description, synced_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			key=excluded.key,
			name=excluded.name,
			description=excluded.description,
			synced_at=excluded.synced_at;`,
		t.ID, t.Key, t.Name, t.Description, t.SyncedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (r *TeamRepo) All() ([]Team, error) {
	rows, err := r.db.Query(`SELECT id, key, name, COALESCE(description, ''), synced_at FROM teams ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Team
	for rows.Next() {
		var t Team
		var synced string
		if err := rows.Scan(&t.ID, &t.Key, &t.Name, &t.Description, &synced); err != nil {
			return nil, err
		}
		t.SyncedAt, _ = time.Parse(time.RFC3339Nano, synced)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TeamRepo) FindByKey(key string) (*Team, error) {
	row := r.db.QueryRow(`SELECT id, key, name, COALESCE(description, ''), synced_at FROM teams WHERE key=? COLLATE NOCASE`, key)
	var t Team
	var synced string
	err := row.Scan(&t.ID, &t.Key, &t.Name, &t.Description, &synced)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.SyncedAt, _ = time.Parse(time.RFC3339Nano, synced)
	return &t, nil
}

func (r *TeamRepo) FindByID(id string) (*Team, error) {
	row := r.db.QueryRow(`SELECT id, key, name, COALESCE(description, ''), synced_at FROM teams WHERE id=?`, id)
	var t Team
	var synced string
	err := row.Scan(&t.ID, &t.Key, &t.Name, &t.Description, &synced)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.SyncedAt, _ = time.Parse(time.RFC3339Nano, synced)
	return &t, nil
}
