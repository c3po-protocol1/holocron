package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/c3po-protocol1/holocron/internal/collector"
	"github.com/c3po-protocol1/holocron/internal/labels"
)

// FormatElapsed formats a duration into a human-readable string.
func FormatElapsed(d time.Duration) string {
	if d < 0 {
		d = 0
	}

	totalSeconds := int(d.Seconds())

	if totalSeconds < 60 {
		return fmt.Sprintf("%ds", totalSeconds)
	}

	minutes := totalSeconds / 60
	seconds := totalSeconds % 60

	if minutes < 60 {
		return fmt.Sprintf("%dm %02ds", minutes, seconds)
	}

	hours := minutes / 60
	minutes = minutes % 60
	return fmt.Sprintf("%dh %02dm", hours, minutes)
}

// StatusIndicator returns the styled status indicator for a session status.
func StatusIndicator(status collector.SessionStatus) string {
	switch status {
	case collector.StatusThinking, collector.StatusToolRunning:
		return activeIndicatorStyle.Render(StatusDotActive)
	case collector.StatusIdle, collector.StatusWaiting:
		return idleIndicatorStyle.Render(StatusDotIdle)
	case collector.StatusError:
		return errorIndicatorStyle.Render(StatusDotError)
	case collector.StatusDone:
		return doneIndicatorStyle.Render(StatusDotDone)
	default:
		return idleIndicatorStyle.Render(StatusDotIdle)
	}
}

// TruncateID truncates a session ID to 10 characters followed by "..".
func TruncateID(id string) string {
	if len(id) <= 10 {
		return id
	}
	return id[:10] + ".."
}

// RenderSessionRow renders a single session row.
func RenderSessionRow(s collector.SessionState, selected bool, now time.Time) string {
	if s.Source == "openclaw" {
		return renderOpenClawRow(s, selected, now)
	}
	return renderDefaultRow(s, selected, now)
}

func renderDefaultRow(s collector.SessionState, selected bool, now time.Time) string {
	var b strings.Builder

	elapsed := now.Sub(time.Unix(0, s.LastEventAt*int64(time.Millisecond)))
	if s.StartedAt > 0 {
		elapsed = now.Sub(time.Unix(0, s.StartedAt*int64(time.Millisecond)))
	}

	indicator := StatusIndicator(s.Status)
	sessionID := TruncateID(s.SessionID)
	elapsedStr := FormatElapsed(elapsed)

	rowStyle := normalRowStyle
	cursor := "  "
	if selected {
		rowStyle = selectedRowStyle
		cursor = "▶ "
	}

	// Line 1: cursor source | session ID | status | elapsed
	line1 := fmt.Sprintf("%s%-14s %s  %s  %s",
		cursor,
		rowStyle.Render(s.Source),
		rowStyle.Render(sessionID),
		indicator,
		dimStyle.Render(elapsedStr),
	)
	b.WriteString(line1)

	// Line 2: workspace path
	if s.Workspace != "" {
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %-14s %s", "", dimStyle.Render(s.Workspace)))
	}

	// Line 3: current tool + target (if active)
	if s.CurrentTool != "" {
		b.WriteString("\n")
		activity := s.CurrentTool
		if s.CurrentTarget != "" {
			activity += " → " + s.CurrentTarget
		}
		b.WriteString(fmt.Sprintf("  %-14s %s", "", dimStyle.Render(activity)))
	}

	return b.String()
}

