package state

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// Snapshot is the serializable slice of Model state we preserve across runs.
type Snapshot struct {
	View             string         `json:"view"`
	CursorIndex      int            `json:"cursor_index"`
	SelectedTeamID   string         `json:"selected_team_id"`
	LastSyncedAtUTC  string         `json:"last_synced_at,omitempty"`
	CursorPerTeam    map[string]int `json:"cursor_per_team,omitempty"`
}

// Load reads a snapshot from path. A missing file returns a zero Snapshot with
// no error, so first launches don't surface a spurious error.
func Load(path string) (Snapshot, error) {
	var s Snapshot
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return s, nil
		}
		return s, err
	}
	if len(raw) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, err
	}
	return s, nil
}

// Save writes the snapshot atomically (tmpfile + rename) with 0600 perms.
func Save(path string, s Snapshot) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
