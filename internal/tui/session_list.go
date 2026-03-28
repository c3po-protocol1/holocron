package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/c3po-protocol1/holocron/internal/collector"
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
