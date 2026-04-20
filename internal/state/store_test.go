package state

import (
	"path/filepath"
	"testing"
)

func TestLoadMissingReturnsZero(t *testing.T) {
	s, err := Load("/nonexistent/path/does/not/exist.json")
	if err != nil {
		t.Fatal(err)
	}
	if s.View != "" || s.CursorIndex != 0 {
		t.Fatalf("want zero snapshot, got %+v", s)
	}
}

func TestRoundtrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.json")
	want := Snapshot{View: "triage", CursorIndex: 7, SelectedTeamID: "t1", CursorPerTeam: map[string]int{"t1": 7}}
	if err := Save(p, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.View != "triage" || got.CursorIndex != 7 || got.SelectedTeamID != "t1" {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if got.CursorPerTeam["t1"] != 7 {
		t.Fatalf("cursor_per_team = %+v", got.CursorPerTeam)
	}
}
