package cli

import (
	"encoding/json"

	"github.com/c3po-protocol1/holocron/internal/collector"
)

// StatusJSON is the JSON output format for a session.
type StatusJSON struct {
	Source        string              `json:"source"`
	SessionID     string             `json:"sessionId"`
	Status        string             `json:"status"`
	Workspace     string             `json:"workspace,omitempty"`
	ElapsedMs     int64              `json:"elapsedMs"`
	CurrentTool   string             `json:"currentTool,omitempty"`
	CurrentTarget string             `json:"currentTarget,omitempty"`
	TokenUsage    *collector.TokenUsage `json:"tokenUsage,omitempty"`
}

// FormatStatusJSON formats sessions as a JSON array.
func FormatStatusJSON(sessions []collector.SessionState) (string, error) {
	if sessions == nil {
		sessions = []collector.SessionState{}
	}

	items := make([]StatusJSON, len(sessions))
	for i, s := range sessions {
		items[i] = StatusJSON{
			Source:        s.Source,
			SessionID:     s.SessionID,
			Status:        string(s.Status),
			Workspace:     s.Workspace,
			ElapsedMs:     s.ElapsedMs,
			CurrentTool:   s.CurrentTool,
			CurrentTarget: s.CurrentTarget,
			TokenUsage:    s.TokenUsage,
		}
	}

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data) + "\n", nil
}
