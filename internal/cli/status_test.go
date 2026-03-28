package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/c3po-protocol1/holocron/internal/collector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSessions() []collector.SessionState {
	now := time.Now()
	return []collector.SessionState{
		{
			Source:        "claude-code",
			SessionID:     "a8837a23abcdef",
			Workspace:     "~/Projects/my-app",
			Status:        collector.StatusToolRunning,
			StartedAt:     now.Add(-2*time.Minute - 13*time.Second).UnixMilli(),
			LastEventAt:   now.UnixMilli(),
			ElapsedMs:     133000,
			CurrentTool:   "Edit",
			CurrentTarget: "src/index.ts",
			EventCount:    12,
		},
		{
			Source:      "claude-code",
			SessionID:   "323ac29babcdef",
			Workspace:   "~/Projects/agent-monitor",
			Status:      collector.StatusIdle,
			StartedAt:   now.Add(-15*time.Minute - 2*time.Second).UnixMilli(),
			LastEventAt: now.Add(-10 * time.Minute).UnixMilli(),
			ElapsedMs:   902000,
			EventCount:  5,
		},
		{
			Source:    "openclaw",
			SessionID: "r2d2session",
			Status:    collector.StatusThinking,
			StartedAt: now.Add(-5*time.Minute - 30*time.Second).UnixMilli(),
			ElapsedMs: 330000,
			TokenUsage: &collector.TokenUsage{
				Input:  12500,
				Output: 3000,
			},
			EventCount: 8,
		},
	}
}

// --- Human-readable formatter tests ---

func TestFormatStatus_EmptyState(t *testing.T) {
	output := FormatStatus(nil, time.Now())
	assert.Contains(t, output, "No sessions found.")
}

func TestFormatStatus_Header(t *testing.T) {
	sessions := testSessions()
	output := FormatStatus(sessions, time.Now())
	assert.Contains(t, output, "3 sessions")
}

func TestFormatStatus_ActiveSession(t *testing.T) {
	sessions := testSessions()
	output := FormatStatus(sessions, time.Now())
	// Active session should show source, truncated ID, status indicator
	assert.Contains(t, output, "claude-code")
	assert.Contains(t, output, "a8837a23ab")
	// Should show tool activity
	assert.Contains(t, output, "Edit")
	assert.Contains(t, output, "src/index.ts")
}

func TestFormatStatus_IdleSession(t *testing.T) {
	sessions := testSessions()
	output := FormatStatus(sessions, time.Now())
	assert.Contains(t, output, "323ac29bab")
	assert.Contains(t, output, "idle")
}

func TestFormatStatus_Workspace(t *testing.T) {
	sessions := testSessions()
	output := FormatStatus(sessions, time.Now())
	assert.Contains(t, output, "~/Projects/my-app")
	assert.Contains(t, output, "~/Projects/agent-monitor")
}

func TestFormatStatus_Elapsed(t *testing.T) {
	sessions := testSessions()
	output := FormatStatus(sessions, time.Now())
	// Should contain elapsed time strings
	assert.Contains(t, output, "2m")
	assert.Contains(t, output, "15m")
}

// --- JSON formatter tests ---

