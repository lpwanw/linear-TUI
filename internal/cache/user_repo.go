package cache

import (
	"database/sql"
	"time"
)

type UserRepo struct{ db *sql.DB }

func (r *UserRepo) Upsert(u User) error {
	_, err := r.db.Exec(`
		INSERT INTO users (id, name, email, is_me, active, synced_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name,
			email=excluded.email,
			is_me=excluded.is_me,
			active=excluded.active,
			synced_at=excluded.synced_at;`,
		u.ID, u.Name, u.Email, boolToInt(u.IsMe), boolToInt(u.Active),
		u.SyncedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (r *UserRepo) All() ([]User, error) {
	rows, err := r.db.Query(`SELECT id, name, COALESCE(email,''), is_me, active, synced_at FROM users WHERE active=1 ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		var isMe, active int
		var synced string
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &isMe, &active, &synced); err != nil {
			return nil, err
		}
		u.IsMe = isMe == 1
		u.Active = active == 1
		u.SyncedAt, _ = time.Parse(time.RFC3339Nano, synced)
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *UserRepo) Me() (*User, error) {
	row := r.db.QueryRow(`SELECT id, name, COALESCE(email,''), is_me, active, synced_at FROM users WHERE is_me=1 LIMIT 1`)
	var u User
	var isMe, active int
	var synced string
	err := row.Scan(&u.ID, &u.Name, &u.Email, &isMe, &active, &synced)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.IsMe = isMe == 1
	u.Active = active == 1
	u.SyncedAt, _ = time.Parse(time.RFC3339Nano, synced)
	return &u, nil
}

func (r *UserRepo) FindByName(name string) (*User, error) {
	row := r.db.QueryRow(`SELECT id, name, COALESCE(email,''), is_me, active, synced_at FROM users WHERE lower(name)=lower(?) LIMIT 1`, name)
	var u User
	var isMe, active int
	var synced string
	err := row.Scan(&u.ID, &u.Name, &u.Email, &isMe, &active, &synced)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.IsMe = isMe == 1
	u.Active = active == 1
	u.SyncedAt, _ = time.Parse(time.RFC3339Nano, synced)
	return &u, nil
}
