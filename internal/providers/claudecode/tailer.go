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
		event = &collector.MonitorEvent{
			Event:  collector.EventMessage,
			Status: collector.StatusThinking,
		}
	case "assistant":
		event = &collector.MonitorEvent{
			Event:  collector.EventMessage,
			Status: collector.StatusIdle,
		}
	case "tool_use":
		event = &collector.MonitorEvent{
			Event:  collector.EventToolStart,
			Status: collector.StatusToolRunning,
			Detail: &collector.EventDetail{
				Tool: entry.Name,
			},
		}
	case "tool_result":
		event = &collector.MonitorEvent{
			Event:  collector.EventToolEnd,
			Status: collector.StatusThinking,
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
