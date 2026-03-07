package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"gh-dashboard/internal/config"
	"gh-dashboard/internal/github"
	"gh-dashboard/internal/state"
	"gh-dashboard/internal/ui"
)

func main() {
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
