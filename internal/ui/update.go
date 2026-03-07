package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"gh-dashboard/internal/config"
	"gh-dashboard/internal/github"
	"gh-dashboard/internal/logger"
)

// ── Messages ─────────────────────────────────────────────────────────────────

type prsLoadedMsg struct {
	prs []github.PR
}

type errMsg struct {
	err error
}

// ── Commands ─────────────────────────────────────────────────────────────────

func fetchPRsCmd(cfg *config.Config, client *github.Client) tea.Cmd {
	return func() tea.Msg {
		var orgs []github.OrgQuery
		for _, o := range cfg.Orgs {
			orgs = append(orgs, github.OrgQuery{
				Name:  o.Name,
				Repos: o.Repos,
			})
		}
		prs, err := client.FetchReviewerPRs(orgs)
		if err != nil {
			return errMsg{err: err}
		}
		return prsLoadedMsg{prs: prs}
	}
}

func openBrowserCmd(url string) tea.Cmd {
	return func() tea.Msg {
		_ = openBrowser(url)
		return nil
	}
}

// ── Update ───────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = m.detailWidth()
		m.viewport.Height = m.contentHeight()
		if len(m.filtered) > 0 || m.err != nil {
			m.updateDetail()
		}
		return m, nil

	case prsLoadedMsg:
		m.prs = msg.prs
		m.loading = false
		m.err = nil
		m.applyFilters()
		m.updateDetail()
		return m, nil

	case errMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		if m.filterMode {
			return m.handleFilterKey(msg)
		}
		return m.handleKey(msg)
	}

	// Pass other messages to the viewport (mouse scroll etc.)
	if !m.filterMode {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {

	case "ctrl+c", "q":
		return m, tea.Quit

	case "up", "k":
		if m.selected > 0 {
			m.selected--
			m.adjustListScroll()
			m.updateDetail()
		}

	case "down", "j":
		if m.selected < len(m.filtered)-1 {
			m.selected++
			m.adjustListScroll()
			m.updateDetail()
		}

	case "/":
		m.filterMode = true
		cmd := m.filterInput.Focus()
		return m, cmd

	case "s":
		m.sortAsc = !m.sortAsc
		m.applyFilters()
		m.updateDetail()

	case "r":
		m.reviewFilter = (m.reviewFilter + 1) % 4
		m.applyFilters()
		m.updateDetail()

	case "o":
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			pr := m.filtered[m.selected]
			logger.L.Info("opening PR in browser", "number", pr.Number, "url", pr.URL, "repo", pr.Repo)
			return m, openBrowserCmd(pr.URL)
		}

	case "m":
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			pr := m.filtered[m.selected]
			m.state.ToggleRead(pr.ID)
			_ = m.state.Save()
		}

	case "f5":
		m.loading = true
		m.err = nil
		return m, fetchPRsCmd(m.config, m.client)

	case "pgup", "pgdown", "ctrl+u", "ctrl+d":
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {

	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.filterMode = false
		m.filterInput.Blur()
		m.filterInput.SetValue("")
		m.applyFilters()
		m.updateDetail()
		return m, nil

	case "enter":
		m.filterMode = false
		m.filterInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	// Re-filter on every keystroke, reset selection to top
	m.selected = 0
	m.listOff = 0
	m.applyFilters()
	m.updateDetail()
	return m, cmd
}
