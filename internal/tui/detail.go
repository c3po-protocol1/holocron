package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/c3po-protocol1/holocron/internal/collector"
)

// InfoPanelHeight is the fixed height of the info panel (lines including border).
const InfoPanelHeight = 13

// DetailModel is the Bubbletea sub-model for the session detail view.
type DetailModel struct {
	session collector.SessionState
	events  []collector.MonitorEvent
	follow  bool
	scroll  int // scroll offset into event log (0 = top)
	width   int
	height  int
}

// NewDetailModel creates a new detail model with preloaded events.
func NewDetailModel(session collector.SessionState, events []collector.MonitorEvent, width, height int) *DetailModel {
	evs := make([]collector.MonitorEvent, len(events))
	copy(evs, events)

	dm := &DetailModel{
		session: session,
		events:  evs,
		follow:  true,
		width:   width,
		height:  height,
	}
	// Start scrolled to bottom (follow mode)
	dm.scrollToBottom()
	return dm
}

// MatchesSession returns true if the event belongs to this detail's session.
func (dm *DetailModel) MatchesSession(source, sessionID string) bool {
	return dm.session.Source == source && dm.session.SessionID == sessionID
}

// AppendEvent adds a new event and updates session state.
func (dm *DetailModel) AppendEvent(ev collector.MonitorEvent) {
	dm.events = append(dm.events, ev)
	dm.session.Status = ev.Status
	dm.session.LastEventAt = ev.Timestamp
	dm.session.EventCount++
	if ev.Workspace != "" {
		dm.session.Workspace = ev.Workspace
	}
	if ev.Detail != nil {
		if ev.Detail.Tool != "" {
			dm.session.CurrentTool = ev.Detail.Tool
		}
		if ev.Detail.Target != "" {
			dm.session.CurrentTarget = ev.Detail.Target
		}
		if ev.Detail.TokenUsage != nil {
			dm.session.TokenUsage = ev.Detail.TokenUsage
		}
	}
	if ev.Status == collector.StatusIdle || ev.Status == collector.StatusDone {
		dm.session.CurrentTool = ""
		dm.session.CurrentTarget = ""
	}
	if dm.follow {
		dm.scrollToBottom()
	}
}

// UpdateSession updates the session state from an external source (e.g. app model).
func (dm *DetailModel) UpdateSession(s collector.SessionState) {
	dm.session = s
}

// SetSize updates the dimensions.
func (dm *DetailModel) SetSize(width, height int) {
	dm.width = width
	dm.height = height
}

// ScrollUp moves the event log scroll up by one line.
func (dm *DetailModel) ScrollUp() {
	if dm.scroll > 0 {
		dm.scroll--
		dm.follow = false
	}
}

// ScrollDown moves the event log scroll down by one line.
func (dm *DetailModel) ScrollDown() {
	maxScroll := dm.maxScroll()
	if dm.scroll < maxScroll {
		dm.scroll++
	}
	if dm.scroll >= maxScroll {
		dm.follow = true
	}
}

// ScrollToTop jumps to the top of the event log.
func (dm *DetailModel) ScrollToTop() {
	dm.scroll = 0
	dm.follow = false
}

// ScrollToBottom jumps to the bottom of the event log.
func (dm *DetailModel) ScrollToBottom() {
	dm.scrollToBottom()
	dm.follow = true
}

// ToggleFollow toggles follow mode.
func (dm *DetailModel) ToggleFollow() {
	dm.follow = !dm.follow
	if dm.follow {
		dm.scrollToBottom()
	}
}

func (dm *DetailModel) scrollToBottom() {
	maxScroll := dm.maxScroll()
	if maxScroll > 0 {
		dm.scroll = maxScroll
	} else {
		dm.scroll = 0
	}
}

func (dm *DetailModel) maxScroll() int {
	visibleLines := dm.eventLogHeight()
	total := len(dm.events)
	if total <= visibleLines {
		return 0
	}
	return total - visibleLines
}

func (dm *DetailModel) eventLogHeight() int {
	// Total height minus: header(2) + info panel + separator(1) + event log header(2) + footer(3)
	h := dm.height - InfoPanelHeight - 8
	if h < 1 {
		h = 1
	}
	return h
}

