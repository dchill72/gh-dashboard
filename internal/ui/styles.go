package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorBlue   = lipgloss.Color("#5C9BEB")
	colorGreen  = lipgloss.Color("#4DC38B")
	colorGray   = lipgloss.Color("#6C757D")
	colorYellow = lipgloss.Color("#E5C07B")
	colorRed    = lipgloss.Color("#E06C75")
	colorWhite  = lipgloss.Color("#EFEFEF")
	colorBgSel  = lipgloss.Color("#2D4A6A")
	colorBgSelR = lipgloss.Color("#2D5A3D")

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite)

	headerTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorBlue)

	dimStyle = lipgloss.NewStyle().
			Foreground(colorGray)

	metaStyle = lipgloss.NewStyle().
			Foreground(colorWhite)

	keyStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	reviewFilterStyle = lipgloss.NewStyle().
				Foreground(colorYellow)

	loadingStyle = lipgloss.NewStyle().
			Foreground(colorGray).
			Italic(true)

	// List item styles — applied to the whole row
	unreadStyle = lipgloss.NewStyle().
			Foreground(colorBlue)

	readStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	selectedUnreadStyle = lipgloss.NewStyle().
				Foreground(colorWhite).
				Background(colorBgSel).
				Bold(true)

	selectedReadStyle = lipgloss.NewStyle().
				Foreground(colorWhite).
				Background(colorBgSelR).
				Bold(true)

	// Detail pane
	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorWhite)

	detailMetaStyle = lipgloss.NewStyle().
			Foreground(colorGray)

	statsStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	draftStyle = lipgloss.NewStyle().
			Foreground(colorYellow)

	approvedStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	changesStyle = lipgloss.NewStyle().
			Foreground(colorRed)

	reviewRequiredStyle = lipgloss.NewStyle().
				Foreground(colorYellow)
)
