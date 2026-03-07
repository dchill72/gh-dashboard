package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"gh-dashboard/internal/config"
	"gh-dashboard/internal/github"
	"gh-dashboard/internal/logger"
	"gh-dashboard/internal/state"
	"gh-dashboard/internal/ui"
)

func main() {
	if err := logger.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to init logger: %v\n", err)
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "error: GITHUB_TOKEN environment variable is required")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.Orgs) == 0 {
		path, _ := config.ConfigPath()
		fmt.Fprintf(os.Stderr, "error: no [[orgs]] configured in %s\n", path)
		os.Exit(1)
	}

	logger.L.Info("config loaded", "host", cfg.GitHub.Host, "orgs", len(cfg.Orgs))
	for _, o := range cfg.Orgs {
		logger.L.Info("org configured", "name", o.Name, "repos", o.Repos)
	}

	st, err := state.Load()
	if err != nil {
		// Non-fatal — start with fresh state
		st = state.New()
	}

	client := github.NewClient(cfg.GitHub.Host, token)
	m := ui.NewModel(cfg, client, st)

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
