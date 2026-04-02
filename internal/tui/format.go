package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/c3po-protocol1/holocron/internal/collector"
)

// eventIcon returns the emoji icon for an event type per F12 spec.
func eventIcon(t collector.EventType) string {
	switch t {
	case collector.EventUserMessage:
		return "👤"
	case collector.EventAssistantMessage:
		return "🤖"
	case collector.EventToolStart:
		return "🔧"
	case collector.EventToolResult, collector.EventToolEnd:
		return "✅"
	case collector.EventStatusChange:
		return "◌"
	case collector.EventSessionStart:
		return "▶"
	case collector.EventSessionEnd:
		return "■"
	case collector.EventError:
		return "✕"
	default:
		return "○"
	}
}

// eventLabel returns the short label for verbose mode headers.
func eventLabel(t collector.EventType) string {
	switch t {
	case collector.EventUserMessage:
		return "USER"
	case collector.EventAssistantMessage:
		return "ASSISTANT"
	case collector.EventToolStart:
		return "TOOL"
	case collector.EventToolResult, collector.EventToolEnd:
		return "RESULT"
	case collector.EventMessage:
		return "MESSAGE"
	case collector.EventStatusChange:
		return "STATUS"
	case collector.EventSessionStart:
		return "SESSION START"
	case collector.EventSessionEnd:
		return "SESSION END"
	case collector.EventError:
		return "ERROR"
	default:
		return string(t)
	}
}

// compactSummary returns a one-liner summary for compact mode.
func compactSummary(ev collector.MonitorEvent) string {
	if ev.Detail == nil {
		if ev.Event == collector.EventStatusChange {
			return string(ev.Status)
		}
		return ""
	}
	d := ev.Detail

	switch ev.Event {
	case collector.EventUserMessage, collector.EventAssistantMessage:
		return coalesce(d.Content, d.Message)
	case collector.EventToolStart:
		s := d.Tool
		if d.Target != "" {
			s += " → " + d.Target
		}
		return s
	case collector.EventToolResult, collector.EventToolEnd:
		s := d.Tool
		if d.Target != "" {
			s += " → " + d.Target
		}
		if d.Message != "" {
			s += " (" + d.Message + ")"
		}
		return s
	case collector.EventStatusChange:
		return string(ev.Status)
	case collector.EventError:
		return d.Message
	case collector.EventMessage:
		return d.Message
	default:
		return d.Message
	}
}

// formatEventCompact formats a single event as a compact one-liner with emoji icon.
func formatEventCompact(ev collector.MonitorEvent) string {
	ts := time.Unix(0, ev.Timestamp*int64(time.Millisecond))
	timeStr := ts.Format("15:04:05")

	icon := eventIcon(ev.Event)
	eventType := fmt.Sprintf("%-14s", string(ev.Event))
	summary := truncate(compactSummary(ev), 50)

	return fmt.Sprintf("  %s  %s %s %s",
		dimStyle.Render(timeStr),
		icon,
		dimStyle.Render(eventType),
		summary,
	)
}

// formatEventVerbose formats an event as a multi-line verbose block.
func formatEventVerbose(ev collector.MonitorEvent, width int) string {
	ts := time.Unix(0, ev.Timestamp*int64(time.Millisecond))
	timeStr := ts.Format("15:04:05")

	icon := eventIcon(ev.Event)
	label := eventLabel(ev.Event)

	// Build header: "22:44:48  👤 USER ─────────────"
	// For tool events, include the tool name in the label
	if ev.Detail != nil && ev.Detail.Tool != "" {
		if ev.Event == collector.EventToolStart {
			label = "TOOL: " + ev.Detail.Tool
		} else if ev.Event == collector.EventToolResult || ev.Event == collector.EventToolEnd {
			label = "RESULT: " + ev.Detail.Tool
		}
	}

	headerPrefix := fmt.Sprintf("%s  %s %s ", timeStr, icon, label)
	// Fill with ─ to width
	fillLen := width - len(headerPrefix)
	if fillLen < 3 {
		fillLen = 3
	}
	header := headerPrefix + strings.Repeat("─", fillLen)

	content := verboseContent(ev)
	if content == "" {
		return header
	}

	// Word-wrap and indent content
	contentWidth := width - 2 // 2 spaces indent
	if contentWidth < 20 {
		contentWidth = 20
	}
	wrapped := wordWrap(content, contentWidth)
	lines := strings.Split(wrapped, "\n")
	var indented []string
	for _, line := range lines {
		indented = append(indented, "  "+line)
	}

	return header + "\n" + strings.Join(indented, "\n")
}

// verboseContent returns the content to display in verbose mode per spec.
func verboseContent(ev collector.MonitorEvent) string {
	if ev.Detail == nil {
		return ""
	}
	d := ev.Detail
	switch ev.Event {
	case collector.EventUserMessage, collector.EventAssistantMessage:
		return coalesce(d.Content, d.Message)
	case collector.EventToolStart:
		return joinNonEmpty("\n", targetLine(d.Target), d.ToolInput)
	case collector.EventToolResult, collector.EventToolEnd:
		return coalesce(d.ToolOutput, d.Message)
	default:
		return d.Message
	}
}

// wordWrap wraps text at word boundaries to fit within width.
func wordWrap(text string, width int) string {
	if width <= 0 {
		width = 1
	}
	if text == "" {
		return ""
	}

	// Split on existing newlines first
	inputLines := strings.Split(text, "\n")
	var outputLines []string

	for _, inputLine := range inputLines {
		if len(inputLine) <= width {
			outputLines = append(outputLines, inputLine)
			continue
		}
		words := strings.Fields(inputLine)
		if len(words) == 0 {
			outputLines = append(outputLines, "")
			continue
		}

		var currentLine strings.Builder
		for _, word := range words {
			if currentLine.Len() == 0 {
				currentLine.WriteString(word)
			} else if currentLine.Len()+1+len(word) <= width {
				currentLine.WriteString(" ")
				currentLine.WriteString(word)
			} else {
				outputLines = append(outputLines, currentLine.String())
				currentLine.Reset()
				currentLine.WriteString(word)
			}
		}
		if currentLine.Len() > 0 {
			outputLines = append(outputLines, currentLine.String())
		}
	}

	return strings.Join(outputLines, "\n")
}

// --- helpers ---

func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func joinNonEmpty(sep string, parts ...string) string {
	var nonEmpty []string
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	return strings.Join(nonEmpty, sep)
}

func targetLine(target string) string {
	if target == "" {
		return ""
	}
	return "Target: " + target
}
