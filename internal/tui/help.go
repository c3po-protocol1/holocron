package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderHelp renders the help overlay content.
func RenderHelp(keys KeyMap, width int) string {
	bindings := []struct {
		key  string
		desc string
	}{
		{"↑/k", "Move up"},
		{"↓/j", "Move down"},
		{"q / Ctrl+C", "Quit"},
		{"?", "Toggle help"},
		{"r", "Force refresh"},
		{"a", "toggle active-only filter"},
	}

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
