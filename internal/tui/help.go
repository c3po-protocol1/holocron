package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderHelp renders the help overlay content for the list view.
func RenderHelp(keys KeyMap, width int) string {
	return renderHelpOverlay(listBindings(keys), width)
}

// RenderDetailHelp renders the help overlay content for the detail view.
func RenderDetailHelp(keys KeyMap, width int) string {
	return renderHelpOverlay(detailBindings(keys), width)
}

type helpBinding struct {
	key  string
	desc string
}

func listBindings(keys KeyMap) []helpBinding {
	return []helpBinding{
		{"↑/k", "Move up"},
		{"↓/j", "Move down"},
		{"Enter", "Open session detail"},
		{"q / Ctrl+C", "Quit"},
		{"?", "Toggle help"},
		{"r", "Force refresh"},
		{"a", "toggle active-only filter"},
		{"g", "cycle group mode (none/agent/channel)"},
	}
}

func detailBindings(keys KeyMap) []helpBinding {
	return []helpBinding{
		{"Esc", "Back to session list"},
		{"↑/k", "Scroll up"},
		{"↓/j", "Scroll down"},
		{"g", "Jump to top"},
		{"G", "Jump to bottom"},
		{"f", "Toggle follow mode"},
		{"q / Ctrl+C", "Quit"},
		{"?", "Toggle help"},
	}
}

func renderHelpOverlay(bindings []helpBinding, width int) string {
	var b strings.Builder
	b.WriteString(helpTitleStyle.Render("Key Bindings"))
	b.WriteString("\n\n")

	for _, bind := range bindings {
		line := fmt.Sprintf("  %s  %s",
			helpKeyStyle.Render(fmt.Sprintf("%-14s", bind.key)),
			helpDescStyle.Render(bind.desc),
		)
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Press ? to close"))

	overlay := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorDim).
		Padding(1, 2).
		Render(b.String())

	return overlay
}
