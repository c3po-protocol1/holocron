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

// ViewMode represents which view is currently active.
type ViewMode string

const (
	ViewList   ViewMode = "list"
	ViewDetail ViewMode = "detail"
)

// EventLoader loads events for a session (subset of store.Store).
type EventLoader interface {
	GetEvents(source, sessionID string, since int64, limit int) ([]collector.MonitorEvent, error)
}

// eventMsg wraps a MonitorEvent received from the event channel.
type eventMsg collector.MonitorEvent

// tickMsg is sent periodically to update elapsed times.
type tickMsg time.Time

// Model is the main Bubbletea model for the TUI.
type Model struct {
	sessions   []collector.SessionState
	cursor     int
	events     <-chan collector.MonitorEvent
	store      EventLoader
	showHelp   bool
	activeOnly bool
	width      int
	height     int
	eventCount int
	keys       KeyMap
	view       ViewMode
	detail     *DetailModel
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
		view:     ViewList,
	}
}

// NewWithStore creates a new TUI Model with a store for loading events.
func NewWithStore(events <-chan collector.MonitorEvent, sessions []collector.SessionState, store EventLoader) Model {
	m := New(events, sessions)
	m.store = store
	return m
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

		// Route to current view
		if m.view == ViewDetail {
			return m.updateDetail(msg)
		}
		return m.updateList(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.detail != nil {
			m.detail.SetSize(msg.Width, msg.Height)
		}
		return m, nil

	case eventMsg:
		ev := collector.MonitorEvent(msg)
		m.applyEvent(ev)
		// Route event to detail model if active and matching
		if m.view == ViewDetail && m.detail != nil && m.detail.MatchesSession(ev.Source, ev.SessionID) {
			m.detail.AppendEvent(ev)
		}
		return m, m.waitForEvent()

	case tickMsg:
		return m, tickCmd()
	}

	return m, nil
}

// updateList handles key events in the list view.
func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		visible = m.sessions
		if m.activeOnly {
			visible = filterActive(m.sessions)
		}
		if m.cursor >= len(visible) {
			m.cursor = 0
		}
	case key.Matches(msg, m.keys.Enter):
		return m.openDetail(visible)
	case key.Matches(msg, m.keys.Refresh):
		// Force re-render
	}
	return m, nil
}

// updateDetail handles key events in the detail view.
func (m Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Back):
		m.view = ViewList
		m.detail = nil
		return m, nil
	case key.Matches(msg, m.keys.Up):
		m.detail.ScrollUp()
	case key.Matches(msg, m.keys.Down):
		m.detail.ScrollDown()
	case key.Matches(msg, m.keys.Top):
		m.detail.ScrollToTop()
	case key.Matches(msg, m.keys.Bottom):
		m.detail.ScrollToBottom()
	case key.Matches(msg, m.keys.Follow):
		m.detail.ToggleFollow()
	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
	}
	return m, nil
}

// openDetail transitions from list to detail view.
func (m Model) openDetail(visible []collector.SessionState) (tea.Model, tea.Cmd) {
	if len(visible) == 0 || m.cursor >= len(visible) {
		return m, nil
	}

	session := visible[m.cursor]

	// Load events from store
	var events []collector.MonitorEvent
	if m.store != nil {
		loaded, err := m.store.GetEvents(session.Source, session.SessionID, 0, 200)
		if err == nil {
			events = loaded
		}
	}

	m.detail = NewDetailModel(session, events, m.width, m.height)
	m.view = ViewDetail
	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	if m.view == ViewDetail && m.detail != nil {
		if m.showHelp {
			return m.renderDetailWithHelp()
		}
		return m.detail.View()
	}

	return m.renderListView()
}

func (m Model) renderDetailWithHelp() string {
	var b strings.Builder
	header := headerStyle.Render("Holocron 🔭 — Session Detail")
	b.WriteString(header)
	b.WriteString("\n\n")
	b.WriteString(RenderDetailHelp(m.keys, m.width))
	return lipgloss.NewStyle().MaxWidth(m.width).Render(b.String())
}

func (m Model) renderListView() string {
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
