package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/c3po-protocol1/holocron/internal/collector"
)

// eventMsg wraps a MonitorEvent received from the event channel.
type eventMsg collector.MonitorEvent

// tickMsg is sent periodically to update elapsed times.
type tickMsg time.Time

// Model is the main Bubbletea model for the TUI.
type Model struct {
	sessions   []collector.SessionState
	cursor     int
	events     <-chan collector.MonitorEvent
	showHelp   bool
	activeOnly bool
	width      int
	height     int
	eventCount int
	keys       KeyMap
}

// New creates a new TUI Model.
func New(events <-chan collector.MonitorEvent, sessions []collector.SessionState) Model {
	s := make([]collector.SessionState, len(sessions))
	copy(s, sessions)
	return Model{
		sessions: s,
		events:   events,
		keys:     DefaultKeyMap(),
		width:    80,
		height:   24,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.waitForEvent(), tickCmd())
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If help is shown, any key closes it except quit
		if m.showHelp {
			switch {
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit
			default:
				m.showHelp = false
				return m, nil
			}
		}

		// Compute visible session count for cursor bounds
		visible := m.sessions
		if m.activeOnly {
			visible = filterActive(m.sessions)
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(visible)-1 {
				m.cursor++
			}
		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
		case key.Matches(msg, m.keys.Active):
			m.activeOnly = !m.activeOnly
			// Re-compute visible after toggle
			visible = m.sessions
			if m.activeOnly {
				visible = filterActive(m.sessions)
			}
			if m.cursor >= len(visible) {
				m.cursor = 0
			}
		case key.Matches(msg, m.keys.Refresh):
			// Force re-render by returning nil cmd
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case eventMsg:
		m.applyEvent(collector.MonitorEvent(msg))
		return m, m.waitForEvent()

	case tickMsg:
		return m, tickCmd()
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	var b strings.Builder

	// Header
	header := headerStyle.Render("Holocron 🔭")
	b.WriteString(header)
	b.WriteString("\n\n")

	// Compute visible sessions
	visible := m.sessions
	if m.activeOnly {
		visible = filterActive(m.sessions)
	}
	hiddenCount := len(m.sessions) - len(visible)

	if m.showHelp {
		b.WriteString(RenderHelp(m.keys, m.width))
		b.WriteString("\n\n")
	} else {
		// Column headers
		colHeader := fmt.Sprintf("  %-14s %-13s %-10s %s", "SOURCE", "SESSION", "STATUS", "ELAPSED")
		b.WriteString(dimStyle.Render(colHeader))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(strings.Repeat("─", min(m.width, 60))))
		b.WriteString("\n")

		// Session list
		if m.activeOnly && len(visible) == 0 && len(m.sessions) > 0 {
			b.WriteString(dimStyle.Render("No active sessions. Press 'a' to show all."))
		} else {
			b.WriteString(RenderSessionList(visible, m.cursor, time.Now()))
		}
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("\n")
	separator := dimStyle.Render(strings.Repeat("─", min(m.width, 60)))
	b.WriteString(separator)
	b.WriteString("\n")

	footerKeys := footerStyle.Render("[q]uit  [?]help  [r]efresh")
	b.WriteString(footerKeys)
	b.WriteString("\n")

	activeCount := 0
	for _, s := range m.sessions {
		if s.Status == collector.StatusThinking || s.Status == collector.StatusToolRunning {
			activeCount++
		}
	}

	filterLabel := "[a]ctive: off"
	if m.activeOnly {
		filterLabel = fmt.Sprintf("[a]ctive: on (%d hidden)", hiddenCount)
	}

	stats := footerStyle.Render(fmt.Sprintf("%d sessions │ %d active │ %d events │ %s",
		len(m.sessions), activeCount, m.eventCount, filterLabel))
	b.WriteString(stats)

	// Apply width constraint
	return lipgloss.NewStyle().MaxWidth(m.width).Render(b.String())
}

// applyEvent updates or creates a session state from an event.
func (m *Model) applyEvent(ev collector.MonitorEvent) {
	m.eventCount++

	for i, s := range m.sessions {
		if s.SessionID == ev.SessionID && s.Source == ev.Source {
			m.sessions[i].Status = ev.Status
			m.sessions[i].LastEventAt = ev.Timestamp
			m.sessions[i].EventCount++
			if ev.Workspace != "" {
				m.sessions[i].Workspace = ev.Workspace
			}
			if ev.Detail != nil {
				if ev.Detail.Tool != "" {
					m.sessions[i].CurrentTool = ev.Detail.Tool
				}
				if ev.Detail.Target != "" {
					m.sessions[i].CurrentTarget = ev.Detail.Target
				}
			}
			// Clear tool info when not actively running
			if ev.Status == collector.StatusIdle || ev.Status == collector.StatusDone {
				m.sessions[i].CurrentTool = ""
				m.sessions[i].CurrentTarget = ""
			}
			return
		}
	}

	// New session
	newSession := collector.SessionState{
		Source:      ev.Source,
		SessionID:   ev.SessionID,
		Workspace:   ev.Workspace,
		Status:      ev.Status,
		StartedAt:   ev.Timestamp,
		LastEventAt: ev.Timestamp,
		EventCount:  1,
	}
	if ev.Detail != nil {
		newSession.CurrentTool = ev.Detail.Tool
		newSession.CurrentTarget = ev.Detail.Target
	}
	m.sessions = append(m.sessions, newSession)
}

// waitForEvent returns a Cmd that waits for the next event from the channel.
func (m Model) waitForEvent() tea.Cmd {
	if m.events == nil {
		return nil
	}
	ch := m.events
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return eventMsg(ev)
	}
}

// tickCmd returns a Cmd that sends a tick every second.
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
