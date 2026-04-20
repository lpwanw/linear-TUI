package config

import (
	"path/filepath"
	"testing"
)

func TestLoad_MissingKey(t *testing.T) {
	t.Setenv(envAPIKey, "")
	if _, err := Load(); err == nil {
		t.Fatal("want error when LINEAR_API_KEY unset")
	}
}

func TestLoad_HonorsXDG(t *testing.T) {
	t.Setenv(envAPIKey, "lin_api_abc")
	t.Setenv("XDG_DATA_HOME", "/tmp/xdgdata")
	t.Setenv("XDG_STATE_HOME", "/tmp/xdgstate")
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdgconfig")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	wantDB := filepath.Join("/tmp/xdgdata", appDir, cacheFileName)
	if cfg.DBPath() != wantDB {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath(), wantDB)
	}
	wantState := filepath.Join("/tmp/xdgstate", appDir, stateFileName)
	if cfg.StatePath() != wantState {
		t.Errorf("StatePath = %q, want %q", cfg.StatePath(), wantState)
	}
}

func TestValidAPIKeyPrefix(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"lin_api_abc", true},
		{"lin_api_", false},
		{"lin_apix", false},
		{"", false},
	}
	for _, c := range cases {
		if got := ValidAPIKeyPrefix(c.in); got != c.want {
			t.Errorf("ValidAPIKeyPrefix(%q)=%v, want %v", c.in, got, c.want)
		}
	}
}