func TestFormatStatusJSON_EmptyState(t *testing.T) {
	output, err := FormatStatusJSON(nil)
	require.NoError(t, err)
	assert.Equal(t, "[]\n", output)

	// Should be valid JSON
	var result []interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestFormatStatusJSON_ValidJSON(t *testing.T) {
	sessions := testSessions()
	output, err := FormatStatusJSON(sessions)
	require.NoError(t, err)

	var result []StatusJSON
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestFormatStatusJSON_Fields(t *testing.T) {
	sessions := testSessions()
	output, err := FormatStatusJSON(sessions)
	require.NoError(t, err)

	var result []StatusJSON
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)

	first := result[0]
	assert.Equal(t, "claude-code", first.Source)
	assert.Equal(t, "a8837a23abcdef", first.SessionID)
	assert.Equal(t, "tool_running", first.Status)
	assert.Equal(t, "~/Projects/my-app", first.Workspace)
	assert.Equal(t, int64(133000), first.ElapsedMs)
	assert.Equal(t, "Edit", first.CurrentTool)
	assert.Equal(t, "src/index.ts", first.CurrentTarget)
}

func TestFormatStatusJSON_OmitsEmptyFields(t *testing.T) {
	sessions := []collector.SessionState{
		{
			Source:    "claude-code",
			SessionID: "abc123",
			Status:    collector.StatusIdle,
			ElapsedMs: 5000,
		},
	}
	output, err := FormatStatusJSON(sessions)
	require.NoError(t, err)

	// Should not contain workspace, currentTool, currentTarget when empty
	assert.NotContains(t, output, "workspace")
	assert.NotContains(t, output, "currentTool")
	assert.NotContains(t, output, "currentTarget")
}

// --- Filter tests ---

func TestFilterActive(t *testing.T) {
	sessions := testSessions()
	filtered := FilterSessions(sessions, true, "")
	assert.Len(t, filtered, 2) // tool_running and thinking are active

	for _, s := range filtered {
		assert.NotEqual(t, collector.StatusIdle, s.Status)
		assert.NotEqual(t, collector.StatusDone, s.Status)
	}
}

func TestFilterBySource(t *testing.T) {
	sessions := testSessions()
	filtered := FilterSessions(sessions, false, "claude-code")
	assert.Len(t, filtered, 2)

	for _, s := range filtered {
		assert.Equal(t, "claude-code", s.Source)
	}
}

func TestFilterActiveAndSource(t *testing.T) {
	sessions := testSessions()
	filtered := FilterSessions(sessions, true, "claude-code")
	assert.Len(t, filtered, 1)
	assert.Equal(t, "a8837a23abcdef", filtered[0].SessionID)
}

func TestFilterNoMatch(t *testing.T) {
	sessions := testSessions()
	filtered := FilterSessions(sessions, false, "nonexistent")
	assert.Empty(t, filtered)
}

func TestFilterNoop(t *testing.T) {
	sessions := testSessions()
	filtered := FilterSessions(sessions, false, "")
	assert.Len(t, filtered, 3)
}

// --- Edge cases ---

func TestFormatStatus_SingleSession(t *testing.T) {
	sessions := []collector.SessionState{
		{
			Source:    "claude-code",
			SessionID: "short",
			Status:    collector.StatusDone,
			Workspace: "~/Projects/test",
			StartedAt: time.Now().Add(-30 * time.Second).UnixMilli(),
			ElapsedMs: 30000,
		},
	}
	output := FormatStatus(sessions, time.Now())
	assert.Contains(t, output, "1 session")
	assert.NotContains(t, output, "1 sessions") // singular
	assert.Contains(t, output, "short") // short ID not truncated
}

func TestFormatStatus_TokenUsage(t *testing.T) {
	sessions := []collector.SessionState{
		{
			Source:    "openclaw",
			SessionID: "r2d2",
			Status:    collector.StatusThinking,
			TokenUsage: &collector.TokenUsage{
				Input:  12500,
				Output: 3000,
			},
		},
	}
	output := FormatStatus(sessions, time.Now())
	assert.Contains(t, output, "tokens:")
}

func TestFormatStatusJSON_TokenUsage(t *testing.T) {
	sessions := []collector.SessionState{
		{
			Source:    "openclaw",
			SessionID: "r2d2",
			Status:    collector.StatusThinking,
			TokenUsage: &collector.TokenUsage{
				Input:  12500,
				Output: 3000,
			},
		},
	}
	output, err := FormatStatusJSON(sessions)
	require.NoError(t, err)

	// Token usage should be included in JSON
	var result []StatusJSON
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)
	assert.NotNil(t, result[0].TokenUsage)
	assert.Equal(t, int64(12500), result[0].TokenUsage.Input)
}

// Ensure FormatStatus output lines don't have excessive trailing whitespace
func TestFormatStatus_CleanOutput(t *testing.T) {
	sessions := testSessions()
	output := FormatStatus(sessions, time.Now())
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Lines shouldn't have trailing whitespace beyond 2 chars
		trimmed := strings.TrimRight(line, " ")
		if len(line)-len(trimmed) > 5 {
			t.Errorf("excessive trailing whitespace on line: %q", line)
		}
	}
}
