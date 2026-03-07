package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"gh-dashboard/internal/github"
)

// ── Top-level View ───────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	// Full-screen loading / error states (only before first data arrives)
	if m.loading && len(m.prs) == 0 {
		return lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Loading PRs from GitHub…")
	}
	if m.err != nil && len(m.prs) == 0 {
		msg := fmt.Sprintf("Error: %v\n\nPress R to retry or q to quit.", m.err)
		return lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Render(msg)
	}

	header := m.renderHeader()
	body := m.renderBody()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

// ── Header ───────────────────────────────────────────────────────────────────

func (m Model) renderHeader() string {
	left := headerTitleStyle.Render("GH PR Dashboard")

	sortLabel := "↓ Newest"
	if m.sortAsc {
		sortLabel = "↑ Oldest"
	}

	badges := []string{
		dimStyle.Render("review:") + reviewFilterStyle.Render(m.reviewFilter.String()),
		dimStyle.Render("sort:") + metaStyle.Render(sortLabel),
	}
	if m.loading {
		badges = append([]string{loadingStyle.Render("refreshing…")}, badges...)
	}
	right := strings.Join(badges, "  ")

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	titleRow := headerStyle.Width(m.width).Render(
		left + strings.Repeat(" ", gap) + right,
	)
	sep := dimStyle.Render(strings.Repeat("─", m.width))
	return lipgloss.JoinVertical(lipgloss.Left, titleRow, sep)
}

// ── Body (split pane) ────────────────────────────────────────────────────────

func (m Model) renderBody() string {
	h := m.contentHeight()
	lw := m.listWidth()
	dw := m.detailWidth()

	list := m.renderList(lw, h)
	sep := renderVertSep(h)
	detail := lipgloss.NewStyle().Width(dw).Height(h).Render(m.viewport.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, list, sep, detail)
}

func renderVertSep(h int) string {
	lines := make([]string, h)
	for i := range lines {
		lines[i] = dimStyle.Render("│")
	}
	return strings.Join(lines, "\n")
}

// ── List pane ────────────────────────────────────────────────────────────────

