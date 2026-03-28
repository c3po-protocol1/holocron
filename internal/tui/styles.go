package tui

import "github.com/charmbracelet/lipgloss"

// Status indicator characters.
const (
	StatusDotActive = "●"
	StatusDotIdle   = "◌"
	StatusDotError  = "✕"
	StatusDotDone   = "✓"
)

// Colors.
var (
	colorGreen  = lipgloss.Color("#00FF00")
	colorRed    = lipgloss.Color("#FF0000")
	colorBlue   = lipgloss.Color("#5F87FF")
	colorDim    = lipgloss.Color("#666666")
	colorWhite  = lipgloss.Color("#FFFFFF")
	colorYellow = lipgloss.Color("#FFFF00")
)

// Styles.
var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorYellow)

	selectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorWhite)

	normalRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CCCCCC"))

	dimStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	activeIndicatorStyle = lipgloss.NewStyle().
				Foreground(colorGreen)

	idleIndicatorStyle = lipgloss.NewStyle().
				Foreground(colorDim)

	errorIndicatorStyle = lipgloss.NewStyle().
				Foreground(colorRed)

	doneIndicatorStyle = lipgloss.NewStyle().
				Foreground(colorBlue)

	footerStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	helpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorYellow)

	helpKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CCCCCC"))
)