func renderOpenClawRow(s collector.SessionState, selected bool, now time.Time) string {
	var b strings.Builder

	elapsed := now.Sub(time.Unix(0, s.LastEventAt*int64(time.Millisecond)))
	if s.StartedAt > 0 {
		elapsed = now.Sub(time.Unix(0, s.StartedAt*int64(time.Millisecond)))
	}

	indicator := StatusIndicator(s.Status)
	elapsedStr := FormatElapsed(elapsed)

	rowStyle := normalRowStyle
	cursor := "  "
	if selected {
		rowStyle = selectedRowStyle
		cursor = "▶ "
	}

	// Agent name from labels (fallback to truncated session ID)
	agent := s.Labels["agent"]
	if agent == "" {
		agent = TruncateID(s.SessionID)
	}

	statusWord := "idle"
	switch s.Status {
	case collector.StatusThinking, collector.StatusToolRunning:
		statusWord = "active"
	case collector.StatusError:
		statusWord = "error"
	case collector.StatusDone:
		statusWord = "done"
	}

	// Line 1: cursor source | agent | status indicator + word | elapsed
	line1 := fmt.Sprintf("%s%-14s %-14s %s %s  %s",
		cursor,
		rowStyle.Render(s.Source),
		rowStyle.Render(agent),
		indicator,
		dimStyle.Render(statusWord),
		dimStyle.Render(elapsedStr),
	)
	b.WriteString(line1)

	// Line 2: channel:session_type
	sessionType := s.Labels["session_type"]
	channel := s.Labels["channel"]
	if sessionType != "" {
		line2 := sessionType
		if channel != "" {
			line2 = channel + ":" + sessionType
		}
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %-14s %s", "", dimStyle.Render(line2)))
	}

	// Line 3: tokens info + model
	if s.TokenUsage != nil || s.Labels["model"] != "" {
		var parts []string
		if s.TokenUsage != nil {
			total := s.TokenUsage.Input + s.TokenUsage.Output
			tokenStr := formatTokenCount(total)
			if budget, ok := s.Labels["total_tokens"]; ok {
				if pct, ok2 := s.Labels["percent_used"]; ok2 {
					tokenStr = fmt.Sprintf("%s / %s (%s%%)", tokenStr, formatTokenCountStr(budget), pct)
				}
			}
			parts = append(parts, "tokens: "+tokenStr)
		}
		if model := s.Labels["model"]; model != "" {
			parts = append(parts, model)
		}
		if len(parts) > 0 {
			b.WriteString("\n")
			b.WriteString(fmt.Sprintf("  %-14s %s", "", dimStyle.Render(strings.Join(parts, " · "))))
		}
	}

	return b.String()
}

// formatTokenCount formats a token count into a human-readable string (e.g. 111k, 1M).
func formatTokenCount(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%dM", n/1_000_000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%dk", n/1000)
	}
	return fmt.Sprintf("%d", n)
}

// formatTokenCountStr formats a token count string into human-readable form.
func formatTokenCountStr(s string) string {
	var n int64
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return s
	}
	return formatTokenCount(n)
}

// filterActive returns only sessions that are not idle or done.
func filterActive(sessions []collector.SessionState) []collector.SessionState {
	var out []collector.SessionState
	for _, s := range sessions {
		if s.Status != collector.StatusIdle && s.Status != collector.StatusDone {
			out = append(out, s)
		}
	}
	return out
}

// RenderEmptyState renders the empty state message.
func RenderEmptyState() string {
	return dimStyle.Render("No sessions detected. Check your config: ~/.holocron/config.yaml")
}

// RenderSessionList renders the full session list.
func RenderSessionList(sessions []collector.SessionState, cursor int, now time.Time) string {
	if len(sessions) == 0 {
		return RenderEmptyState()
	}

	var rows []string
	for i, s := range sessions {
		rows = append(rows, RenderSessionRow(s, i == cursor, now))
	}
	return strings.Join(rows, "\n\n")
}

// RenderGroupedList renders session groups with headers and indented sessions.
func RenderGroupedList(groups []labels.SessionGroup, cursor int, now time.Time, width int) string {
	flat := FlattenGroups(groups)
	if len(flat) == 0 {
		return RenderEmptyState()
	}

	var b strings.Builder
	sessionIdx := 0
	for gi, g := range groups {
		if gi > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(renderGroupHeader(g, width))
		b.WriteString("\n")
		for si, s := range g.Sessions {
			if si > 0 {
				b.WriteString("\n\n")
			}
			b.WriteString(RenderSessionRow(s, sessionIdx == cursor, now))
			sessionIdx++
		}
	}

	return b.String()
}

// renderGroupHeader renders a group header line like: ─── r2d2 (2 sessions, 1 active) ──────
func renderGroupHeader(g labels.SessionGroup, width int) string {
	countStr := fmt.Sprintf("%d session", len(g.Sessions))
	if len(g.Sessions) != 1 {
		countStr += "s"
	}
	if g.Active > 0 {
		countStr += fmt.Sprintf(", %d active", g.Active)
	}

	label := fmt.Sprintf("─── %s (%s) ", g.Label, countStr)

	// Fill remaining width with ─
	remaining := width - len(label)
	if remaining < 3 {
		remaining = 3
	}
	line := label + strings.Repeat("─", remaining)

	return groupHeaderStyle.Render(line)
}

// FlattenGroups returns all sessions across groups as a flat slice, preserving group order.
func FlattenGroups(groups []labels.SessionGroup) []collector.SessionState {
	var out []collector.SessionState
	for _, g := range groups {
		out = append(out, g.Sessions...)
	}
	return out
}
