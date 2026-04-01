package claudecode

import (
	"encoding/json"
	"time"

	"github.com/c3po-protocol1/holocron/internal/collector"
	"github.com/google/uuid"
)

const sourceName = "claude-code"

// RawJSONLEntry represents a single line from a Claude Code JSONL file.
type RawJSONLEntry struct {
	Type      string          `json:"type"`
	Message   json.RawMessage `json:"message,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	CWD       string          `json:"cwd,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
	Version   string          `json:"version,omitempty"`
	GitBranch string          `json:"gitBranch,omitempty"`
	Timestamp string          `json:"timestamp,omitempty"`
}

// ParseJSONLLine parses a single JSONL line and returns a MonitorEvent if applicable.
// Returns nil for unrecognized or non-event types (e.g. queue-operation).
func ParseJSONLLine(line []byte, sessionID, workspace string) *collector.MonitorEvent {
	var entry RawJSONLEntry
	if err := json.Unmarshal(line, &entry); err != nil {
		return nil
	}

	// Use entry's sessionID if available, fall back to provided one
	sid := entry.SessionID
	if sid == "" {
		sid = sessionID
	}

	ws := workspace
	if entry.CWD != "" {
		ws = entry.CWD
	}

	var event *collector.MonitorEvent

	switch entry.Type {
	case "user":
		content := extractUserContent(entry.Message)
		if content == "" {
			return nil
		}
		event = &collector.MonitorEvent{
			Event:  collector.EventUserMessage,
			Status: collector.StatusThinking,
			Detail: &collector.EventDetail{
				Content: truncateContent(content, maxContentSize),
				Message: truncateContent(content, maxMessageSize),
				Role:    "user",
			},
		}
	case "assistant":
		content := extractAssistantContent(entry.Message)
		detail := &collector.EventDetail{
			Role: "assistant",
		}
		if content != "" {
			detail.Content = truncateContent(content, maxContentSize)
			detail.Message = truncateContent(content, maxMessageSize)
		}
		detail.TokenUsage = extractAssistantTokenUsage(entry.Message)
		event = &collector.MonitorEvent{
			Event:  collector.EventAssistantMessage,
			Status: collector.StatusIdle,
			Detail: detail,
		}
	case "tool_use":
		toolInput := extractToolInput(entry.Input)
		target := extractToolTarget(entry.Name, entry.Input)
		event = &collector.MonitorEvent{
			Event:  collector.EventToolStart,
			Status: collector.StatusToolRunning,
			Detail: &collector.EventDetail{
				Tool:      entry.Name,
				Target:    target,
				ToolInput: truncateContent(toolInput, maxContentSize),
			},
		}
	case "tool_result":
		output := extractToolResultContent(entry.Content)
		detail := &collector.EventDetail{
			Role: "tool",
		}
		if output != "" {
			detail.ToolOutput = truncateContent(output, maxContentSize)
			detail.Message = truncateContent(output, maxMessageSize)
		}
		event = &collector.MonitorEvent{
			Event:  collector.EventToolResult,
			Status: collector.StatusThinking,
			Detail: detail,
		}
	default:
		return nil
	}

	event.ID = uuid.New().String()
	event.Source = sourceName
	event.SessionID = sid
	event.Workspace = ws
	event.Timestamp = time.Now().UnixMilli()

	// Extract labels from metadata
	labels := map[string]string{"channel": "local"}
	if entry.GitBranch != "" {
		labels["git_branch"] = entry.GitBranch
	}
	if entry.Version != "" {
		labels["claude_version"] = entry.Version
	}
	event.Labels = labels

	return event
}