func (m Model) renderList(w, h int) string {
	var lines []string
	listH := h

	// Filter input row (when active)
	if m.filterMode {
		fi := lipgloss.NewStyle().Width(w).Render("/ " + m.filterInput.View())
		lines = append(lines, fi)
		listH--
	}

	if len(m.filtered) == 0 {
		empty := dimStyle.Render("  no PRs match")
		lines = append(lines, lipgloss.NewStyle().Width(w).Render(empty))
		for len(lines) < h {
			lines = append(lines, strings.Repeat(" ", w))
		}
		return strings.Join(lines, "\n")
	}

	for i := m.listOff; i < m.listOff+listH && i < len(m.filtered); i++ {
		pr := m.filtered[i]
		lines = append(lines, m.renderListItem(pr, i == m.selected, w))
	}

	// Pad remaining rows
	for len(lines) < h {
		lines = append(lines, strings.Repeat(" ", w))
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderListItem(pr github.PR, selected bool, w int) string {
	date := pr.CreatedAt.Format("06-01-02") // YY-MM-DD (8 chars)

	// Use just the repository name (drop the org prefix)
	repo := pr.Repo
	if idx := strings.Index(repo, "/"); idx >= 0 {
		repo = repo[idx+1:]
	}
	const maxRepo = 16
	repo = runesTruncate(repo, maxRepo)
	repo = repo + strings.Repeat(" ", maxRepo-utf8.RuneCountInString(repo))

	const bulletW = 2 // "● "
	const dateW = 8   // YY-MM-DD
	const padding = 2 // spaces between sections
	titleW := w - bulletW - maxRepo - dateW - padding*2
	if titleW < 4 {
		titleW = 4
	}

	title := runesTruncate(pr.Title, titleW)
	title = title + strings.Repeat(" ", titleW-utf8.RuneCountInString(title))

	plain := fmt.Sprintf("● %s  %s  %s", title, repo, date)

	isRead := m.state.IsRead(pr.ID)

	if selected {
		if isRead {
			return selectedReadStyle.Width(w).Render(plain)
		}
		return selectedUnreadStyle.Width(w).Render(plain)
	}
	if isRead {
		return readStyle.Width(w).Render(plain)
	}
	return unreadStyle.Width(w).Render(plain)
}

// ── Detail pane content ──────────────────────────────────────────────────────

func renderDetail(pr github.PR, w int) string {
	var sb strings.Builder

	// Title
	title := fmt.Sprintf("#%d  %s", pr.Number, pr.Title)
	sb.WriteString(detailTitleStyle.Render(runesTruncateWide(title, w-2)))
	sb.WriteString("\n")

	// Meta line
	var badges []string
	if pr.IsDraft {
		badges = append(badges, draftStyle.Render("[DRAFT]"))
	}
	switch pr.ReviewDecision {
	case "APPROVED":
		badges = append(badges, approvedStyle.Render("✓ Approved"))
	case "CHANGES_REQUESTED":
		badges = append(badges, changesStyle.Render("✗ Changes Requested"))
	case "REVIEW_REQUIRED":
		badges = append(badges, reviewRequiredStyle.Render("⟳ Review Required"))
	}
	for _, role := range pr.Roles {
		switch role {
		case "reviewer":
			badges = append(badges, dimStyle.Render("[reviewer]"))
		case "assignee":
			badges = append(badges, dimStyle.Render("[assignee]"))
		}
	}

	meta := fmt.Sprintf("@%s  •  %s  •  %s",
		pr.Author, pr.Repo, pr.CreatedAt.Format("2006-01-02 15:04 UTC"))
	sb.WriteString(detailMetaStyle.Render(meta))
	if len(badges) > 0 {
		sb.WriteString("  " + strings.Join(badges, "  "))
	}
	sb.WriteString("\n\n")

	// Body (rendered as markdown)
	if pr.Body != "" {
		bodyW := w - 2
		if bodyW < 20 {
			bodyW = 20
		}
		sb.WriteString(renderMarkdown(pr.Body, bodyW))
	} else {
		sb.WriteString(dimStyle.Render("  (no description)"))
		sb.WriteString("\n\n")
	}

	// Stats
	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render(strings.Repeat("─", w-2)))
	sb.WriteString("\n")
	sb.WriteString(statsStyle.Render(fmt.Sprintf(
		"  Commits: %d   Files: %d   +%d / -%d",
		pr.Commits, pr.ChangedFiles, pr.Additions, pr.Deletions,
	)))
	sb.WriteString("\n")

	return sb.String()
}

func renderMarkdown(body string, width int) string {
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return body
	}
	out, err := r.Render(body)
	if err != nil {
		return body
	}
	return out
}

// ── Footer ───────────────────────────────────────────────────────────────────

func (m Model) renderFooter() string {
	sep := dimStyle.Render(strings.Repeat("─", m.width))

	hints := []string{
		keyHint("↑↓/jk", "nav"),
		keyHint("/", "filter"),
		keyHint("s", "sort"),
		keyHint("r", "review"),
		keyHint("o", "open"),
		keyHint("m", "mark read"),
		keyHint("F5", "refresh"),
		keyHint("PgUp/Dn", "scroll"),
		keyHint("q", "quit"),
	}
	if len(m.filtered) > 0 {
		count := dimStyle.Render(fmt.Sprintf("  [%d/%d]", m.selected+1, len(m.filtered)))
		hints = append(hints, count)
	}

	help := statusStyle.Render(strings.Join(hints, "  "))
	return lipgloss.JoinVertical(lipgloss.Left, sep, help)
}

var statusStyle = lipgloss.NewStyle().Foreground(colorGray)

func keyHint(key, desc string) string {
	return keyStyle.Render(key) + dimStyle.Render(":"+desc)
}

// ── String utilities ─────────────────────────────────────────────────────────

// runesTruncate truncates s to at most max runes, appending "…" if cut.
func runesTruncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

// runesTruncateWide is the same but with a wider ellipsis cutoff for display.
func runesTruncateWide(s string, max int) string {
	return runesTruncate(s, max)
}