// View renders the detail view.
func (dm *DetailModel) View() string {
	var b strings.Builder
	now := time.Now()

	// Header
	header := headerStyle.Render("Holocron 🔭 — Session Detail")
	b.WriteString(header)
	b.WriteString("\n\n")

	// Info panel
	b.WriteString(RenderInfoPanel(dm.session, now))
	b.WriteString("\n\n")

	// Event log header
	eventLogTitle := dimStyle.Render("─ Event Log ")
	eventLogTitle += dimStyle.Render(strings.Repeat("─", max(0, min(dm.width, 60)-14)))
	b.WriteString(eventLogTitle)
	b.WriteString("\n")

	if len(dm.events) == 0 {
		b.WriteString(dimStyle.Render("  No events recorded for this session yet."))
	} else {
		visibleLines := dm.eventLogHeight()
		start := dm.scroll
		end := start + visibleLines
		if end > len(dm.events) {
			end = len(dm.events)
		}
		if start > len(dm.events) {
			start = len(dm.events)
		}
		for i := start; i < end; i++ {
			b.WriteString(FormatEventRow(dm.events[i]))
			if i < end-1 {
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n\n")

	// Footer
	separator := dimStyle.Render(strings.Repeat("─", min(dm.width, 60)))
	b.WriteString(separator)
	b.WriteString("\n")

	followLabel := "off"
	if dm.follow {
		followLabel = "on"
	}

	footerKeys := footerStyle.Render(fmt.Sprintf("[Esc]back  [↑↓]scroll  [G]bottom  [g]top  [f]ollow: %s", followLabel))
	b.WriteString(footerKeys)

	return lipgloss.NewStyle().MaxWidth(dm.width).Render(b.String())
}

// --- Info Panel ---

// RenderInfoPanel renders the session info panel.
func RenderInfoPanel(s collector.SessionState, now time.Time) string {
	var b strings.Builder

	// Status line
	statusStr := string(s.Status)
	if s.CurrentTool != "" {
		statusStr += " (" + s.CurrentTool + ")"
	}

	// Duration
	var elapsed time.Duration
	if s.StartedAt > 0 {
		elapsed = now.Sub(time.Unix(0, s.StartedAt*int64(time.Millisecond)))
	}

	b.WriteString(fmt.Sprintf("  Source:    %s\n", s.Source))
	b.WriteString(fmt.Sprintf("  Session:   %s\n", s.SessionID))
	if s.Workspace != "" {
		b.WriteString(fmt.Sprintf("  Workspace: %s\n", s.Workspace))
	}
	b.WriteString(fmt.Sprintf("  Status:    %s %s\n", StatusIndicator(s.Status), statusStr))
	b.WriteString(fmt.Sprintf("  Duration:  %s\n", FormatElapsed(elapsed)))

	// OpenClaw-specific: model
	if model := s.Labels["model"]; model != "" {
		b.WriteString(fmt.Sprintf("  Model:     %s\n", model))
	}

	// Token usage
	if s.TokenUsage != nil {
		b.WriteString(fmt.Sprintf("  Tokens:    %s\n", formatTokenDetailed(s.TokenUsage)))
	}

	// OpenClaw-specific: context percentage
	if pct := s.Labels["percent_used"]; pct != "" {
		if total := s.Labels["total_tokens"]; total != "" {
			used := ""
			if s.TokenUsage != nil {
				used = formatTokenCountFloat(s.TokenUsage.Input + s.TokenUsage.Output)
			}
			b.WriteString(fmt.Sprintf("  Context:   %s / %s (%s%%)\n", used, formatTokenCountStr(total), pct))
		}
	}

	b.WriteString(fmt.Sprintf("  Events:    %d", s.EventCount))

	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorDim).
		Padding(0, 1).
		Render(b.String())

	return panel
}

// --- Event Formatting (T3) ---

// EventIndicator returns the plain-text indicator for an event type.
func EventIndicator(event collector.EventType) string {
	switch event {
	case collector.EventToolStart, collector.EventToolEnd:
		return "●"
	case collector.EventMessage:
		return "○"
	case collector.EventStatusChange:
		return "◌"
	case collector.EventSessionStart:
		return "▶"
	case collector.EventSessionEnd:
		return "■"
	case collector.EventError:
		return "✕"
	default:
		return "·"
	}
}

// eventIndicatorStyled returns the styled indicator for an event type.
func eventIndicatorStyled(event collector.EventType) string {
	indicator := EventIndicator(event)
	switch event {
	case collector.EventToolStart, collector.EventToolEnd:
		return activeIndicatorStyle.Render(indicator)
	case collector.EventMessage:
		return dimStyle.Render(indicator)
	case collector.EventStatusChange:
		return idleIndicatorStyle.Render(indicator)
	case collector.EventSessionStart:
		return activeIndicatorStyle.Render(indicator)
	case collector.EventSessionEnd:
		return doneIndicatorStyle.Render(indicator)
	case collector.EventError:
		return errorIndicatorStyle.Render(indicator)
	default:
		return dimStyle.Render(indicator)
	}
}

// FormatEventRow formats a single event into a display row.
func FormatEventRow(ev collector.MonitorEvent) string {
	ts := time.Unix(0, ev.Timestamp*int64(time.Millisecond))
	timeStr := ts.Format("15:04:05")

	indicator := eventIndicatorStyled(ev.Event)
	eventType := fmt.Sprintf("%-14s", string(ev.Event))

	summary := eventSummary(ev)

	return fmt.Sprintf("  %s  %s %s %s",
		dimStyle.Render(timeStr),
		indicator,
		dimStyle.Render(eventType),
		summary,
	)
}

func eventSummary(ev collector.MonitorEvent) string {
	switch ev.Event {
	case collector.EventToolStart, collector.EventToolEnd:
		if ev.Detail != nil {
			s := ev.Detail.Tool
			if ev.Detail.Target != "" {
				s += " → " + ev.Detail.Target
			}
			return truncate(s, 50)
		}
	case collector.EventMessage:
		if ev.Detail != nil && ev.Detail.Message != "" {
			return truncate(ev.Detail.Message, 50)
		}
	case collector.EventStatusChange:
		return string(ev.Status)
	case collector.EventError:
		if ev.Detail != nil && ev.Detail.Message != "" {
			return errorIndicatorStyle.Render(truncate(ev.Detail.Message, 50))
		}
	}
	return ""
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

// formatTokenDetailed formats token usage for the info panel.
func formatTokenDetailed(tu *collector.TokenUsage) string {
	parts := []string{
		formatTokenCountFloat(tu.Input) + " in",
		formatTokenCountFloat(tu.Output) + " out",
	}
	if tu.CacheRead > 0 {
		parts = append(parts, formatTokenCountFloat(tu.CacheRead)+" cache")
	}
	return strings.Join(parts, " / ")
}

// formatTokenCountFloat formats a count with one decimal for k values.
func formatTokenCountFloat(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1000 {
		f := float64(n) / 1000
		if f == float64(int64(f)) {
			return fmt.Sprintf("%dk", n/1000)
		}
		return fmt.Sprintf("%.1fk", f)
	}
	return fmt.Sprintf("%d", n)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
