package cache

import (
	"testing"
	"time"
)

func openTestDB(t *testing.T) *Repos {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return NewRepos(db)
}

func TestMigrateIdempotent(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for i := 0; i < 3; i++ {
		if err := Migrate(db); err != nil {
			t.Fatalf("migrate #%d: %v", i, err)
		}
	}
}

func TestTeamRoundtrip(t *testing.T) {
	repos := openTestDB(t)
	now := time.Now()
	team := Team{ID: "t1", Key: "ENG", Name: "Engineering", SyncedAt: now}
	if err := repos.Teams.Upsert(team); err != nil {
		t.Fatal(err)
	}
	got, err := repos.Teams.FindByKey("eng")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ID != "t1" {
		t.Fatalf("FindByKey eng = %+v, want id t1", got)
	}
}

func TestIssueQueries(t *testing.T) {
	repos := openTestDB(t)
	now := time.Now()
	_ = repos.Teams.Upsert(Team{ID: "t1", Key: "ENG", Name: "Engineering", SyncedAt: now})
	_ = repos.Users.Upsert(User{ID: "u1", Name: "Me", IsMe: true, Active: true, SyncedAt: now})
	_ = repos.States.Upsert(WorkflowState{ID: "s1", TeamID: "t1", Name: "Todo", Type: StateTypeUnstarted, SyncedAt: now})
	_ = repos.States.Upsert(WorkflowState{ID: "s2", TeamID: "t1", Name: "Done", Type: StateTypeCompleted, SyncedAt: now})
	_ = repos.States.Upsert(WorkflowState{ID: "s3", TeamID: "t1", Name: "Triage", Type: StateTypeTriage, SyncedAt: now})

	_ = repos.Issues.Upsert(Issue{ID: "i1", Identifier: "ENG-1", Title: "Alpha", StateID: "s1", AssigneeID: "u1", TeamID: "t1", UpdatedAt: now, SyncedAt: now})
	_ = repos.Issues.Upsert(Issue{ID: "i2", Identifier: "ENG-2", Title: "Beta done", StateID: "s2", AssigneeID: "u1", TeamID: "t1", UpdatedAt: now, SyncedAt: now})
	_ = repos.Issues.Upsert(Issue{ID: "i3", Identifier: "ENG-3", Title: "Unassigned", StateID: "s3", AssigneeID: "", TeamID: "t1", UpdatedAt: now, SyncedAt: now})

	mine, err := repos.Issues.AssignedTo("u1", []string{StateTypeCompleted, StateTypeCanceled})
	if err != nil {
		t.Fatal(err)
	}
	if len(mine) != 1 || mine[0].ID != "i1" {
		t.Fatalf("mine = %+v, want [i1]", mine)
	}

	triage, err := repos.Issues.UnassignedInTeam("t1", []string{StateTypeTriage, StateTypeBacklog})
	if err != nil {
		t.Fatal(err)
	}
	if len(triage) != 1 || triage[0].ID != "i3" {
		t.Fatalf("triage = %+v, want [i3]", triage)
	}
}

func TestUpsertOverwrites(t *testing.T) {
	repos := openTestDB(t)
	now := time.Now()
	_ = repos.Teams.Upsert(Team{ID: "t1", Key: "ENG", Name: "Engineering", SyncedAt: now})
	_ = repos.Teams.Upsert(Team{ID: "t1", Key: "ENG", Name: "Engineering Renamed", SyncedAt: now})
	got, _ := repos.Teams.FindByID("t1")
	if got.Name != "Engineering Renamed" {
		t.Fatalf("name = %q, want Engineering Renamed", got.Name)
	}
}
