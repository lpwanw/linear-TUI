package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/taynguyen/linear-tui/internal/api"
	"github.com/taynguyen/linear-tui/internal/app"
	"github.com/taynguyen/linear-tui/internal/cache"
	"github.com/taynguyen/linear-tui/internal/config"
	"github.com/taynguyen/linear-tui/internal/state"
	"github.com/taynguyen/linear-tui/internal/sync"
)

const version = "0.0.1"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Println("linear-tui", version)
			return
		case "--help", "-h":
			printHelp()
			return
		}
	}

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	db, err := cache.Open(cfg.DBPath())
	if err != nil {
		return fmt.Errorf("open cache: %w", err)
	}
	defer db.Close()

	if err := cache.Migrate(db); err != nil {
		return fmt.Errorf("migrate cache: %w", err)
	}

	repos := cache.NewRepos(db)
	client := api.NewClient(cfg.APIKey, nil)
	syncSvc := sync.New(client, repos)

	restored, _ := state.Load(cfg.StatePath())

	model := app.New(app.Deps{
		Cfg:      cfg,
		Client:   client,
		Repos:    repos,
		Sync:     syncSvc,
		Restored: restored,
	})

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	final, err := p.Run()
	if err != nil {
		return err
	}

	if m, ok := final.(app.Model); ok {
		_ = state.Save(cfg.StatePath(), m.Snapshot())
	}
	return nil
}

func printHelp() {
	fmt.Print(`linear-tui — vim-style terminal UI for Linear

Usage:
  lnr              launch TUI
  lnr --version    print version
  lnr --help       print this help

Environment:
  LINEAR_API_KEY   required; Linear personal API key (lin_api_...)
  NO_COLOR         disable color output
  XDG_DATA_HOME    override data dir (cache.db)
  XDG_STATE_HOME   override state dir (state.json)
  XDG_CONFIG_HOME  override config dir
`)
}
