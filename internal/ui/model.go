package ui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"gh-dashboard/internal/config"
	"gh-dashboard/internal/github"
	"gh-dashboard/internal/state"
)

// ReviewFilter controls which PRs are shown based on review decision.
type ReviewFilter int

const (
	ReviewAll ReviewFilter = iota
	ReviewPending
	ReviewApproved
	ReviewChangesRequested
)

func (r ReviewFilter) String() string {
	switch r {
	case ReviewPending:
		return "Pending"
	case ReviewApproved:
		return "Approved"
	case ReviewChangesRequested:
		return "Changes Requested"
	default:
		return "All"
	}
}

// detailCache holds the last glamour-rendered detail pane so we only re-render
// when the selected PR or the available width actually changes.
type detailCache struct {
	prID    string
	width   int
	content string
}

type Model struct {
	config *config.Config
	client *github.Client
	state  *state.State

	// Data
	prs      []github.PR
	filtered []github.PR
	selected int
	listOff  int // list scroll offset

	// Filter
	filterMode  bool
	filterInput textinput.Model

	// State
	reviewFilter ReviewFilter
	sortAsc      bool

	// Detail scrolling
	viewport    viewport.Model
	renderCache detailCache

	// Loading / error
	loading bool
	err     error

	// Terminal dimensions
	width  int
	height int
}

func NewModel(cfg *config.Config, client *github.Client, st *state.State) Model {
	fi := textinput.New()
	fi.Placeholder = "filter by title or repo..."
	fi.CharLimit = 100

	return Model{
		config:      cfg,
		client:      client,
		state:       st,
		filterInput: fi,
		loading:     true,
		sortAsc:     false, // newest first by default
		viewport:    viewport.New(0, 0),
	}
}

func (m Model) Init() tea.Cmd {
	return fetchPRsCmd(m.config, m.client)
}

// ── Layout helpers ──────────────────────────────────────────────────────────

func (m Model) listWidth() int {
	w := m.width * 38 / 100
	if w < 28 {
		w = 28
	}
	return w
}

func (m Model) detailWidth() int {
	w := m.width - m.listWidth() - 1 // 1 for the │ separator
	if w < 20 {
		w = 20
	}
	return w
}

// contentHeight returns the number of lines available for the list/detail area.
func (m Model) contentHeight() int {
	h := m.height - 4 // 2-line header + 2-line footer
	if h < 1 {
		h = 1
	}
	return h
}

// listVisibleHeight returns how many list rows fit, accounting for filter input.
func (m Model) listVisibleHeight() int {
	h := m.contentHeight()
	if m.filterMode {
		h--
	}
	if h < 1 {
		h = 1
	}
	return h
}

// ── Filter / sort ────────────────────────────────────────────────────────────

func (m *Model) applyFilters() {
	filterText := strings.ToLower(m.filterInput.Value())

	var result []github.PR
	for _, pr := range m.prs {
		// Review state filter
		switch m.reviewFilter {
		case ReviewPending:
			if pr.ReviewDecision != "" && pr.ReviewDecision != "REVIEW_REQUIRED" {
				continue
			}
		case ReviewApproved:
			if pr.ReviewDecision != "APPROVED" {
				continue
			}
		case ReviewChangesRequested:
			if pr.ReviewDecision != "CHANGES_REQUESTED" {
				continue
			}
		}

		// Text filter against title and repo
		if filterText != "" {
			matchTitle := strings.Contains(strings.ToLower(pr.Title), filterText)
			matchRepo := strings.Contains(strings.ToLower(pr.Repo), filterText)
			if !matchTitle && !matchRepo {
				continue
			}
		}

		result = append(result, pr)
	}

	// Sort by creation date
	sort.Slice(result, func(i, j int) bool {
		if m.sortAsc {
			return result[i].CreatedAt.Before(result[j].CreatedAt)
		}
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	m.filtered = result

	// Clamp selection
	if len(m.filtered) == 0 {
		m.selected = 0
		m.listOff = 0
	} else if m.selected >= len(m.filtered) {
		m.selected = len(m.filtered) - 1
	}
	m.adjustListScroll()
}

func (m *Model) adjustListScroll() {
	h := m.listVisibleHeight()
	if m.selected < m.listOff {
		m.listOff = m.selected
	}
	if m.selected >= m.listOff+h {
		m.listOff = m.selected - h + 1
	}
}

// ── Detail pane ──────────────────────────────────────────────────────────────

// updateDetail re-renders the selected PR into the viewport and marks it read.
// Glamour rendering is skipped when the same PR and width are already cached.
func (m *Model) updateDetail() {
	if m.width == 0 {
		return
	}
	if len(m.filtered) == 0 {
		m.renderCache = detailCache{} // viewport is about to show non-PR content
		m.viewport.SetContent(dimStyle.Render("\n  No PRs to display.\n\n  Press F5 to refresh or adjust filters."))
		return
	}
	if m.selected >= len(m.filtered) {
		return
	}
	pr := m.filtered[m.selected]

	// Auto-mark as read when navigated to
	m.state.MarkRead(pr.ID)
	_ = m.state.Save()

	w := m.detailWidth()
	if m.renderCache.prID == pr.ID && m.renderCache.width == w {
		// Same PR and same width — reuse cached render, just reset scroll.
		m.viewport.GotoTop()
		return
	}

	content := renderDetail(pr, w)
	m.renderCache = detailCache{prID: pr.ID, width: w, content: content}
	m.viewport.SetContent(content)
	m.viewport.GotoTop()
}
