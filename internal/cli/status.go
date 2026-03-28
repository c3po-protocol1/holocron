package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/c3po-protocol1/holocron/internal/collector"
	"github.com/c3po-protocol1/holocron/internal/tui"
)

// FormatStatus formats sessions as a human-readable status summary.
func FormatStatus(sessions []collector.SessionState, now time.Time) string {
	if len(sessions) == 0 {
		return "No sessions found."
	}

	var b strings.Builder

	// Header
	noun := "sessions"
	if len(sessions) == 1 {
		noun = "session"
	}
	b.WriteString(fmt.Sprintf("Holocron — %d %s\n\n", len(sessions), noun))

	// Session rows
	for i, s := range sessions {
		if i > 0 {
			b.WriteString("\n")
		}

		indicator := statusDot(s.Status)
		sessionID := tui.TruncateID(s.SessionID)

		elapsed := time.Duration(0)
		if s.StartedAt > 0 {
			elapsed = now.Sub(time.Unix(0, s.StartedAt*int64(time.Millisecond)))
		}
		elapsedStr := tui.FormatElapsed(elapsed)

		statusStr := formatStatusLabel(s.Status)

		// Line 1: indicator source sessionID status elapsed workspace
		line1 := fmt.Sprintf("%s %-12s %-12s %-10s %7s",
			indicator, s.Source, sessionID, statusStr, elapsedStr)
		if s.Workspace != "" {
			line1 += "  " + s.Workspace
		}
		b.WriteString(line1)

		// Line 2: tool → target or token info
		if s.CurrentTool != "" {
			activity := s.CurrentTool
			if s.CurrentTarget != "" {
				activity += " → " + s.CurrentTarget
			}
			b.WriteString(fmt.Sprintf("\n%s %s", strings.Repeat(" ", 27), activity))
		}

		if s.TokenUsage != nil {
			total := s.TokenUsage.Input + s.TokenUsage.Output + s.TokenUsage.CacheRead
			b.WriteString(fmt.Sprintf("\n%s tokens: %s", strings.Repeat(" ", 27), formatTokenCount(total)))
		}
	}

	return b.String()
}

// FilterSessions filters sessions by active status and/or source type.
func FilterSessions(sessions []collector.SessionState, activeOnly bool, source string) []collector.SessionState {
	if !activeOnly && source == "" {
		return sessions
	}

	var result []collector.SessionState
	for _, s := range sessions {
		if activeOnly && !isActive(s.Status) {
			continue
		}
		if source != "" && s.Source != source {
			continue
		}
		result = append(result, s)
	}
	return result
}

func isActive(status collector.SessionStatus) bool {
	switch status {
	case collector.StatusThinking, collector.StatusToolRunning, collector.StatusWaiting:
		return true
	default:
		return false
	}
}

func statusDot(status collector.SessionStatus) string {
	switch status {
	case collector.StatusThinking, collector.StatusToolRunning:
		return "●"
	case collector.StatusIdle, collector.StatusWaiting:
		return "◌"
	case collector.StatusError:
		return "✕"
	case collector.StatusDone:
		return "✓"
	default:
		return "◌"
	}
}

func formatStatusLabel(status collector.SessionStatus) string {
	switch status {
	case collector.StatusToolRunning:
		return "running"
	default:
		return string(status)
	}
}

func formatTokenCount(total int64) string {
	if total >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(total)/1_000_000)
	}
	if total >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(total)/1_000)
	}
	return fmt.Sprintf("%d", total)
}
